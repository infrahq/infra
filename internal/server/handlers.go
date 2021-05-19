package server

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

func parsetoken(db *bolt.DB, req *http.Request) (user *User, token *Token, err error) {
	sk, _, _ := req.BasicAuth()
	authorization := req.Header.Get("Authorization")
	if authorization != "" {
		sk = strings.Replace(authorization, "Bearer ", "", -1)
	}

	if len(sk) != len("sk_")+IDLength+SecretKeyLength {
		return nil, nil, errors.New("invalid token")
	}

	sk = strings.Replace(sk, "sk_", "", -1)

	id := sk[0:IDLength]

	err = db.View(func(tx *bolt.Tx) error {
		token, err := GetToken(tx, "tk_"+id, true)
		if err != nil {
			return err
		}

		user, err = GetUser(tx, token.User)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	secret := sk[IDLength : IDLength+SecretKeyLength]
	if err = bcrypt.CompareHashAndPassword(token.HashedSecret, []byte(secret)); err != nil {
		return nil, nil, err
	}

	if time.Now().After(time.Unix(token.Expires, 0)) {
		return nil, nil, errors.New("expired token")
	}
	return
}

func TokenAuthMiddleware(db *bolt.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("skipauth") {
			c.Next()
			return
		}

		user, token, err := parsetoken(db, c.Request)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Set("user", user.ID)
		c.Set("email", user.Email)
		c.Set("token", token.ID)
		c.Next()
	}
}

func PermissionMiddleware(permission string, cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("skipauth") {
			c.Next()
			return
		}

		email := c.GetString("email")
		if email == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		p := PermissionForEmail(email, cfg)
		if !IsEqualOrHigherPermission(p, permission) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Set("permission", p)
		c.Next()
	}
}

func ProxyHandler(kubernetes *Kubernetes) gin.HandlerFunc {
	remote, err := url.Parse(kubernetes.Config.Host)
	if err != nil {
		fmt.Println(err)
	}

	ca, err := ioutil.ReadFile(kubernetes.Config.TLSClientConfig.CAFile)
	if err != nil {
		fmt.Println(err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(ca)
	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}

	return func(c *gin.Context) {
		email := c.GetString("email")
		c.Request.Header.Del("Authorization")
		c.Request.Header.Set("Impersonate-User", email)
		c.Request.Header.Add("Authorization", "Bearer "+string(kubernetes.Config.BearerToken))
		http.StripPrefix("/v1/proxy", proxy).ServeHTTP(c.Writer, c.Request)
	}
}

type RetrieveProvidersResponse struct {
	ProviderConfig
}

type CreateTokenResponse struct {
	Token
	SecretToken string `json:"secret_token"`
}

type ListTokensResponseData struct {
	Token
	User User `json:"user"`
}

type ListTokensResponse struct {
	Data []ListTokensResponseData `json:"data"`
}

type CreateUserResponse struct {
	User
}

type ListUsersResponseData struct {
	User
	Permission string `json:"permission"`
}

type ListUsersResponse struct {
	Data []ListUsersResponseData `json:"data"`
}

type DeleteResponse struct {
	Deleted bool   `json:"deleted"`
	ID      string `json:"id"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func oktaToken(db *bolt.DB, cfg *Config, code string) (token *Token, sk string, err error) {
	email, err := cfg.Providers.Okta.EmailFromCode(code)
	if err != nil {
		return nil, "", err
	}

	err = db.Update(func(tx *bolt.Tx) (err error) {
		user, err := FindUser(tx, email)
		if err != nil {
			return err
		}
		uid := user.ID
		token := &Token{
			User: uid,
		}

		sk, err = PutToken(tx, token)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}

	token.HashedSecret = []byte{}
	return token, sk, nil
}

func addRoutes(router *gin.Engine, db *bolt.DB, kube *Kubernetes, cfg *Config) error {
	router.GET("/v1/providers", func(c *gin.Context) {
		c.JSON(http.StatusOK, RetrieveProvidersResponse{cfg.Providers})
	})

	router.POST("/v1/tokens", func(c *gin.Context) {
		type Params struct {
			OktaCode string `form:"okta-code"`
			User     string `form:"user"`
		}

		var params Params
		if err := c.ShouldBind(&params); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		var err error
		if params.OktaCode != "" && cfg.Providers.Okta.Valid() {
			token, sk, err := oktaToken(db, cfg, params.OktaCode)
			if err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{"invalid code"})
				return
			}
			c.JSON(http.StatusCreated, CreateTokenResponse{*token, sk})
			return
		}

		curuser, curtoken, _ := parsetoken(db, c.Request)

		// token for oneself
		if params.User == "" {
			if curuser == nil && curtoken == nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{"invalid user"})
				return
			}
			var token Token
			var sk string
			err := db.Update(func(tx *bolt.Tx) error {
				token.User = curtoken.ID
				sk, err = PutToken(tx, &token)
				if err != nil {
					return err
				}
				if err = DeleteToken(tx, curtoken.ID); err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
				return
			}

			token.HashedSecret = []byte{}

			c.JSON(http.StatusCreated, CreateTokenResponse{token, sk})
			return
		}

		if !c.GetBool("skipauth") && curuser == nil && IsEqualOrHigherPermission(PermissionForEmail(curuser.Email, cfg), "admin") {
			c.JSON(http.StatusUnauthorized, ErrorResponse{"unauthorized"})
			return
		}

		var token Token
		var sk string
		var uid string

		token.User = uid
		err = db.Update(func(tx *bolt.Tx) (err error) {
			sk, err = PutToken(tx, &token)
			if err != nil {
				return err
			}

			token.HashedSecret = []byte{}

			return nil
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		c.JSON(http.StatusCreated, CreateTokenResponse{token, sk})
	})

	router.GET("/v1/tokens", TokenAuthMiddleware(db), PermissionMiddleware("admin", cfg), func(c *gin.Context) {
		data := make([]ListTokensResponseData, 0)

		err := db.View(func(tx *bolt.Tx) (err error) {
			var tokens []Token
			tokens, err = ListTokens(tx, "")
			if err != nil {
				return err
			}

			for _, t := range tokens {
				user, err := GetUser(tx, t.User)
				if err != nil {
					return err
				}

				data = append(data, ListTokensResponseData{
					t,
					*user,
				})
			}

			sort.Slice(data, func(i, j int) bool {
				return data[i].Created > data[j].Created
			})

			return err
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		c.JSON(http.StatusOK, ListTokensResponse{data})
	})

	router.DELETE("/v1/tokens/:id", TokenAuthMiddleware(db), PermissionMiddleware("admin", cfg), func(c *gin.Context) {
		type binds struct {
			ID string `uri:"id" binding:"required"`
		}

		var params binds
		if err := c.BindUri(&params); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		err := db.Update(func(tx *bolt.Tx) error {
			return DeleteToken(tx, params.ID)
		})

		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		c.JSON(http.StatusOK, DeleteResponse{true, params.ID})
	})

	router.GET("/v1/users", TokenAuthMiddleware(db), PermissionMiddleware("view", cfg), func(c *gin.Context) {
		var users []User
		err := db.View(func(tx *bolt.Tx) (err error) {
			users, err = ListUsers(tx)
			return err
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		sort.Slice(users, func(i, j int) bool {
			return users[i].Created > users[j].Created
		})

		var data []ListUsersResponseData
		for _, u := range users {
			data = append(data, ListUsersResponseData{u, PermissionForEmail(u.Email, cfg)})
		}

		c.JSON(http.StatusOK, ListUsersResponse{data})
	})

	router.POST("/v1/users", TokenAuthMiddleware(db), PermissionMiddleware("admin", cfg), func(c *gin.Context) {
		type binds struct {
			Email string `form:"email" binding:"required"`
		}

		var form binds
		if err := c.Bind(&form); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		user := &User{
			Email:     form.Email,
			Providers: []string{},
		}

		err := db.Update(func(tx *bolt.Tx) (err error) {
			return PutUser(tx, user)
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		if err := kube.UpdatePermissions(db, cfg); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		c.JSON(http.StatusCreated, CreateUserResponse{*user})
	})

	router.DELETE("/v1/users/:id", TokenAuthMiddleware(db), PermissionMiddleware("admin", cfg), func(c *gin.Context) {
		type binds struct {
			ID string `uri:"id" binding:"required"`
		}

		var params binds
		if err := c.BindUri(&params); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		err := db.Update(func(tx *bolt.Tx) (err error) {
			user, err := GetUser(tx, params.ID)
			if err != nil {
				return errors.New("user does not exist")
			}

			if len(user.Providers) > 0 {
				return errors.New("user managed by external providers")
			}

			return DeleteUser(tx, params.ID)
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		if err := kube.UpdatePermissions(db, cfg); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		c.JSON(http.StatusOK, DeleteResponse{true, params.ID})
	})

	if kube != nil {
		proxyHandler := ProxyHandler(kube)
		router.GET("/v1/proxy/*all", TokenAuthMiddleware(db), proxyHandler)
		router.POST("/v1/proxy/*all", TokenAuthMiddleware(db), proxyHandler)
		router.PUT("/v1/proxy/*all", TokenAuthMiddleware(db), proxyHandler)
		router.PATCH("/v1/proxy/*all", TokenAuthMiddleware(db), proxyHandler)
		router.DELETE("/v1/proxy/*all", TokenAuthMiddleware(db), proxyHandler)
	}

	return nil
}

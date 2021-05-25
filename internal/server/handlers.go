package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gorm.io/gorm"
)

// TODO (jmorganca): put this in a secret – eventually use certificates instead
var key = []byte("secret")

func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("skipauth") {
			c.Next()
			return
		}

		fmt.Println(c.Request.BasicAuth())
		fmt.Println(c.Request.Header.Get("Authorization"))

		authorization := c.Request.Header.Get("Authorization")
		raw := strings.Replace(authorization, "Bearer ", "", -1)

		tok, err := jwt.ParseSigned(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		out := make(map[string]interface{})
		if err := tok.Claims(key, &out); err != nil {
			fmt.Println(err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		email := out["email"].(string)

		c.Set("email", email)
		c.Next()
	}
}

func createToken(email string) (string, error) {
	key := []byte("secret")
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return "", err
	}

	// TODO (jmorganca): create refresh tokens

	cl := jwt.Claims{
		Subject:  "subject", // TODO (jmorganca): make this the user ID
		Issuer:   "infra",
		Expiry:   jwt.NewNumericDate(time.Now().Add(time.Hour * 1)),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}
	custom := struct {
		Email string `json:"email"`
	}{
		email,
	}

	raw, err := jwt.Signed(signer).Claims(cl).Claims(custom).CompactSerialize()
	if err != nil {
		return "", err
	}

	return raw, nil
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
			fmt.Println("user has permission " + p + " required: " + permission)
			return
		}

		c.Set("permission", p)
		c.Next()
	}
}

func ProxyHandler(kubernetes *Kubernetes) (handler gin.HandlerFunc, err error) {
	remote, err := url.Parse(kubernetes.Config.Host)
	if err != nil {
		return
	}

	ca, err := ioutil.ReadFile(kubernetes.Config.TLSClientConfig.CAFile)
	if err != nil {
		return
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
	}, err
}

type RetrieveCAResponse struct {
	CA string `json:"ca"`
}

type RetrieveProvidersResponse struct {
	ProviderConfig
}

type CreateTokenResponse struct {
	Token string
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

func addRoutes(router *gin.Engine, db *gorm.DB, kube *Kubernetes, cfg *Config) error {
	router.GET("/v1/providers", func(c *gin.Context) {
		c.JSON(http.StatusOK, RetrieveProvidersResponse{cfg.Providers})
	})

	router.POST("/v1/tokens", func(c *gin.Context) {
		type Params struct {
			// Via Okta
			OktaCode string `form:"okta-code"`

			// Via email + password
			Email    string `form:"email" validate:"email"`
			Password string `form:"password"`
		}

		var params Params
		if err := c.ShouldBind(&params); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		var err error
		if params.OktaCode != "" && cfg.Providers.Okta.Valid() {
			email, err := cfg.Providers.Okta.EmailFromCode(params.OktaCode)
			if err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{"invalid code"})
				return
			}

			token, err := createToken(email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, ErrorResponse{"failed to create token"})
				return
			}

			c.JSON(http.StatusCreated, CreateTokenResponse{token})
			return
		}

		if !c.GetBool("skipauth") && !IsEqualOrHigherPermission(PermissionForEmail(c.GetString("email"), cfg), "admin") {
			c.JSON(http.StatusUnauthorized, ErrorResponse{"unauthorized"})
			return
		}

		if params.Email == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{"email cannot be empty"})
			return
		}

		token, err := createToken(params.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{"could not create token"})
			return
		}

		c.JSON(http.StatusCreated, CreateTokenResponse{token})
	})

	router.GET("/v1/users", TokenAuthMiddleware(), PermissionMiddleware("view", cfg), func(c *gin.Context) {
		var users []User
		result := db.Find(&users)

		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{"could not list users"})
			return
		}

		data := make([]ListUsersResponseData, 0)
		for _, u := range users {
			data = append(data, ListUsersResponseData{u, PermissionForEmail(u.Email, cfg)})
		}

		c.JSON(http.StatusOK, ListUsersResponse{data})
	})

	router.POST("/v1/users", TokenAuthMiddleware(), PermissionMiddleware("admin", cfg), func(c *gin.Context) {
		type binds struct {
			Email    string `form:"email" binding:"email,required"`
			Password string `form:"password" binding:"required"`
		}

		var form binds
		if err := c.Bind(&form); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(form.Password), bcrypt.DefaultCost)
		if err != nil {
			return
		}

		user := &User{
			Email:    form.Email,
			Password: hashedPassword,
			Provider: "infra",
		}

		result := db.Create(&user)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{"could not create user"})
			return
		}

		if err := kube.UpdatePermissions(db, cfg); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		c.JSON(http.StatusCreated, CreateUserResponse{*user})
	})

	router.DELETE("/v1/users/:id", TokenAuthMiddleware(), PermissionMiddleware("admin", cfg), func(c *gin.Context) {
		type binds struct {
			Email string `uri:"id" binding:"required"`
		}

		var params binds
		if err := c.BindUri(&params); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		result := db.Where("email = ?", params.Email).Delete(User{})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{"could not delete user"})
			return
		}

		if result.RowsAffected == 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{"user does not exist"})
			return
		}

		if err := kube.UpdatePermissions(db, cfg); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		c.JSON(http.StatusOK, DeleteResponse{true, params.Email})
	})

	if kube != nil && kube.Config != nil {
		proxyHandler, err := ProxyHandler(kube)
		if err != nil {
			return err
		}
		router.GET("/v1/proxy/*all", TokenAuthMiddleware(), proxyHandler)
		router.POST("/v1/proxy/*all", TokenAuthMiddleware(), proxyHandler)
		router.PUT("/v1/proxy/*all", TokenAuthMiddleware(), proxyHandler)
		router.PATCH("/v1/proxy/*all", TokenAuthMiddleware(), proxyHandler)
		router.DELETE("/v1/proxy/*all", TokenAuthMiddleware(), proxyHandler)
	}

	return nil
}

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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gorm.io/gorm"
)

func TokenAuthMiddleware(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("skipauth") {
			c.Next()
			return
		}

		// Check bearer header
		authorization := c.Request.Header.Get("Authorization")
		raw := strings.Replace(authorization, "Bearer ", "", -1)

		// Check cookie
		if raw == "" {
			raw, _ = c.Cookie("token")
		}

		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		tok, err := jwt.ParseSigned(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		cl := jwt.Claims{}
		out := make(map[string]interface{})
		if err := tok.Claims(secret, &cl, &out); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		err = cl.Validate(jwt.Expected{
			Issuer: "infra",
			Time:   time.Now(),
		})
		switch {
		case errors.Is(err, jwt.ErrExpired):
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "expired"})
			return
		case err != nil:
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		email := out["email"].(string)

		c.Set("email", email)
		c.Next()
	}
}

func createToken(email string, secret []byte) (string, error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: secret}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return "", err
	}

	cl := jwt.Claims{
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

type RetrieveProvidersResponse struct {
	Provider
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
	Deleted bool `json:"deleted"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func addRoutes(router *gin.Engine, db *gorm.DB, kube *Kubernetes, cfg *Config, settings *Settings) error {
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

		// Via Okta
		// TODO (jmorganca): support web-based login for okta
		var err error
		if params.OktaCode != "" && cfg.Providers.Okta.Valid() {
			email, err := cfg.Providers.Okta.EmailFromCode(params.OktaCode)
			if err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{"invalid code"})
				return
			}

			token, err := createToken(email, settings.TokenSecret)
			if err != nil {
				c.JSON(http.StatusInternalServerError, ErrorResponse{"failed to create token"})
				fmt.Println(err)
				return
			}

			c.SetSameSite(http.SameSiteStrictMode)
			c.SetCookie("token", token, 3600, "/", c.Request.Host, false, true)
			c.SetCookie("login", "1", 3600, "/", c.Request.Host, false, false)

			c.JSON(http.StatusCreated, CreateTokenResponse{token})
			return
		}

		// Via email + password
		type EmailParams struct {
			Email    string `form:"email" validate:"required,email"`
			Password string `form:"password" validate:"required,email"`
		}

		var emailParams EmailParams
		if err := c.ShouldBind(&emailParams); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			fmt.Println(err)
			return
		}

		var user User
		if err := db.Where("email = ?", emailParams.Email).First(&user).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unauthorized"})
			return
		}

		if err = bcrypt.CompareHashAndPassword(user.Password, []byte(emailParams.Password)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unauthorized"})
			return
		}

		token, err := createToken(params.Email, settings.TokenSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{"could not create token"})
			fmt.Println(err)
			return
		}

		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie("token", token, 3600, "/", c.Request.Host, false, true)
		c.SetCookie("login", "1", 3600, "/", c.Request.Host, false, false)
		c.JSON(http.StatusCreated, CreateTokenResponse{token})
	})

	router.GET("/v1/users", TokenAuthMiddleware(settings.TokenSecret), PermissionMiddleware("view", cfg), func(c *gin.Context) {
		var users []User
		err := db.Find(&users).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{"could not list users"})
			return
		}

		data := make([]ListUsersResponseData, 0)
		for _, u := range users {
			data = append(data, ListUsersResponseData{u, PermissionForEmail(u.Email, cfg)})
		}

		c.JSON(http.StatusOK, ListUsersResponse{data})
	})

	router.POST("/v1/users", TokenAuthMiddleware(settings.TokenSecret), func(c *gin.Context) {
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
			c.JSON(http.StatusInternalServerError, ErrorResponse{"could not create user"})
			return
		}

		var user User
		user.Email = form.Email
		user.Password = hashedPassword
		user.Provider = "infra"

		err = db.Create(&user).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{"could not create user"})
			return
		}

		if err := kube.UpdatePermissions(db, cfg); err != nil {
			fmt.Println("could not update kubernetes permissions: ", err)
		}

		c.JSON(http.StatusCreated, CreateUserResponse{user})
	})

	router.DELETE("/v1/users/:id", TokenAuthMiddleware(settings.TokenSecret), PermissionMiddleware("admin", cfg), func(c *gin.Context) {
		type binds struct {
			Email string `uri:"id" binding:"required"`
		}

		var params binds
		if err := c.BindUri(&params); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		if c.GetString("email") == params.Email {
			c.JSON(http.StatusBadRequest, ErrorResponse{"can not delete yourself"})
			return
		}

		err := db.Where("email = ?", params.Email).Delete(User{}).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, ErrorResponse{"user does not exist"})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{"could not delete user"})
			return
		}

		if err := kube.UpdatePermissions(db, cfg); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
			return
		}

		c.JSON(http.StatusOK, DeleteResponse{true})
	})

	if kube != nil && kube.Config != nil {
		proxyHandler, err := ProxyHandler(kube)
		if err != nil {
			return err
		}
		router.GET("/v1/proxy/*all", TokenAuthMiddleware(settings.TokenSecret), proxyHandler)
		router.POST("/v1/proxy/*all", TokenAuthMiddleware(settings.TokenSecret), proxyHandler)
		router.PUT("/v1/proxy/*all", TokenAuthMiddleware(settings.TokenSecret), proxyHandler)
		router.PATCH("/v1/proxy/*all", TokenAuthMiddleware(settings.TokenSecret), proxyHandler)
		router.DELETE("/v1/proxy/*all", TokenAuthMiddleware(settings.TokenSecret), proxyHandler)
	}

	return nil
}

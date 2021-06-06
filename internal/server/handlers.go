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
	"github.com/infrahq/infra/internal/okta"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gorm.io/gorm"
)

type Handlers struct {
	db         *gorm.DB
	kubernetes *Kubernetes
}

type DeleteResponse struct {
	Deleted bool `json:"deleted"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *Handlers) TokenAuthMiddleware() gin.HandlerFunc {
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

		var settings Settings
		err = h.db.First(&settings).Error
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if err := tok.Claims([]byte(settings.TokenSecret), &cl, &out); err != nil {
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

func (h *Handlers) createToken(email string) (string, error) {
	var settings Settings
	err := h.db.First(&settings).Error
	if err != nil {
		return "", err
	}

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: []byte(settings.TokenSecret)}, (&jose.SignerOptions{}).WithType("JWT"))
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

func (h *Handlers) RoleMiddleware(roles ...string) gin.HandlerFunc {
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

		var u User
		err := h.db.Preload("Permissions").Where("email = ?", email).First(&u).Error
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "error"})
			return
		}

		for _, p := range u.Permissions {
			for _, allowed := range roles {
				if p.RoleName == allowed {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
}

func (h *Handlers) ProxyHandler() (handler gin.HandlerFunc, err error) {
	remote, err := url.Parse(h.kubernetes.Config.Host)
	if err != nil {
		return
	}

	ca, err := ioutil.ReadFile(h.kubernetes.Config.TLSClientConfig.CAFile)
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
		c.Request.Header.Add("Authorization", "Bearer "+string(h.kubernetes.Config.BearerToken))
		http.StripPrefix("/v1/proxy", proxy).ServeHTTP(c.Writer, c.Request)
	}, err
}

func (h *Handlers) Healthz(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (h *Handlers) ListUsers(c *gin.Context) {
	var users []User
	err := h.db.Preload("Permissions.Role").Preload("Providers").Find(&users).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{"could not list users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": users})
}

func (h *Handlers) CreateUser(c *gin.Context) {
	type binds struct {
		Email    string `form:"email" binding:"email,required"`
		Password string `form:"password" binding:"required"`
	}

	var form binds
	if err := c.Bind(&form); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	var user User
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var infraProvider Provider
		if err := tx.Where(&Provider{Kind: DefaultInfraProviderKind}).First(&infraProvider).Error; err != nil {
			return err
		}

		if tx.Model(&infraProvider).Where(&User{Email: form.Email}).Association("Users").Count() > 0 {
			return errors.New("user with this email already exists")
		}

		err := infraProvider.CreateUser(tx, form.Email, form.Password)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	if err := h.kubernetes.UpdatePermissions(); err != nil {
		fmt.Println("could not update kubernetes permissions: ", err)
	}

	c.JSON(http.StatusCreated, user)
}

func (h *Handlers) DeleteUser(c *gin.Context) {
	type binds struct {
		ID string `uri:"id" binding:"required"`
	}

	var params binds
	if err := c.BindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var self User
		err := tx.First(&self, "email = ?", c.GetString("email")).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if self.ID == params.ID {
			return errors.New("cannot delete self")
		}

		var user User
		err = tx.First(&user, "id = ?", params.ID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user does not exist")
		}

		if err != nil {
			return err
		}

		if tx.Model(&user).Where(&Provider{Kind: DefaultInfraProviderKind}).Association("Providers").Count() == 0 {
			return errors.New("user managed by identity provider")
		}

		var infraProvider Provider
		if err := tx.Where(&Provider{Kind: DefaultInfraProviderKind}).First(&infraProvider).Error; err != nil {
			return err
		}

		err = infraProvider.DeleteUser(tx, &user)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	if err := h.kubernetes.UpdatePermissions(); err != nil {
		fmt.Println("could not update kubernetes permissions: ", err)
	}

	c.JSON(http.StatusOK, DeleteResponse{true})
}

func (h *Handlers) ListProviders(c *gin.Context) {
	var providers []Provider
	if err := h.db.Preload("Users").Find(&providers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": providers})
}

func (h *Handlers) CreateProvider(c *gin.Context) {
	type binds struct {
		Kind string `form:"kind" binding:"required"`
	}

	var params binds
	if err := c.Bind(&params); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	var provider Provider

	err := h.db.Transaction(func(tx *gorm.DB) error {
		switch params.Kind {
		case "infra":
			count := tx.Where(&Provider{Kind: "infra"}).First(&Provider{}).RowsAffected
			if count > 0 {
				return errors.New("can only have one infra provider")
			}

			provider.Kind = "infra"

			err := tx.Where(&provider).FirstOrCreate(&provider).Error
			if err != nil {
				return err
			}

		case "okta":
			type oktaBinds struct {
				OktaApiToken     string `form:"oktaApiToken" binding:"required"`
				OktaDomain       string `form:"oktaDomain" binding:"required,fqdn"`
				OktaClientID     string `form:"oktaClientID" binding:"required"`
				OktaClientSecret string `form:"oktaClientSecret" binding:"required"`
			}

			var oktaParams oktaBinds
			if err := c.Bind(&oktaParams); err != nil {
				return err
			}

			count := tx.Where(&Provider{Kind: "okta", Domain: oktaParams.OktaDomain}).First(&Provider{}).RowsAffected
			if count > 0 {
				return errors.New("okta provider with this domain already exists")
			}

			provider.Kind = "okta"
			provider.ApiToken = oktaParams.OktaApiToken
			provider.Domain = oktaParams.OktaDomain
			provider.ClientID = oktaParams.OktaClientID
			provider.ClientSecret = oktaParams.OktaClientSecret

			result := tx.FirstOrCreate(&provider)
			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	c.JSON(http.StatusCreated, provider)
}

func (h *Handlers) DeleteProvider(c *gin.Context) {
	type binds struct {
		ID string `uri:"id" binding:"required"`
	}

	var params binds
	if err := c.BindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		// Dont allow deleting the infra provider
		var provider Provider
		count := tx.First(&provider, "ID = ?", params.ID).RowsAffected
		if count == 0 {
			return errors.New("no such provider")
		}

		if provider.Kind == "infra" {
			return errors.New("cannot delete infra provider")
		}

		return tx.Delete(&provider).Error
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	c.JSON(http.StatusOK, DeleteResponse{true})
}

func (h *Handlers) CreateToken(c *gin.Context) {
	type Params struct {
		// Via Okta
		OktaDomain string `form:"okta-domain"`
		OktaCode   string `form:"okta-code"`

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
	// TODO (jmorganca): check okta edge cases in client verification
	var err error
	if params.OktaCode != "" {
		var provider Provider
		if err := h.db.Where(&Provider{Kind: "okta", Domain: params.OktaDomain}).First(&provider).Error; err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{"no provider with okta domain"})
			return
		}

		email, err := okta.EmailFromCode(
			params.OktaCode,
			provider.Domain,
			provider.ClientID,
			provider.ClientSecret,
		)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{"invalid code"})
			return
		}

		token, err := h.createToken(email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{"failed to create token"})
			fmt.Println(err)
			return
		}

		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie("token", token, 3600, "/", c.Request.Host, false, true)
		c.SetCookie("login", "1", 3600, "/", c.Request.Host, false, false)

		c.JSON(http.StatusCreated, gin.H{"token": token})
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
	if err := h.db.Where("email = ?", emailParams.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{"unauthorized"})
		return
	}

	if err = bcrypt.CompareHashAndPassword(user.Password, []byte(emailParams.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{"unauthorized"})
		return
	}

	token, err := h.createToken(params.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{"could not create token"})
		fmt.Println(err)
		return
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("token", token, 3600, "/", c.Request.Host, false, true)
	c.SetCookie("login", "1", 3600, "/", c.Request.Host, false, false)
	c.JSON(http.StatusCreated, gin.H{"Token": token})
}

func (h *Handlers) AdminSignup(c *gin.Context) {
	type binds struct {
		Email    string `form:"email" binding:"email,required"`
		Password string `form:"password" binding:"required"`
	}

	var form binds
	if err := c.Bind(&form); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	var settings Settings
	if err := h.db.First(&settings).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not create admin user"})
		return
	}

	if settings.DisableSignup {
		c.JSON(http.StatusBadRequest, ErrorResponse{"admin signup disabled"})
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var exists User
		count := tx.First(&exists).RowsAffected
		if count > 0 {
			return errors.New("users already exist and admin must be the first user")
		}

		var provider Provider
		provider.Kind = DefaultInfraProviderKind
		if err := tx.Where(&provider).First(&provider).Error; err != nil {
			return err
		}

		if err := provider.CreateUser(tx, form.Email, form.Password); err != nil {
			return err
		}

		permission := Permission{UserEmail: form.Email, RoleName: DefaultRoleAdmin}
		return tx.Create(&permission).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{err.Error()})
		return
	}

	if err := h.kubernetes.UpdatePermissions(); err != nil {
		fmt.Println("could not update kubernetes permissions: ", err)
	}

	token, err := h.createToken(form.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{"could not create token"})
		return
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("token", token, 3600, "/", c.Request.Host, false, true)
	c.SetCookie("login", "1", 3600, "/", c.Request.Host, false, false)
	c.Status(http.StatusCreated)
}

func (h *Handlers) addRoutes(router *gin.Engine) error {
	router.GET("/v1/users", h.TokenAuthMiddleware(), h.RoleMiddleware("view", "edit", "admin"), h.ListUsers)
	router.POST("/v1/users", h.TokenAuthMiddleware(), h.RoleMiddleware("edit", "admin"), h.CreateUser)
	router.DELETE("/v1/users/:id", h.TokenAuthMiddleware(), h.RoleMiddleware("edit", "admin"), h.DeleteUser)

	router.GET("/v1/providers", h.ListProviders)
	router.POST("/v1/providers", h.TokenAuthMiddleware(), h.RoleMiddleware("admin"), h.CreateProvider)
	router.DELETE("/v1/providers/:id", h.TokenAuthMiddleware(), h.RoleMiddleware("admin"), h.DeleteProvider)

	router.POST("/v1/tokens", h.CreateToken)
	router.POST("/v1/admin/signup", h.AdminSignup)

	if h.kubernetes != nil && h.kubernetes.Config != nil {
		proxyHandler, err := h.ProxyHandler()
		if err != nil {
			return err
		}
		router.GET("/v1/proxy/*all", h.TokenAuthMiddleware(), proxyHandler)
		router.POST("/v1/proxy/*all", h.TokenAuthMiddleware(), proxyHandler)
		router.PUT("/v1/proxy/*all", h.TokenAuthMiddleware(), proxyHandler)
		router.PATCH("/v1/proxy/*all", h.TokenAuthMiddleware(), proxyHandler)
		router.DELETE("/v1/proxy/*all", h.TokenAuthMiddleware(), proxyHandler)
	}

	return nil
}

package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/okta"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gorm.io/gorm"
)

type Handlers struct {
	db *gorm.DB
}

type DeleteResponse struct {
	Deleted bool `json:"deleted"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *Handlers) TokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("skipauth") {
			c.Next()
			return
		}

		// Check bearer header
		authorization := c.Request.Header.Get("Authorization")
		raw := strings.Replace(authorization, "Bearer ", "", -1)

		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if len(raw) != 36 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := raw[0:12]
		secret := raw[12:36]

		var token Token
		if err := h.db.Preload("User").First(&token, "id = ?", id).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if err := token.CheckSecret(secret); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Set("user", token.User)
		c.Next()
	}
}

func (h *Handlers) RoleMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("skipauth") {
			c.Next()
			return
		}

		raw, ok := c.Get("user")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		user, ok := raw.(User)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var grants []Grant
		err := h.db.Where("user_email = ?", user.Email).Find(&grants).Error
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "error"})
			return
		}

		for _, g := range grants {
			for _, allowed := range roles {
				if g.RoleName == allowed {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
}

func (h *Handlers) APIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("skipauth") {
			c.Next()
			return
		}

		// Check bearer header
		authorization := c.Request.Header.Get("Authorization")
		raw := strings.Replace(authorization, "Bearer ", "", -1)

		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if len(raw) != 24 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var apiKey APIKey
		if err := h.db.First(&apiKey, "key = ?", raw).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Next()
	}
}

func (h *Handlers) createJWT(email string) (string, time.Time, error) {
	var settings Settings
	err := h.db.First(&settings).Error
	if err != nil {
		return "", time.Time{}, err
	}

	var key jose.JSONWebKey
	err = key.UnmarshalJSON(settings.PrivateJWK)
	if err != nil {
		return "", time.Time{}, err
	}

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return "", time.Time{}, err
	}

	expiry := time.Now().Add(time.Minute * 5)

	cl := jwt.Claims{
		Issuer:   "infra",
		Expiry:   jwt.NewNumericDate(expiry),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}
	custom := struct {
		Email string `json:"email"`
		Nonce string `json:"nonce"`
	}{
		email,
		generate.RandString(10),
	}

	raw, err := jwt.Signed(signer).Claims(cl).Claims(custom).CompactSerialize()
	if err != nil {
		return "", time.Time{}, err
	}

	return raw, expiry, nil
}

func (h *Handlers) Healthz(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (h *Handlers) WellKnownJWKs(c *gin.Context) {
	var settings Settings
	err := h.db.First(&settings).Error
	if err != nil {
		fmt.Println(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve JWKs"})
		return
	}

	var pubKey jose.JSONWebKey
	err = pubKey.UnmarshalJSON(settings.PublicJWK)
	if err != nil {
		fmt.Println(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve JWKs"})
		return
	}

	c.JSON(http.StatusOK, struct {
		Keys []jose.JSONWebKey `json:"keys"`
	}{
		[]jose.JSONWebKey{pubKey},
	})
}

func (h *Handlers) ListResources(c *gin.Context) {
	var resources []Resource
	err := h.db.Not("name = ?", "infra").Find(&resources).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{"could not list resources"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resources})
}

func (h *Handlers) ListUsers(c *gin.Context) {
	var users []User
	err := h.db.Preload("Grants.Role").Preload("Grants.Resource").Preload("Providers").Find(&users).Error
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

		err := infraProvider.CreateUser(tx, &user, form.Email, form.Password, "infra.member")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
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

	var self User
	raw, _ := c.Get("user")
	self, _ = raw.(User)

	err := h.db.Transaction(func(tx *gorm.DB) error {
		if self.ID == params.ID {
			return errors.New("cannot delete self")
		}

		var user User
		err := tx.First(&user, "id = ?", params.ID).Error
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

	c.JSON(http.StatusOK, DeleteResponse{true})
}

func (h *Handlers) ListProviders(c *gin.Context) {
	var providers []Provider
	if err := h.db.Find(&providers).Error; err != nil {
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
				ApiToken     string `form:"apiToken" binding:"required"`
				Domain       string `form:"domain" binding:"required,fqdn"`
				ClientID     string `form:"clientID" binding:"required"`
				ClientSecret string `form:"clientSecret" binding:"required"`
			}

			var oktaParams oktaBinds
			if err := c.Bind(&oktaParams); err != nil {
				return err
			}

			count := tx.Where(&Provider{Kind: "okta", Domain: oktaParams.Domain}).First(&Provider{}).RowsAffected
			if count > 0 {
				return errors.New("okta provider with this domain already exists")
			}

			provider.Kind = "okta"
			provider.ApiToken = oktaParams.ApiToken
			provider.Domain = oktaParams.Domain
			provider.ClientID = oktaParams.ClientID
			provider.ClientSecret = oktaParams.ClientSecret

			result := tx.Where(&Provider{Kind: "okta", Domain: oktaParams.Domain}).Create(&provider)
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

	provider.SyncUsers(h.db)

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

func (h *Handlers) ListGrants(c *gin.Context) {
	type binds struct {
		Resource string `form:"resource"`
		User     string `form:"user"`
		Role     string `form:"role"`
	}

	var params binds
	if err := c.BindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	var grants []Grant
	q := h.db.Debug()

	q = q.Joins("Role")
	if params.Role != "" {
		q = q.Where("Role.id = ? OR Role.name = ?", params.Role, params.Role)
	}

	q = q.Joins("User")
	if params.User != "" {
		q = q.Where("User.id = ? OR User.email = ?", params.User, params.User)
	}

	q = q.Joins("Resource")
	if params.Resource != "" {
		q = q.Where("Resource.id = ? OR Resource.name = ?", params.Resource, params.Resource)
	}

	err := q.Find(&grants).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{"could not list grants"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": grants})
}

func (h *Handlers) CreateGrant(c *gin.Context) {
	type binds struct {
		User     string `form:"user" binding:"email,required"`
		Resource string `form:"resource" binding:"required"`
		Role     string `form:"role"`
	}

	var params binds
	if err := c.Bind(&params); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	var grant Grant
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := tx.First(&user, "email = ?", params.User).Error; err != nil {
			return err
		}

		var resource Resource
		if err := tx.First(&resource, "name = ?", params.Resource).Error; err != nil {
			return err
		}

		grant := &Grant{
			UserEmail:    user.Email,
			ResourceName: resource.Name,
		}

		// TODO (jmorganca): match roles to resources properly (i.e. cannot add kubernetes roles to infra resource)
		if params.Role != "" {
			grant.RoleName = params.Role
		}

		result := tx.FirstOrCreate(grant, grant)
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errors.New("grant already exists")
		}

		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	c.JSON(http.StatusCreated, grant)
}

func (h *Handlers) DeleteGrant(c *gin.Context) {
	type binds struct {
		ID string `uri:"id" binding:"required"`
	}

	var params binds
	if err := c.BindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	err := h.db.Where("id = ?", params.ID).Delete(&Grant{}).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
	}

	c.JSON(http.StatusOK, DeleteResponse{true})
}

func (h *Handlers) CreateCreds(c *gin.Context) {
	intf, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{"unauthorized"})
		return
	}

	user, ok := intf.(User)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{"unauthorized"})
		return
	}

	token, expiry, err := h.createJWT(user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{"could not create credentials"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": token, "expirationTimestamp": expiry.Format(time.RFC3339)})
}

func (h *Handlers) ListAPIKeys(c *gin.Context) {
	var apiKeys []APIKey
	err := h.db.Find(&apiKeys).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{"could not list api keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": apiKeys})
}

func (h *Handlers) Login(c *gin.Context) {
	type Params struct {
		// Via Okta
		OktaDomain string `form:"okta-domain"`
		OktaCode   string `form:"okta-code"`

		// Via email + password
		Email    string `form:"email" validate:"email"`
		Password string `form:"password"`

		// Via refresh token token
		Token string `form:"token" validate:"len=32"`
	}

	var params Params
	if err := c.ShouldBind(&params); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	var user User
	var token Token

	switch {
	case params.Token != "":
		id := params.Token[:12]
		secret := params.Token[12:36]

		if err := h.db.First(&token, "id = ?", id).Error; err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{"unauthorized"})
			return
		}

		if err := token.CheckSecret(secret); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if time.Now().After(time.Unix(token.Expires, 0)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "expired"})
			return
		}
	case params.OktaCode != "":
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

		err = h.db.Where("email = ?", email).First(&user).Error
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{"user does not exist"})
			return
		}

	default:
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

		if err := h.db.Where("email = ?", emailParams.Email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{"unauthorized"})
			return
		}

		if err := bcrypt.CompareHashAndPassword(user.Password, []byte(emailParams.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{"unauthorized"})
			return
		}
	}

	// Rotate token if specified
	if token.ID != "" {
		if err := h.db.Where(&Token{ID: token.ID}).Delete(&Token{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{"could not create token"})
			return
		}
	}

	var newToken Token
	secret, err := NewToken(h.db, user.ID, &newToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{"could not create token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": newToken.ID + secret})
}

func (h *Handlers) Logout(c *gin.Context) {
	type Params struct {
		Token string `form:"token" validate:"required,len=36"`
	}

	var params Params
	if err := c.ShouldBind(&params); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	intf, exists := c.Get("token")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{"unauthorized"})
		return
	}

	token, ok := intf.(Token)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{"unauthorized"})
		return
	}

	if err := h.db.Where(&Token{UserID: token.UserID}).Delete(&Token{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{"could not delete sesssion"})
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handlers) Config(c *gin.Context) {
	// TODO (jmorganca): authorize + filter based on join token
	var grants []Grant
	err := h.db.Preload("User").Preload("Role").Not("resource_name = ?", "infra").Find(&grants).Error
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": grants})
}

func (h *Handlers) Register(c *gin.Context) {
	type binds struct {
		CA       string `form:"ca" binding:"required"`
		Endpoint string `form:"endpoint" binding:"required"`
		Name     string `form:"name" binding:"required"`
	}

	var form binds
	if err := c.Bind(&form); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	var resource Resource
	err := h.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Where(&Resource{Name: form.Name}).FirstOrCreate(&resource).Error
		if err != nil {
			return err
		}

		resource.KubernetesCA = form.CA
		resource.KubernetesEndpoint = form.Endpoint

		return tx.Save(&resource).Error
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handlers) addRoutes(router *gin.Engine) error {
	router.GET("/healthz", h.Healthz)
	router.GET("/.well-known/jwks.json", h.WellKnownJWKs)

	router.GET("/v1/resources", h.TokenMiddleware(), h.RoleMiddleware("infra.member", "infra.owner"), h.ListResources)
	router.GET("/v1/users", h.TokenMiddleware(), h.RoleMiddleware("infra.member", "infra.owner"), h.ListUsers)
	router.POST("/v1/users", h.TokenMiddleware(), h.RoleMiddleware("infra.owner"), h.CreateUser)
	router.DELETE("/v1/users/:id", h.TokenMiddleware(), h.RoleMiddleware("infra.owner"), h.DeleteUser)
	router.GET("/v1/providers", h.ListProviders)
	router.POST("/v1/providers", h.TokenMiddleware(), h.RoleMiddleware("infra.owner"), h.CreateProvider)
	router.DELETE("/v1/providers/:id", h.TokenMiddleware(), h.RoleMiddleware("infra.owner"), h.DeleteProvider)
	router.GET("/v1/grants", h.TokenMiddleware(), h.RoleMiddleware("infra.member", "infra.owner"), h.ListGrants)
	router.POST("/v1/grants", h.TokenMiddleware(), h.RoleMiddleware("infra.owner"), h.CreateGrant)
	router.DELETE("/v1/grants/:id", h.TokenMiddleware(), h.RoleMiddleware("infra.owner"), h.DeleteGrant)
	router.POST("/v1/creds", h.TokenMiddleware(), h.RoleMiddleware("infra.member", "infra.owner"), h.CreateCreds)
	router.GET("/v1/apikeys", h.TokenMiddleware(), h.RoleMiddleware("infra.owner"), h.ListAPIKeys)
	router.POST("/v1/login", h.Login)
	router.POST("/v1/logout", h.TokenMiddleware(), h.Logout)

	router.GET("/v1/config", h.APIKeyMiddleware(), h.Config)
	router.POST("/v1/register", h.APIKeyMiddleware(), h.Register)

	return nil
}

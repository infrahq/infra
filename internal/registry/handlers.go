package registry

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

func (h *Handlers) AdminMiddleware() gin.HandlerFunc {
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

		if !user.Admin {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Next()
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

func (h *Handlers) ApiKeyOrTokenMiddleware() gin.HandlerFunc {
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

		var apiKey APIKey
		if len(raw) == 24 && h.db.First(&apiKey, "key = ?", raw).Debug().Error == nil {
			c.Next()
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

func (h *Handlers) ListDestinations(c *gin.Context) {
	var destinations []Destination
	err := h.db.Find(&destinations).Error
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{"could not list destinations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": destinations})
}

func (h *Handlers) CreateDestination(c *gin.Context) {
	type binds struct {
		CA       string `form:"ca" binding:"required"`
		Endpoint string `form:"endpoint" binding:"required"`
		Name     string `form:"name" binding:"required"`
	}

	var form binds
	if err := c.Bind(&form); err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	var destination Destination
	var created bool
	err := h.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where(&Destination{Name: form.Name}).FirstOrCreate(&destination)
		if result.Error != nil {
			return result.Error
		}

		created = result.RowsAffected > 0

		destination.KubernetesCA = form.CA
		destination.KubernetesEndpoint = form.Endpoint

		return tx.Save(&destination).Error
	})
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	// TODO (jmorganca): should we instead return an error if the destination already exists?
	if created {
		c.Status(http.StatusCreated)
	} else {
		c.Status(http.StatusOK)
	}
}

func (h *Handlers) ListUsers(c *gin.Context) {
	type binds struct {
		Email string `form:"email"`
	}

	var params binds
	if err := c.BindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	var users []User
	q := h.db.Preload("Permissions").Preload("Sources")

	if params.Email != "" {
		q = q.Where("email = ?", params.Email)
	}

	err := q.Find(&users).Error
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
		var infraSource Source
		if err := tx.Where(&Source{Kind: DefaultInfraSourceKind}).First(&infraSource).Error; err != nil {
			return err
		}

		if tx.Model(&infraSource).Where(&User{Email: form.Email}).Association("Users").Count() > 0 {
			return errors.New("user with this email already exists")
		}

		err := infraSource.CreateUser(tx, &user, form.Email, form.Password)
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

		if tx.Model(&user).Where(&Source{Kind: DefaultInfraSourceKind}).Association("Sources").Count() == 0 {
			return errors.New("user managed by external identity source")
		}

		var count int64
		err = tx.Where(&User{Admin: true}).Find(&[]User{}).Count(&count).Error
		if err != nil {
			return err
		}

		if user.Admin && count == 1 {
			return errors.New("cannot delete last admin user")
		}

		var infraSource Source
		if err := tx.Where(&Source{Kind: DefaultInfraSourceKind}).First(&infraSource).Error; err != nil {
			return err
		}

		err = infraSource.DeleteUser(tx, &user)
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

func (h *Handlers) ListSources(c *gin.Context) {
	var sources []Source
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		err := tx.Where(&User{Admin: true}).Find(&[]User{}).Count(&count).Error
		if err != nil {
			return err
		}

		if count == 0 {
			return errors.New("no admin user")
		}

		return tx.Find(&sources).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sources})
}

func (h *Handlers) CreateSource(c *gin.Context) {
	type binds struct {
		Kind string `form:"kind" binding:"required,oneof=okta"`
	}

	var params binds
	if err := c.Bind(&params); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	var source Source

	err := h.db.Transaction(func(tx *gorm.DB) error {
		switch params.Kind {
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

			count := tx.Where(&Source{Kind: "okta", Domain: oktaParams.Domain}).First(&Source{}).RowsAffected
			if count > 0 {
				return errors.New("okta source with this domain already exists")
			}

			source.Kind = "okta"
			source.ApiToken = oktaParams.ApiToken
			source.Domain = oktaParams.Domain
			source.ClientID = oktaParams.ClientID
			source.ClientSecret = oktaParams.ClientSecret

			result := tx.Where(&Source{Kind: "okta", Domain: oktaParams.Domain}).Create(&source)
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

	source.SyncUsers(h.db)

	c.JSON(http.StatusCreated, source)
}

func (h *Handlers) DeleteSource(c *gin.Context) {
	type binds struct {
		ID string `uri:"id" binding:"required"`
	}

	var params binds
	if err := c.BindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var source Source
		count := tx.First(&source, "ID = ?", params.ID).RowsAffected
		if count == 0 {
			return errors.New("no such source")
		}

		if source.Kind == DefaultInfraSourceKind {
			return errors.New("cannot delete infra source")
		}

		return tx.Delete(&source).Error
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	c.JSON(http.StatusOK, DeleteResponse{true})
}

func (h *Handlers) ListPermissions(c *gin.Context) {
	type binds struct {
		Destination string `form:"destination"`
		User        string `form:"user"`
	}

	var params binds
	if err := c.BindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	var permissions []Permission
	q := h.db.Joins("User")
	if params.User != "" {
		q = q.Where("User.id = ? OR User.email = ?", params.User, params.User)
	}

	q = q.Joins("Destination")
	if params.Destination != "" {
		q = q.Where("Destination.id = ? OR Destination.name = ?", params.Destination, params.Destination)
	}

	err := q.Find(&permissions).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{"could not list permissions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": permissions})
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
		var source Source
		if err := h.db.Where(&Source{Kind: "okta", Domain: params.OktaDomain}).First(&source).Error; err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{"no source with okta domain"})
			return
		}

		email, err := okta.EmailFromCode(
			params.OktaCode,
			source.Domain,
			source.ClientID,
			source.ClientSecret,
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

func (h *Handlers) Signup(c *gin.Context) {
	type Params struct {
		Email    string `form:"email" validate:"required,email"`
		Password string `form:"password" validate:"required,email"`
	}

	var params Params
	if err := c.ShouldBind(&params); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		fmt.Println(err)
		return
	}

	var token Token
	var secret string
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		err := tx.Where(&User{Admin: true}).Find(&[]User{}).Count(&count).Error
		if err != nil {
			return err
		}

		if count > 0 {
			return errors.New("admin user already exists")
		}

		var infraSource Source
		if err := tx.Where(&Source{Kind: DefaultInfraSourceKind}).First(&infraSource).Error; err != nil {
			return err
		}

		var user User
		if err := infraSource.CreateUser(tx, &user, params.Email, params.Password); err != nil {
			return err
		}

		user.Admin = true

		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		secret, err = NewToken(tx, user.ID, &token)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token.ID + secret})
}

func (h *Handlers) addRoutes(router *gin.Engine) error {
	router.GET("/healthz", h.Healthz)
	router.GET("/.well-known/jwks.json", h.WellKnownJWKs)

	router.GET("/v1/users", h.TokenMiddleware(), h.ListUsers)
	router.POST("/v1/users", h.TokenMiddleware(), h.AdminMiddleware(), h.CreateUser)
	router.DELETE("/v1/users/:id", h.TokenMiddleware(), h.AdminMiddleware(), h.DeleteUser)
	router.GET("/v1/destinations", h.TokenMiddleware(), h.ListDestinations)
	router.POST("/v1/destinations", h.APIKeyMiddleware(), h.CreateDestination)
	router.GET("/v1/sources", h.ListSources)
	router.POST("/v1/sources", h.TokenMiddleware(), h.AdminMiddleware(), h.CreateSource)
	router.DELETE("/v1/sources/:id", h.TokenMiddleware(), h.AdminMiddleware(), h.DeleteSource)
	router.GET("/v1/permissions", h.ApiKeyOrTokenMiddleware(), h.ListPermissions)
	router.POST("/v1/creds", h.TokenMiddleware(), h.CreateCreds)
	router.GET("/v1/apikeys", h.TokenMiddleware(), h.AdminMiddleware(), h.ListAPIKeys)
	router.POST("/v1/login", h.Login)
	router.POST("/v1/logout", h.TokenMiddleware(), h.Logout)
	router.POST("/v1/signup", h.Signup)

	return nil
}

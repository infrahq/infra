package registry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
	"gopkg.in/segmentio/analytics-go.v3"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type API struct {
	db       *gorm.DB
	okta     Okta
	t        *Telemetry
	registry *Registry
}

type CustomJWTClaims struct {
	Email       string `json:"email" validate:"required"`
	Destination string `json:"dest" validate:"required"`
	Nonce       string `json:"nonce" validate:"required"`
}

func NewAPIMux(reg *Registry, router *gin.RouterGroup) {
	a := API{
		db:       reg.db,
		okta:     reg.okta,
		t:        reg.tel,
		registry: reg,
	}

	authorized := router.Group("/" /*, AuthRequired()*/)
	{
		authorized.GET("/users", a.bearerAuthMiddleware(api.USERS_READ), a.ListUsers)
		authorized.GET("/users/:id", a.bearerAuthMiddleware(api.USERS_READ), a.GetUser)
		authorized.GET("/groups", a.bearerAuthMiddleware(api.GROUPS_READ), a.ListGroups)
		authorized.GET("/groups/:id", a.bearerAuthMiddleware(api.GROUPS_READ), a.GetGroup)
		authorized.GET("/destinations", a.bearerAuthMiddleware(api.DESTINATIONS_READ), a.ListDestinations)
		authorized.POST("/destinations", a.bearerAuthMiddleware(api.DESTINATIONS_CREATE), a.CreateDestination)
		authorized.GET("/destinations/:id", a.bearerAuthMiddleware(api.DESTINATIONS_READ), a.GetDestination)
		authorized.GET("/api-keys", a.bearerAuthMiddleware(api.API_KEYS_READ), a.ListAPIKeys)
		authorized.POST("/api-keys", a.bearerAuthMiddleware(api.API_KEYS_CREATE), a.CreateAPIKey)
		authorized.DELETE("/api-keys/:id", a.bearerAuthMiddleware(api.API_KEYS_DELETE), a.DeleteAPIKey)
		authorized.POST("/tokens", a.bearerAuthMiddleware(api.TOKENS_CREATE), a.CreateToken)
		authorized.GET("/roles", a.bearerAuthMiddleware(api.ROLES_READ), a.ListRoles)
		authorized.GET("/roles/:id", a.bearerAuthMiddleware(api.ROLES_READ), a.GetRole)
		authorized.POST("/logout", a.bearerAuthMiddleware(api.AUTH_DELETE), a.Logout)
	}

	unauthorized := router.Group("/")
	// these endpoints are left unauthenticated so that infra login can see what the providers are that are available
	{
		unauthorized.GET("/providers", a.ListProviders)
		unauthorized.GET("/providers/:id", a.GetProvider)

		unauthorized.POST("/login", a.Login)
		unauthorized.GET("/version", a.Version)
	}
}

func sendAPIError(c *gin.Context, code int, message string) {
	c.JSON(code, &api.Error{
		Code:    int32(code),
		Message: message,
	})
}

func (a *API) bearerAuthMiddleware(required api.InfraAPIPermission) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.Request.Header.Get("Authorization")
		raw := strings.ReplaceAll(authorization, "Bearer ", "") // TODO: switch on bearer instead of filtering it out

		if raw == "" {
			// Fall back to checking cookies if the bearer header is not provided
			cookie, err := c.Cookie(CookieTokenName)
			if err != nil {
				logging.L.Debug("could not read token from cookie")
				sendAPIError(c, http.StatusUnauthorized, "unauthorized")

				return
			}

			raw = cookie
		}

		switch len(raw) {
		case TokenLen:
			token, err := ValidateAndGetToken(a.db, raw)
			if err != nil {
				logging.L.Debug(err.Error())

				switch err.Error() {
				case "token expired":
					sendAPIError(c, http.StatusForbidden, "forbidden")
				default:
					sendAPIError(c, http.StatusUnauthorized, "unauthorized")
				}

				return
			}

			c.Set(tokenContextKey, token)
			c.Next()

			return
		case APIKeyLen:
			var apiKey APIKey
			if err := a.db.First(&apiKey, &APIKey{Key: raw}).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					logging.S.Debugf("invalid API key: %w", err)
				} else {
					logging.S.Errorf("api key lookup: %w", err)
				}

				sendAPIError(c, http.StatusUnauthorized, "unauthorized")

				return
			}

			hasPermission := checkPermission(required, apiKey.Permissions)
			if !hasPermission {
				// at this point we know their key is valid, so we can present a more detailed error
				sendAPIError(c, http.StatusForbidden, string(required)+" permission is required")
				return
			}

			c.Set(apiKeyContextKey, &apiKey)
			c.Next()

			return
		}

		logging.L.Debug("invalid token length provided")
		sendAPIError(c, http.StatusUnauthorized, "unauthorized")
	}
}

// checkPermission checks if a token that has already been validated has a specified permission
func checkPermission(required api.InfraAPIPermission, tokenPermissions string) bool {
	if tokenPermissions == string(api.STAR) {
		// this is the root token
		return true
	}

	permissions := strings.Split(tokenPermissions, " ")
	for _, permission := range permissions {
		if permission == string(required) {
			return true
		}
	}

	return false
}

var tokenContextKey string = "tokenContextKey"

func extractToken(context *gin.Context) (*Token, error) {
	val, ok := context.Get(tokenContextKey)
	if !ok {
		return nil, errors.New("token not found in context")
	}

	token, ok := val.(*Token)
	if !ok {
		return nil, errors.New("token not found in context")
	}

	return token, nil
}

var apiKeyContextKey string = "apiKeyContextKey"

func extractAPIKey(context context.Context) (*APIKey, error) {
	apiKey, ok := context.Value(apiKeyContextKey).(*APIKey)
	if !ok {
		return nil, errors.New("apikey not found in context")
	}

	return apiKey, nil
}

func (a *API) ListUsers(c *gin.Context) {
	userEmail := c.Request.URL.Query().Get("email")

	var users []User

	err := a.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Preload("Roles.Destination.Labels").Preload("Groups.Roles.Destination.Labels").Preload(clause.Associations).Find(&users, &User{Email: userEmail}).Error
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(c, http.StatusInternalServerError, "could not list users")

		return
	}

	results := make([]api.User, 0)
	for _, u := range users {
		results = append(results, u.marshal())
	}

	c.JSON(http.StatusOK, results)
}

func (a *API) GetUser(c *gin.Context) {
	userId := c.Param("id")
	if userId == "" {
		sendAPIError(c, http.StatusBadRequest, "Path parameter \"id\" is required")

		return
	}

	var user User

	err := a.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Preload("Roles.Destination").Preload("Groups.Roles.Destination").Preload(clause.Associations).First(&user, &User{Id: userId}).Error
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logging.S.Debugf("invalid user ID: %w", err)
			sendAPIError(c, http.StatusNotFound, fmt.Sprintf("Could not find user ID \"%s\"", userId))
		} else {
			logging.S.Errorf("user ID lookup: %w", err)
			sendAPIError(c, http.StatusInternalServerError, err.Error())
		}

		return
	}

	result := user.marshal()

	c.JSON(http.StatusOK, result)
}

func (a *API) ListGroups(c *gin.Context) {
	groupName := c.Request.URL.Query().Get("name")

	var groups []Group
	if err := a.db.Preload(clause.Associations).Find(&groups, &Group{Name: groupName}).Error; err != nil {
		logging.L.Error(err.Error())
		sendAPIError(c, http.StatusInternalServerError, "could not list groups")

		return
	}

	results := make([]api.Group, 0)
	for _, g := range groups {
		results = append(results, g.marshal())
	}

	c.JSON(http.StatusOK, results)
}

func (a *API) GetGroup(c *gin.Context) {
	groupId := c.Param("id")
	if groupId == "" {
		sendAPIError(c, http.StatusBadRequest, "Path parameter \"id\" is required")

		return
	}

	var group Group
	if err := a.db.Preload(clause.Associations).First(&group, &Group{Id: groupId}).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logging.S.Debugf("invalid group ID: %w", err)
			sendAPIError(c, http.StatusNotFound, fmt.Sprintf("Could not find group ID \"%s\"", groupId))
		} else {
			logging.S.Errorf("group ID lookup: %w", err)
			sendAPIError(c, http.StatusInternalServerError, err.Error())
		}

		return
	}

	result := group.marshal()

	c.JSON(http.StatusOK, result)
}

func (a *API) ListProviders(c *gin.Context) {
	// caution: this endpoint is unauthenticated, do not return sensitive info
	providerKind := c.Request.URL.Query().Get("kind")

	var providers []Provider
	if err := a.db.Find(&providers, &Provider{Kind: providerKind}).Error; err != nil {
		logging.L.Error(err.Error())
		sendAPIError(c, http.StatusInternalServerError, "could not list providers")

		return
	}

	results := make([]api.Provider, 0)
	for _, p := range providers {
		results = append(results, p.marshal())
	}

	c.JSON(http.StatusOK, results)
}

func (a *API) GetProvider(c *gin.Context) {
	// caution: this endpoint is unauthenticated, do not return sensitive info
	providerId := c.Param("id")
	if providerId == "" {
		sendAPIError(c, http.StatusBadRequest, "Path parameter \"id\" is required")

		return
	}

	var provider Provider
	if err := a.db.First(&provider, &Provider{Id: providerId}).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logging.S.Debugf("invalid provider ID: %w", err)
			sendAPIError(c, http.StatusNotFound, fmt.Sprintf("Could not find provider ID \"%s\"", providerId))
		} else {
			logging.S.Errorf("provider ID lookup: %w", err)
			sendAPIError(c, http.StatusInternalServerError, err.Error())
		}

		return
	}

	result := provider.marshal()

	c.JSON(http.StatusOK, result)
}

func (a *API) ListDestinations(c *gin.Context) {
	destinationName := c.Request.URL.Query().Get("name")
	destinationKind := c.Request.URL.Query().Get("kind")

	var destinations []Destination
	if err := a.db.Preload("Labels").Find(&destinations, &Destination{Name: destinationName, Kind: destinationKind}).Error; err != nil {
		logging.L.Error(err.Error())
		sendAPIError(c, http.StatusInternalServerError, "could not list destinations")

		return
	}

	results := make([]api.Destination, 0)
	for _, d := range destinations {
		results = append(results, d.marshal())
	}

	c.JSON(http.StatusOK, results)
}

func (a *API) GetDestination(c *gin.Context) {
	destinationId := c.Param("id")
	if destinationId == "" {
		sendAPIError(c, http.StatusBadRequest, "Path parameter \"id\" is required")

		return
	}

	var destination Destination
	if err := a.db.First(&destination, &Destination{Id: destinationId}).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logging.S.Debugf("invalid destination ID: %w", err)
			sendAPIError(c, http.StatusNotFound, fmt.Sprintf("Could not find destination ID \"%s\"", destinationId))
		} else {
			logging.S.Errorf("destination ID lookup: %w", err)
			sendAPIError(c, http.StatusInternalServerError, err.Error())
		}

		return
	}

	result := destination.marshal()

	c.JSON(http.StatusOK, result)
}

func (a *API) CreateDestination(c *gin.Context) {
	_, err := extractAPIKey(c)
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(c, http.StatusUnauthorized, "unauthorized")

		return
	}

	body := &api.DestinationCreateRequest{}

	if err := c.BindJSON(body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err.Error())
		return
	}

	if !body.Kind.IsValid() {
		sendAPIError(c, http.StatusBadRequest, fmt.Sprintf("unrecognized destination kind %s", string(body.Kind)))
		return
	}

	destination := Destination{
		Name:               body.Name,
		Kind:               string(body.Kind),
		KubernetesCa:       body.Kubernetes.Ca,
		KubernetesEndpoint: body.Kubernetes.Endpoint,
	}

	automaticLabels := []string{
		string(body.Kind),
	}

	err = a.db.Transaction(func(tx *gorm.DB) error {
		for _, l := range append(body.Labels, automaticLabels...) {
			var label Label
			if err := tx.FirstOrCreate(&label, &Label{Value: l}).Error; err != nil {
				return err
			}

			destination.Labels = append(destination.Labels, label)
		}

		if err := tx.FirstOrCreate(&destination, &Destination{NodeID: body.NodeID}).Error; err != nil {
			return err
		}

		if err := tx.Save(&destination).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(c, http.StatusInternalServerError, err.Error())

		return
	}

	c.JSON(http.StatusCreated, destination)
}

func (a *API) ListAPIKeys(c *gin.Context) {
	_, err := extractAPIKey(c)
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(c, http.StatusUnauthorized, "unauthorized")

		return
	}

	keyName := c.Request.URL.Query().Get("name")

	var keys []APIKey

	err = a.db.Find(&keys, &APIKey{Name: keyName}).Error
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(c, http.StatusInternalServerError, "could not list keys")

		return
	}

	results := make([]api.InfraAPIKey, 0)

	for _, k := range keys {
		resKey, err := k.marshal()
		if err != nil {
			sendAPIError(c, http.StatusInternalServerError, "unexpected value encountered while marshalling API key")
			return
		}

		results = append(results, *resKey)
	}

	c.JSON(http.StatusOK, results)
}

func (a *API) DeleteAPIKey(c *gin.Context) {
	_, err := extractAPIKey(c)
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(c, http.StatusUnauthorized, "unauthorized")

		return
	}

	id := c.Param("id")
	if id == "" {
		sendAPIError(c, http.StatusBadRequest, "API key ID must be specified")
		return
	}

	if err := a.db.Delete(&APIKey{Id: id}).Error; err != nil {
		logging.S.Errorf("api key delete: %w", err)
		sendAPIError(c, http.StatusInternalServerError, err.Error())

		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (a *API) CreateAPIKey(c *gin.Context) {
	_, err := extractAPIKey(c)
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(c, http.StatusUnauthorized, "unauthorized")

		return
	}

	body := &api.InfraAPIKeyCreateRequest{}
	if err := c.BindJSON(body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err.Error())
		return
	}

	if strings.ToLower(body.Name) == engineAPIKeyName || strings.ToLower(body.Name) == rootAPIKeyName {
		// this name is used for the default API key that engines use to connect to Infra
		sendAPIError(c, http.StatusBadRequest, fmt.Sprintf("cannot create an API key with the name %s, this name is reserved", body.Name))
		return
	}

	var apiKey APIKey

	err = a.db.Transaction(func(tx *gorm.DB) error {
		tx.First(&apiKey, &APIKey{Name: body.Name})
		if apiKey.Id != "" {
			return ErrExistingKey
		}

		apiKey.Name = body.Name
		var permissions string
		for _, p := range body.Permissions {
			permissions += string(p) + " "
		}
		permissions = strings.TrimSpace(permissions)

		if len(strings.ReplaceAll(permissions, " ", "")) == 0 {
			return ErrKeyPermissionsNotFound
		}
		apiKey.Permissions = permissions
		return tx.Create(&apiKey).Error
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrExistingKey):
			logging.S.Debugf("API key existing creation: %w", err)
			sendAPIError(c, http.StatusConflict, "An API key with this name already exists")
		case errors.Is(err, ErrKeyPermissionsNotFound):
			logging.S.Debugf("API key permission creation: %w", err)
			sendAPIError(c, http.StatusBadRequest, "API key could not be created, permissions are required")
		default:
			logging.S.Errorf("API key creation: %w", err)
			sendAPIError(c, http.StatusInternalServerError, err.Error())
		}

		return
	}

	res, err := apiKey.marshalWithSecret()
	if err != nil {
		sendAPIError(c, http.StatusInternalServerError, "unexpected value encountered while marshalling response")
		return
	}

	c.JSON(http.StatusCreated, res)
}

func (a *API) ListRoles(c *gin.Context) {
	roleName := c.Request.URL.Query().Get("name")
	roleKind := c.Request.URL.Query().Get("kind")
	destinationId := c.Request.URL.Query().Get("destination")

	var roles []Role

	err := a.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Preload("Groups.Users").Preload(clause.Associations).Find(&roles, &Role{Name: roleName, Kind: roleKind, DestinationId: destinationId}).Error
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(c, http.StatusInternalServerError, "could not list roles")

		return
	}

	results := make([]api.Role, 0)
	for _, r := range roles {
		results = append(results, r.marshal())
	}

	c.JSON(http.StatusOK, results)
}

func (a *API) GetRole(c *gin.Context) {
	roleId := c.Param("id")
	if roleId == "" {
		sendAPIError(c, http.StatusBadRequest, "Path parameter \"id\" is required")

		return
	}

	var role Role
	if err := a.db.Preload("Groups.Users").Preload(clause.Associations).First(&role, &Role{Id: roleId}).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logging.S.Debugf("invalid role ID: %w", err)
			sendAPIError(c, http.StatusNotFound, fmt.Sprintf("Could not find role ID \"%s\"", roleId))
		} else {
			logging.S.Errorf("role ID lookup: %w", err)
			sendAPIError(c, http.StatusInternalServerError, err.Error())
		}

		return
	}

	result := role.marshal()

	c.JSON(http.StatusOK, result)
}

var signatureAlgFromKeyAlgorithm = map[string]string{
	"ED25519": "EdDSA", // elliptic curve 25519
}

func (a *API) createJWT(destination, email string) (rawJWT string, expiry time.Time, err error) {
	var settings Settings

	err = a.db.First(&settings).Error
	if err != nil {
		return "", time.Time{}, fmt.Errorf("can't find jwt settings: %w", err)
	}

	var key jose.JSONWebKey

	err = key.UnmarshalJSON(settings.PrivateJWK)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("unmarshal privateJWK: %w", err)
	}

	sigAlg, ok := signatureAlgFromKeyAlgorithm[key.Algorithm]
	if !ok {
		return "", time.Time{}, fmt.Errorf("unexpected key algorithm %q needs matching signature algorithm", key.Algorithm)
	}

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.SignatureAlgorithm(sigAlg), Key: key}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("creating signer for signature algorithm %q: %w", key.Algorithm, err)
	}

	nonce, err := generate.RandString(10)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generating nonce: %w", err)
	}

	expiry = time.Now().Add(time.Minute * 5)
	cl := jwt.Claims{
		Issuer:    "infra",
		NotBefore: jwt.NewNumericDate(time.Now().Add(-5 * time.Minute)), // allow for clock drift
		Expiry:    jwt.NewNumericDate(expiry),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	custom := CustomJWTClaims{
		Email:       email,
		Destination: destination,
		Nonce:       nonce,
	}

	rawJWT, err = jwt.Signed(signer).Claims(cl).Claims(custom).CompactSerialize()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("serializing jwt: %w", err)
	}

	return rawJWT, expiry, nil
}

func (a *API) CreateToken(c *gin.Context) {
	token, err := extractToken(c)
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(c, http.StatusUnauthorized, "unauthorized")

		return
	}

	body := &api.TokenRequest{}
	if err := c.BindJSON(body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err.Error())
		return
	}

	jwt, expiry, err := a.createJWT(*body.Destination, token.User.Email)
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(c, http.StatusInternalServerError, "could not generate cred")

		return
	}

	c.JSON(http.StatusOK, api.Token{Token: jwt, Expires: expiry.Unix()})
}

func (a *API) Login(c *gin.Context) {
	body := &api.LoginRequest{}
	if err := c.BindJSON(body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err.Error())
		return
	}

	var user User

	var token Token

	switch {
	case body.Okta != nil:
		var provider Provider
		if err := a.db.Where(&Provider{Kind: ProviderKindOkta, Domain: body.Okta.Domain}).First(&provider).Error; err != nil {
			logging.L.Debug("Could not retrieve okta provider from db: " + err.Error())
			sendAPIError(c, http.StatusBadRequest, "invalid okta login information")

			return
		}

		clientSecret, err := a.registry.GetSecret(provider.ClientSecret)
		if err != nil {
			logging.L.Error("Could not retrieve okta client secret from provider: " + err.Error())
			sendAPIError(c, http.StatusInternalServerError, "invalid okta login information")

			return
		}

		email, err := a.okta.EmailFromCode(
			body.Okta.Code,
			provider.Domain,
			provider.ClientID,
			clientSecret,
		)
		if err != nil {
			logging.L.Debug("Could not extract email from okta info: " + err.Error())
			sendAPIError(c, http.StatusUnauthorized, "invalid okta login information")

			return
		}

		err = a.db.Where("email = ?", email).First(&user).Error
		if err != nil {
			logging.L.Debug("Could not get user from database: " + err.Error())
			sendAPIError(c, http.StatusUnauthorized, "invalid okta login information")

			return
		}
	default:
		sendAPIError(c, http.StatusBadRequest, "invalid login information provided")
		return
	}

	secret, err := NewToken(a.db, user.Id, a.registry.options.SessionDuration, &token)
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(c, http.StatusInternalServerError, "could not create token")

		return
	}

	tokenString := token.Id + secret

	setAuthCookie(c, tokenString, a.registry.options.SessionDuration)

	if err := a.t.Enqueue(analytics.Track{Event: "infra.login", UserId: user.Id}); err != nil {
		logging.S.Debug(err)
	}

	c.JSON(http.StatusOK, api.LoginResponse{Name: user.Email, Token: tokenString})
}

func (a *API) Logout(c *gin.Context) {
	token, err := extractToken(c)
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(c, http.StatusBadRequest, "invalid token")

		return
	}

	if err := a.db.Where(&Token{UserId: token.UserId}).Delete(&Token{}).Error; err != nil {
		sendAPIError(c, http.StatusInternalServerError, "could not log out user")
		logging.L.Error(err.Error())

		return
	}

	deleteAuthCookie(c)

	if err := a.t.Enqueue(analytics.Track{Event: "infra.logout", UserId: token.UserId}); err != nil {
		logging.S.Debug(err)
	}

	c.JSON(http.StatusOK, nil)
}

func (a *API) Version(c *gin.Context) {
	c.JSON(http.StatusOK, api.Version{Version: internal.Version})
}

func (s *Provider) marshal() api.Provider {
	res := api.Provider{
		Id:       s.Id,
		Created:  s.Created,
		Updated:  s.Updated,
		ClientID: s.ClientID,
		Domain:   s.Domain,
		Kind:     s.Kind,
	}

	return res
}

func (d *Destination) marshal() api.Destination {
	res := api.Destination{
		NodeID:  d.NodeID,
		Name:    d.Name,
		Kind:    d.Kind,
		Id:      d.Id,
		Created: d.Created,
		Updated: d.Updated,
	}

	if d.Kind == DestinationKindKubernetes {
		res.Kubernetes = &api.DestinationKubernetes{
			Ca:       d.KubernetesCa,
			Endpoint: d.KubernetesEndpoint,
		}
	}

	for _, l := range d.Labels {
		switch l.Value {
		case d.Kind: // skip Kind
		default:
			res.Labels = append(res.Labels, l.Value)
		}
	}

	return res
}

func (k *APIKey) marshal() (*api.InfraAPIKey, error) {
	res := &api.InfraAPIKey{
		Name:    k.Name,
		Id:      k.Id,
		Created: k.Created,
	}

	permissions, err := marshalPermissions(k.Permissions)
	if err != nil {
		return nil, err
	}

	res.Permissions = permissions

	return res, nil
}

// This function returns the secret key, it should only be used after the initial key creation
func (k *APIKey) marshalWithSecret() (*api.InfraAPIKeyCreateResponse, error) {
	res := &api.InfraAPIKeyCreateResponse{
		Name:    k.Name,
		Id:      k.Id,
		Created: k.Created,
		Key:     k.Key,
	}

	permissions, err := marshalPermissions(k.Permissions)
	if err != nil {
		return nil, err
	}

	res.Permissions = permissions

	return res, nil
}

func marshalPermissions(permissions string) ([]api.InfraAPIPermission, error) {
	var apiPermissions []api.InfraAPIPermission

	storedPermissions := strings.Split(permissions, " ")
	for _, p := range storedPermissions {
		apiPermission, err := api.NewInfraAPIPermissionFromValue(p)
		if err != nil {
			logging.S.Errorf("Error converting stored permission %q to API permission: %w", p, err)
			return nil, err
		}

		apiPermissions = append(apiPermissions, *apiPermission)
	}

	return apiPermissions, nil
}

func (r Role) marshal() api.Role {
	res := api.Role{
		Id:        r.Id,
		Created:   r.Created,
		Updated:   r.Updated,
		Name:      r.Name,
		Namespace: r.Namespace,
	}

	res.Kind = api.RoleKind(r.Kind)

	for _, u := range r.Users {
		res.Users = append(res.Users, u.marshal())
	}

	for _, g := range r.Groups {
		res.Groups = append(res.Groups, g.marshal())
	}

	res.Destination = r.Destination.marshal()

	return res
}

func (u *User) marshal() api.User {
	res := api.User{
		Id:      u.Id,
		Email:   u.Email,
		Created: u.Created,
		Updated: u.Updated,
	}

	for _, g := range u.Groups {
		res.Groups = append(res.Groups, g.marshal())
	}

	for _, r := range u.Roles {
		res.Roles = append(res.Roles, r.marshal())
	}

	return res
}

func (g *Group) marshal() api.Group {
	res := api.Group{
		Id:         g.Id,
		Created:    g.Created,
		Updated:    g.Updated,
		Name:       g.Name,
		ProviderID: g.ProviderId,
	}

	for _, u := range g.Users {
		res.Users = append(res.Users, u.marshal())
	}

	for _, r := range g.Roles {
		res.Roles = append(res.Roles, r.marshal())
	}

	return res
}

package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
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

var (
	validate        *validator.Validate = validator.New()
	SessionDuration time.Duration       = time.Hour * 24
)

func NewAPIMux(reg *Registry) *mux.Router {
	a := API{
		db:       reg.db,
		okta:     reg.okta,
		t:        reg.tel,
		registry: reg,
	}

	r := mux.NewRouter()
	v1 := r.PathPrefix("/v1").Subrouter()
	v1.Handle("/users", a.bearerAuthMiddleware(api.USERS_READ, http.HandlerFunc(a.ListUsers))).Methods(http.MethodGet)
	v1.Handle("/users/{id}", a.bearerAuthMiddleware(api.USERS_READ, http.HandlerFunc(a.GetUser))).Methods(http.MethodGet)

	v1.Handle("/groups", a.bearerAuthMiddleware(api.GROUPS_READ, http.HandlerFunc(a.ListGroups))).Methods(http.MethodGet)
	v1.Handle("/groups/{id}", a.bearerAuthMiddleware(api.GROUPS_READ, http.HandlerFunc(a.GetGroup))).Methods(http.MethodGet)

	v1.Handle("/providers", http.HandlerFunc(a.ListProviders)).Methods(http.MethodGet)
	v1.Handle("/providers", a.bearerAuthMiddleware(api.PROVIDERS_CREATE, http.HandlerFunc(a.CreateProvider))).Methods(http.MethodPost)
	v1.Handle("/providers/{id}", http.HandlerFunc(a.GetProvider)).Methods(http.MethodGet)
	v1.Handle("/providers/{id}", a.bearerAuthMiddleware(api.PROVIDERS_UPDATE, http.HandlerFunc(a.UpdateProvider))).Methods(http.MethodPut)
	v1.Handle("/providers/{id}", a.bearerAuthMiddleware(api.PROVIDERS_DELETE, http.HandlerFunc(a.DeleteProvider))).Methods(http.MethodDelete)

	v1.Handle("/destinations", a.bearerAuthMiddleware(api.DESTINATIONS_READ, http.HandlerFunc(a.ListDestinations))).Methods(http.MethodGet)
	v1.Handle("/destinations", a.bearerAuthMiddleware(api.DESTINATIONS_CREATE, http.HandlerFunc(a.CreateDestination))).Methods(http.MethodPost)
	v1.Handle("/destinations/{id}", a.bearerAuthMiddleware(api.DESTINATIONS_READ, http.HandlerFunc(a.GetDestination))).Methods(http.MethodGet)

	v1.Handle("/api-keys", a.bearerAuthMiddleware(api.API_KEYS_READ, http.HandlerFunc(a.ListAPIKeys))).Methods(http.MethodGet)
	v1.Handle("/api-keys", a.bearerAuthMiddleware(api.API_KEYS_CREATE, http.HandlerFunc(a.CreateAPIKey))).Methods(http.MethodPost)
	v1.Handle("/api-keys/{id}", a.bearerAuthMiddleware(api.API_KEYS_DELETE, http.HandlerFunc(a.DeleteAPIKey))).Methods(http.MethodDelete)

	v1.Handle("/tokens", a.bearerAuthMiddleware(api.TOKENS_CREATE, http.HandlerFunc(a.CreateToken))).Methods(http.MethodPost)

	v1.Handle("/roles", a.bearerAuthMiddleware(api.ROLES_READ, http.HandlerFunc(a.ListRoles))).Methods(http.MethodGet)
	v1.Handle("/roles/{id}", a.bearerAuthMiddleware(api.ROLES_READ, http.HandlerFunc(a.GetRole))).Methods(http.MethodGet)

	v1.Handle("/login", http.HandlerFunc(a.Login)).Methods(http.MethodPost)
	v1.Handle("/logout", a.bearerAuthMiddleware(api.AUTH_DELETE, http.HandlerFunc(a.Logout))).Methods(http.MethodPost)

	v1.Handle("/version", http.HandlerFunc(a.Version)).Methods(http.MethodGet)

	return r
}

func sendAPIError(w http.ResponseWriter, code int, message string) {
	err := api.Error{
		Code:    int32(code),
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(err); err != nil {
		logging.L.Error("could not send API error: " + err.Error())
	}
}

func (a *API) bearerAuthMiddleware(required api.InfraAPIPermission, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")
		raw := strings.ReplaceAll(authorization, "Bearer ", "")

		if raw == "" {
			// Fall back to checking cookies if the bearer header is not provided
			cookie, err := r.Cookie(CookieTokenName)
			if err != nil {
				logging.L.Debug("could not read token from cookie")
				sendAPIError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			raw = cookie.Value
		}

		switch len(raw) {
		case TokenLen:
			token, err := ValidateAndGetToken(a.db, raw)
			if err != nil {
				logging.L.Debug(err.Error())
				switch err.Error() {
				case "token expired":
					sendAPIError(w, http.StatusForbidden, "forbidden")
				default:
					sendAPIError(w, http.StatusUnauthorized, "unauthorized")
				}
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), tokenContextKey{}, token)))
			return
		case APIKeyLen:
			var apiKey APIKey
			if err := a.db.First(&apiKey, &APIKey{Key: raw}).Error; err != nil {
				logging.L.Error(err.Error())
				sendAPIError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			hasPermission := checkPermission(required, apiKey.Permissions)
			if !hasPermission {
				// at this point we know their key is valid, so we can present a more detailed error
				sendAPIError(w, http.StatusForbidden, string(required)+" permission is required")
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), apiKeyContextKey{}, &apiKey)))
			return
		}

		logging.L.Debug("invalid token length provided")
		sendAPIError(w, http.StatusUnauthorized, "unauthorized")
	})
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

type tokenContextKey struct{}

func extractToken(context context.Context) (*Token, error) {
	token, ok := context.Value(tokenContextKey{}).(*Token)
	if !ok {
		return nil, errors.New("token not found in context")
	}

	return token, nil
}

type apiKeyContextKey struct{}

func extractAPIKey(context context.Context) (*APIKey, error) {
	apiKey, ok := context.Value(apiKeyContextKey{}).(*APIKey)
	if !ok {
		return nil, errors.New("apikey not found in context")
	}

	return apiKey, nil
}

func (a *API) ListUsers(w http.ResponseWriter, r *http.Request) {
	userEmail := r.URL.Query().Get("email")

	var users []User

	err := a.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Preload("Roles.Destination").Preload("Groups.Roles.Destination").Preload(clause.Associations).Find(&users, &User{Email: userEmail}).Error
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not list users")

		return
	}

	results := make([]api.User, 0)
	for _, u := range users {
		results = append(results, u.marshal())
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(results); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not list users")
	}
}

func (a *API) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	userId := vars["id"]
	if userId == "" {
		sendAPIError(w, http.StatusBadRequest, "Path parameter \"id\" is required")

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
		logging.L.Error(err.Error())

		if errors.Is(err, gorm.ErrRecordNotFound) {
			sendAPIError(w, http.StatusNotFound, fmt.Sprintf("Could not find user ID \"%s\"", userId))
		} else {
			sendAPIError(w, http.StatusBadRequest, fmt.Sprintf("Could not find user ID \"%s\"", userId))
		}

		return
	}

	result := user.marshal()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, fmt.Sprintf("Could not get user \"%s\"", userId))
	}
}

func (a *API) ListGroups(w http.ResponseWriter, r *http.Request) {
	groupName := r.URL.Query().Get("name")

	var groups []Group
	if err := a.db.Preload(clause.Associations).Find(&groups, &Group{Name: groupName}).Error; err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not list groups")

		return
	}

	results := make([]api.Group, 0)
	for _, g := range groups {
		results = append(results, g.marshal())
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(results); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not list groups")
	}
}

func (a *API) GetGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	groupId := vars["id"]
	if groupId == "" {
		sendAPIError(w, http.StatusBadRequest, "Path parameter \"id\" is required")

		return
	}

	var group Group
	if err := a.db.Preload(clause.Associations).First(&group, &Group{Id: groupId}).Error; err != nil {
		logging.L.Error(err.Error())

		if errors.Is(err, gorm.ErrRecordNotFound) {
			sendAPIError(w, http.StatusNotFound, fmt.Sprintf("Could not find group ID \"%s\"", groupId))
		} else {
			sendAPIError(w, http.StatusBadRequest, fmt.Sprintf("Could not find group ID \"%s\"", groupId))
		}

		return
	}

	result := group.marshal()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not list groups")
	}
}

func (a *API) ListProviders(w http.ResponseWriter, r *http.Request) {
	providerKind := r.URL.Query().Get("kind")

	var providers []Provider
	if err := a.db.Find(&providers, &Provider{Kind: providerKind}).Error; err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not list providers")

		return
	}

	results := make([]api.Provider, 0)
	for _, p := range providers {
		results = append(results, p.marshal())
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(results); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not list providers")
	}
}

func (a *API) CreateProvider(w http.ResponseWriter, r *http.Request) {
	var body api.Provider
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	var provider Provider

	err := a.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where(&Destination{Kind: body.Kind}).FirstOrCreate(&provider)
		if result.Error != nil {
			return result.Error
		}

		provider.ClientID = body.ClientID
		provider.Domain = body.Domain
		provider.ClientSecret = body.ClientSecret

		if body.Kind == ProviderKindOkta {
			provider.APIToken = body.Okta.APIToken
		}

		return tx.Save(&provider).Error
	})
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not persist provider")

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(provider.marshal()); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not create provider")
	}
}

func (a *API) GetProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	providerId := vars["id"]
	if providerId == "" {
		sendAPIError(w, http.StatusBadRequest, "Path parameter \"id\" is required")

		return
	}

	var provider Provider
	if err := a.db.First(&provider, &Provider{Id: providerId}).Error; err != nil {
		logging.L.Error(err.Error())

		if errors.Is(err, gorm.ErrRecordNotFound) {
			sendAPIError(w, http.StatusNotFound, fmt.Sprintf("Could not find provider ID \"%s\"", providerId))
		} else {
			sendAPIError(w, http.StatusBadRequest, fmt.Sprintf("Could not find provider ID \"%s\"", providerId))
		}

		return
	}

	result := provider.marshal()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not list providers")
	}
}

func (a *API) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		sendAPIError(w, http.StatusBadRequest, "Provider ID must be specified")
	}

	var body api.Provider
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	var p Provider

	err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&p, &Provider{Id: id}).Error; err != nil {
			return err
		}

		p.ClientID = body.ClientID
		p.Domain = body.Domain
		p.ClientSecret = body.ClientSecret

		if body.Kind == ProviderKindOkta {
			p.APIToken = body.Okta.APIToken
		}

		return tx.Save(&p).Error
	})
	if err != nil {
		logging.L.Error(err.Error())

		if errors.Is(err, gorm.ErrRecordNotFound) {
			sendAPIError(w, http.StatusNotFound, "no provider found for the specified ID")
			return
		}

		sendAPIError(w, http.StatusInternalServerError, "could not persist updated provider")

		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *API) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		sendAPIError(w, http.StatusBadRequest, "Provider ID must be specified")
	}

	if err := a.db.Delete(&Provider{Id: id}).Error; err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, err.Error())

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *API) ListDestinations(w http.ResponseWriter, r *http.Request) {
	destinationName := r.URL.Query().Get("name")
	destinationKind := r.URL.Query().Get("kind")

	var destinations []Destination
	if err := a.db.Find(&destinations, &Destination{Name: destinationName, Kind: destinationKind}).Error; err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not list destinations")

		return
	}

	results := make([]api.Destination, 0)
	for _, d := range destinations {
		results = append(results, d.marshal())
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(results); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not list destinations")
	}
}

func (a *API) GetDestination(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	destinationId := vars["id"]
	if destinationId == "" {
		sendAPIError(w, http.StatusBadRequest, "Path parameter \"id\" is required")

		return
	}

	var destination Destination
	if err := a.db.First(&destination, &Destination{Id: destinationId}).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logging.L.Debug(err.Error())
			sendAPIError(w, http.StatusNotFound, fmt.Sprintf("Could not find destination ID \"%s\"", destinationId))
		} else {
			logging.L.Error(err.Error())
			sendAPIError(w, http.StatusBadRequest, fmt.Sprintf("Could not find destination ID \"%s\"", destinationId))
		}

		return
	}

	result := destination.marshal()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not list destinations")
	}
}

func (a *API) CreateDestination(w http.ResponseWriter, r *http.Request) {
	_, err := extractAPIKey(r.Context())
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusUnauthorized, "unauthorized")

		return
	}

	var body api.DestinationCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	var destination Destination

	err = a.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where(&Destination{Name: body.Name}).FirstOrCreate(&destination)
		if result.Error != nil {
			return result.Error
		}
		destination.Name = body.Name
		destination.Kind = DestinationKindKubernetes
		destination.KubernetesCa = body.Kubernetes.Ca
		destination.KubernetesEndpoint = body.Kubernetes.Endpoint
		return tx.Save(&destination).Error
	})
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, err.Error())

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(destination.marshal()); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not create destination")
	}
}

func (a *API) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keyName := r.URL.Query().Get("name")

	var keys []APIKey

	err := a.db.Find(&keys, &APIKey{Name: keyName}).Error
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not list keys")

		return
	}

	results := make([]api.InfraAPIKey, 0)
	for _, k := range keys {
		results = append(results, k.marshal())
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(results); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not list api-keys")
	}
}

func (a *API) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		sendAPIError(w, http.StatusBadRequest, "API key ID must be specified")
	}

	if err := a.db.Delete(&APIKey{Id: id}).Error; err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, err.Error())

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *API) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var body api.InfraAPIKeyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.ToLower(body.Name) == engineAPIKeyName || strings.ToLower(body.Name) == rootAPIKeyName {
		// this name is used for the default API key that engines use to connect to Infra
		sendAPIError(w, http.StatusBadRequest, fmt.Sprintf("cannot create an API key with the name %s, this name is reserved", body.Name))
		return
	}

	var apiKey APIKey

	err := a.db.Transaction(func(tx *gorm.DB) error {
		tx.First(&apiKey, &APIKey{Name: body.Name})
		if apiKey.Id != "" {
			return ErrExistingKey
		}

		apiKey.Name = body.Name
		var permissions string
		for _, p := range body.Permissions {
			permissions += " " + string(p)
		}
		if len(strings.ReplaceAll(permissions, " ", "")) == 0 {
			return ErrKeyPermissionsNotFound
		}
		apiKey.Permissions = permissions
		return tx.Create(&apiKey).Error
	})
	if err != nil {
		logging.L.Error(err.Error())

		if errors.Is(err, ErrExistingKey) {
			sendAPIError(w, http.StatusNotFound, err.Error())
			return
		}

		if errors.Is(err, ErrKeyPermissionsNotFound) {
			sendAPIError(w, http.StatusBadRequest, err.Error())
			return
		}

		sendAPIError(w, http.StatusInternalServerError, err.Error())

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(apiKey.marshalWithSecret()); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not create api-key")
	}
}

func (a *API) ListRoles(w http.ResponseWriter, r *http.Request) {
	roleName := r.URL.Query().Get("name")
	roleKind := r.URL.Query().Get("kind")
	destinationId := r.URL.Query().Get("destination")

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
		sendAPIError(w, http.StatusInternalServerError, "could not list roles")

		return
	}

	results := make([]api.Role, 0)
	for _, r := range roles {
		results = append(results, r.marshal())
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(results); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not list roles")
	}
}

func (a *API) GetRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	roleId := vars["id"]
	if roleId == "" {
		sendAPIError(w, http.StatusBadRequest, "Path parameter \"id\" is required")

		return
	}

	var role Role
	if err := a.db.Preload("Groups.Users").Preload(clause.Associations).First(&role, &Role{Id: roleId}).Error; err != nil {
		logging.L.Error(err.Error())

		if errors.Is(err, gorm.ErrRecordNotFound) {
			sendAPIError(w, http.StatusNotFound, fmt.Sprintf("Could not find role ID \"%s\"", roleId))
		} else {
			sendAPIError(w, http.StatusBadRequest, fmt.Sprintf("Could not find role ID \"%s\"", roleId))
		}

		return
	}

	result := role.marshal()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not list roles")
	}
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

func (a *API) CreateToken(w http.ResponseWriter, r *http.Request) {
	token, err := extractToken(r.Context())
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusUnauthorized, "unauthorized")

		return
	}

	var body api.TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	jwt, expiry, err := a.createJWT(*body.Destination, token.User.Email)
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not generate cred")

		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(api.Token{Token: jwt, Expires: expiry.Unix()}); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not create cred")
	}
}

func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	var body api.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	var user User

	var token Token

	switch {
	case body.Okta != nil:
		var provider Provider
		if err := a.db.Where(&Provider{Kind: ProviderKindOkta, Domain: body.Okta.Domain}).First(&provider).Error; err != nil {
			logging.L.Debug("Could not retrieve okta provider from db: " + err.Error())
			sendAPIError(w, http.StatusBadRequest, "invalid okta login information")

			return
		}

		clientSecret, err := a.registry.GetSecret(provider.ClientSecret)
		if err != nil {
			logging.L.Error("Could not retrieve okta client secret from provider: " + err.Error())
			sendAPIError(w, http.StatusInternalServerError, "invalid okta login information")

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
			sendAPIError(w, http.StatusUnauthorized, "invalid okta login information")

			return
		}

		err = a.db.Where("email = ?", email).First(&user).Error
		if err != nil {
			logging.L.Debug("Could not get user from database: " + err.Error())
			sendAPIError(w, http.StatusUnauthorized, "invalid okta login information")

			return
		}
	default:
		sendAPIError(w, http.StatusBadRequest, "invalid login information provided")
		return
	}

	secret, err := NewToken(a.db, user.Id, SessionDuration, &token)
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not create token")

		return
	}

	tokenString := token.Id + secret

	setAuthCookie(w, tokenString)

	w.Header().Set("Content-Type", "application/json")

	if err := a.t.Enqueue(analytics.Track{Event: "infra.login", UserId: user.Id}); err != nil {
		logging.S.Debug(err)
	}

	if err := json.NewEncoder(w).Encode(api.LoginResponse{Name: user.Email, Token: tokenString}); err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusInternalServerError, "could not login")
	}
}

func (a *API) Logout(w http.ResponseWriter, r *http.Request) {
	token, err := extractToken(r.Context())
	if err != nil {
		logging.L.Error(err.Error())
		sendAPIError(w, http.StatusBadRequest, "invalid token")

		return
	}

	if err := a.db.Where(&Token{UserId: token.UserId}).Delete(&Token{}).Error; err != nil {
		sendAPIError(w, http.StatusInternalServerError, "could not log out user")
		logging.L.Error(err.Error())

		return
	}

	deleteAuthCookie(w)

	if err := a.t.Enqueue(analytics.Track{Event: "infra.logout", UserId: token.UserId}); err != nil {
		logging.S.Debug(err)
	}

	w.WriteHeader(http.StatusOK)
}

func (a *API) Version(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(api.Version{Version: internal.Version}); err != nil {
		logging.S.Errorf("encode version: %w", err)
		sendAPIError(w, http.StatusInternalServerError, "could not get version")
	}
}

func (s *Provider) marshal() api.Provider {
	res := api.Provider{
		Id:           s.Id,
		Created:      s.Created,
		Updated:      s.Updated,
		ClientID:     s.ClientID,
		Domain:       s.Domain,
		ClientSecret: s.ClientSecret,
		Kind:         s.Kind,
	}

	if s.Kind == ProviderKindOkta {
		res.Okta = &api.ProviderOkta{
			APIToken: s.APIToken,
		}
	}

	return res
}

func (d *Destination) marshal() api.Destination {
	res := api.Destination{
		Name:    d.Name,
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

	return res
}

func (k *APIKey) marshal() api.InfraAPIKey {
	res := api.InfraAPIKey{
		Name:    k.Name,
		Id:      k.Id,
		Created: k.Created,
	}
	res.Permissions = marshalPermissions(k.Permissions)

	return res
}

// This function returns the secret key, it should only be used after the initial key creation
func (k *APIKey) marshalWithSecret() api.InfraAPIKeyCreateResponse {
	res := api.InfraAPIKeyCreateResponse{
		Name:    k.Name,
		Id:      k.Id,
		Created: k.Created,
		Key:     k.Key,
	}
	res.Permissions = marshalPermissions(k.Permissions)

	return res
}

func marshalPermissions(permissions string) []api.InfraAPIPermission {
	var apiPermissions []api.InfraAPIPermission

	storedPermissions := strings.Split(permissions, " ")
	for _, p := range storedPermissions {
		apiPermission, err := api.NewInfraAPIPermissionFromValue(p)
		if err != nil {
			logging.L.Error("Error converting stored permission to API permission: " + p)
			continue
		}

		apiPermissions = append(apiPermissions, *apiPermission)
	}

	return apiPermissions
}

func (r Role) marshal() api.Role {
	res := api.Role{
		Id:        r.Id,
		Created:   r.Created,
		Updated:   r.Updated,
		Name:      r.Name,
		Namespace: r.Namespace,
	}

	switch r.Kind {
	case RoleKindKubernetesRole:
		res.Kind = api.ROLE
	case RoleKindKubernetesClusterRole:
		res.Kind = api.CLUSTER_ROLE
	}

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

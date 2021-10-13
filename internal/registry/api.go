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
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/logging"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Api struct {
	db   *gorm.DB
	k8s  *kubernetes.Kubernetes
	okta Okta
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

func NewApiMux(db *gorm.DB, k8s *kubernetes.Kubernetes, okta Okta) *mux.Router {
	a := Api{
		db:   db,
		k8s:  k8s,
		okta: okta,
	}

	r := mux.NewRouter()
	v1 := r.PathPrefix("/v1").Subrouter()
	v1.Handle("/users", a.bearerAuthMiddleware(api.USERS_READ, http.HandlerFunc(a.ListUsers))).Methods(http.MethodGet)
	v1.Handle("/users/{id}", a.bearerAuthMiddleware(api.USERS_READ, http.HandlerFunc(a.GetUser))).Methods(http.MethodGet)

	v1.Handle("/groups", a.bearerAuthMiddleware(api.GROUPS_READ, http.HandlerFunc(a.ListGroups))).Methods(http.MethodGet)
	v1.Handle("/groups/{id}", a.bearerAuthMiddleware(api.GROUPS_READ, http.HandlerFunc(a.GetGroup))).Methods(http.MethodGet)

	v1.Handle("/sources", http.HandlerFunc(a.ListSources)).Methods(http.MethodGet)
	v1.Handle("/sources/{id}", http.HandlerFunc(a.GetSource)).Methods(http.MethodGet)

	v1.Handle("/destinations", a.bearerAuthMiddleware(api.DESTINATIONS_READ, http.HandlerFunc(a.ListDestinations))).Methods(http.MethodGet)
	v1.Handle("/destinations", a.bearerAuthMiddleware(api.DESTINATIONS_CREATE, http.HandlerFunc(a.CreateDestination))).Methods(http.MethodPost)
	v1.Handle("/destinations/{id}", a.bearerAuthMiddleware(api.DESTINATIONS_READ, http.HandlerFunc(a.GetDestination))).Methods(http.MethodGet)

	v1.Handle("/api-keys", a.bearerAuthMiddleware(api.API_KEYS_READ, http.HandlerFunc(a.ListApiKeys))).Methods(http.MethodGet)
	v1.Handle("/api-keys", a.bearerAuthMiddleware(api.API_KEYS_CREATE, http.HandlerFunc(a.CreateAPIKey))).Methods(http.MethodPost)
	v1.Handle("/api-keys/{id}", a.bearerAuthMiddleware(api.API_KEYS_DELETE, http.HandlerFunc(a.DeleteApiKey))).Methods(http.MethodDelete)

	v1.Handle("/tokens", a.bearerAuthMiddleware(api.TOKENS_CREATE, http.HandlerFunc(a.CreateToken))).Methods(http.MethodPost)

	v1.Handle("/roles", a.bearerAuthMiddleware(api.ROLES_READ, http.HandlerFunc(a.ListRoles))).Methods(http.MethodGet)
	v1.Handle("/roles/{id}", a.bearerAuthMiddleware(api.ROLES_READ, http.HandlerFunc(a.GetRole))).Methods(http.MethodGet)

	v1.Handle("/login", http.HandlerFunc(a.Login)).Methods(http.MethodPost)
	v1.Handle("/logout", a.bearerAuthMiddleware(api.AUTH_DELETE, http.HandlerFunc(a.Logout))).Methods(http.MethodPost)

	v1.Handle("/version", http.HandlerFunc(a.Version)).Methods(http.MethodGet)

	return r
}

func sendApiError(w http.ResponseWriter, code int, message string) {
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

func (a *Api) bearerAuthMiddleware(required api.InfraAPIPermission, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")
		raw := strings.ReplaceAll(authorization, "Bearer ", "")

		if raw == "" {
			// Fall back to checking cookies if the bearer header is not provided
			cookie, err := r.Cookie(CookieTokenName)
			if err != nil {
				logging.L.Debug("could not read token from cookie")
				sendApiError(w, http.StatusUnauthorized, "unauthorized")
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
					sendApiError(w, http.StatusForbidden, "forbidden")
				default:
					sendApiError(w, http.StatusUnauthorized, "unauthorized")
				}
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), tokenContextKey{}, token)))
			return
		case ApiKeyLen:
			var apiKey ApiKey
			if err := a.db.First(&apiKey, &ApiKey{Key: raw}).Error; err != nil {
				logging.L.Error(err.Error())
				sendApiError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			hasPermission := checkPermission(required, apiKey.Permissions)
			if !hasPermission {
				// at this point we know their key is valid, so we can present a more detailed error
				sendApiError(w, http.StatusForbidden, string(required)+" permission is required")
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), apiKeyContextKey{}, &apiKey)))
			return
		}

		logging.L.Debug("invalid token length provided")
		sendApiError(w, http.StatusUnauthorized, "unauthorized")
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

func extractAPIKey(context context.Context) (*ApiKey, error) {
	apiKey, ok := context.Value(apiKeyContextKey{}).(*ApiKey)
	if !ok {
		return nil, errors.New("apikey not found in context")
	}

	return apiKey, nil
}

func (a *Api) ListUsers(w http.ResponseWriter, r *http.Request) {
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
		sendApiError(w, http.StatusInternalServerError, "could not list users")

		return
	}

	results := make([]api.User, 0)
	for _, u := range users {
		results = append(results, dbToAPIUser(u))
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(results); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list users")
	}
}

func (a *Api) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	userId := vars["id"]
	if userId == "" {
		sendApiError(w, http.StatusBadRequest, "Path parameter \"id\" is required")
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
		sendApiError(w, http.StatusNotFound, fmt.Sprintf("Could not find user ID \"%s\"", userId))

		return
	}

	result := dbToAPIUser(user)

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, fmt.Sprintf("Could not get user \"%s\"", userId))
	}
}

func (a *Api) ListGroups(w http.ResponseWriter, r *http.Request) {
	groupName := r.URL.Query().Get("name")

	var groups []Group
	if err := a.db.Preload(clause.Associations).Find(&groups, &Group{Name: groupName}).Error; err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list groups")

		return
	}

	results := make([]api.Group, 0)
	for _, g := range groups {
		results = append(results, dbToAPIGroup(g))
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(results); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list groups")
	}
}

func (a *Api) GetGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	groupId := vars["id"]
	if groupId == "" {
		sendApiError(w, http.StatusBadRequest, "Path parameter \"id\" is required")
	}

	var group Group
	if err := a.db.Preload(clause.Associations).First(&group, &Group{Id: groupId}).Error; err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusNotFound, "could not list groups")

		return
	}

	result := dbToAPIGroup(group)

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list groups")
	}
}

func (a *Api) ListSources(w http.ResponseWriter, r *http.Request) {
	sourceType := r.URL.Query().Get("type")

	var sources []Source
	if err := a.db.Find(&sources, &Source{Type: sourceType}).Error; err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list sources")

		return
	}

	results := make([]api.Source, 0)
	for _, s := range sources {
		results = append(results, dbToAPISource(s))
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(results); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list sources")
	}
}

func (a *Api) GetSource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	sourceId := vars["id"]
	if sourceId == "" {
		sendApiError(w, http.StatusBadRequest, "Path parameter \"id\" is required")
	}

	var source Source
	if err := a.db.First(&source, &Source{Id: sourceId}).Error; err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusNotFound, "could not list sources")

		return
	}

	result := dbToAPISource(source)

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list sources")
	}
}

func (a *Api) ListDestinations(w http.ResponseWriter, r *http.Request) {
	destinationName := r.URL.Query().Get("name")
	destinationType := r.URL.Query().Get("type")

	var destinations []Destination
	if err := a.db.Find(&destinations, &Destination{Name: destinationName, Type: destinationType}).Error; err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list destinations")

		return
	}

	results := make([]api.Destination, 0)
	for _, d := range destinations {
		results = append(results, dbToAPIdestination(d))
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(results); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list destinations")
	}
}

func (a *Api) GetDestination(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	destinationId := vars["id"]
	if destinationId == "" {
		sendApiError(w, http.StatusBadRequest, "Path parameter \"id\" is required")
	}

	var destination Destination
	if err := a.db.First(&destination, &Destination{Id: destinationId}).Error; err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusNotFound, "could not list destinations")

		return
	}

	result := dbToAPIdestination(destination)

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list destinations")
	}
}

func (a *Api) CreateDestination(w http.ResponseWriter, r *http.Request) {
	_, err := extractAPIKey(r.Context())
	if err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusUnauthorized, "unauthorized")

		return
	}

	var body api.DestinationCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendApiError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendApiError(w, http.StatusBadRequest, err.Error())
		return
	}

	var destination Destination

	err = a.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where(&Destination{Name: body.Name}).FirstOrCreate(&destination)
		if result.Error != nil {
			return result.Error
		}
		destination.Name = body.Name
		destination.Type = DestinationTypeKubernetes
		destination.KubernetesCa = body.Kubernetes.Ca
		destination.KubernetesEndpoint = body.Kubernetes.Endpoint
		return tx.Save(&destination).Error
	})
	if err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, err.Error())

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(dbToAPIdestination(destination)); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not create destination")
	}
}

func (a *Api) ListApiKeys(w http.ResponseWriter, r *http.Request) {
	keyName := r.URL.Query().Get("name")

	var keys []ApiKey

	err := a.db.Find(&keys, &ApiKey{Name: keyName}).Error
	if err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list keys")

		return
	}

	results := make([]api.InfraAPIKey, 0)
	for _, k := range keys {
		results = append(results, dbToAPIKey(k))
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(results); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list api-keys")
	}
}

func (a *Api) DeleteApiKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		sendApiError(w, http.StatusBadRequest, "ApiKey id must be specified")
	}

	err := a.db.Transaction(func(tx *gorm.DB) error {
		var existingKey ApiKey
		tx.First(&existingKey, &ApiKey{Id: id})
		if existingKey.Id == "" {
			return ErrExistingKey
		}

		tx.Delete(&existingKey)

		return nil
	})
	if err != nil {
		logging.L.Error(err.Error())

		if errors.Is(err, ErrExistingKey) {
			sendApiError(w, http.StatusNotFound, err.Error())
			return
		}

		sendApiError(w, http.StatusInternalServerError, err.Error())

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *Api) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var body api.InfraAPIKeyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendApiError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendApiError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.ToLower(body.Name) == engineApiKeyName || strings.ToLower(body.Name) == rootAPIKeyName {
		// this name is used for the default API key that engines use to connect to the registry
		sendApiError(w, http.StatusBadRequest, fmt.Sprintf("cannot create an API key with the name %s, this name is reserved", body.Name))
		return
	}

	var apiKey ApiKey

	err := a.db.Transaction(func(tx *gorm.DB) error {
		tx.First(&apiKey, &ApiKey{Name: body.Name})
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
			sendApiError(w, http.StatusNotFound, err.Error())
			return
		}

		if errors.Is(err, ErrKeyPermissionsNotFound) {
			sendApiError(w, http.StatusBadRequest, err.Error())
			return
		}

		sendApiError(w, http.StatusInternalServerError, err.Error())

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(dbToApiKeyWithSecret(&apiKey)); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not create api-key")
	}
}

func (a *Api) ListRoles(w http.ResponseWriter, r *http.Request) {
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
		sendApiError(w, http.StatusInternalServerError, "could not list roles")

		return
	}

	results := make([]api.Role, 0)
	for _, r := range roles {
		results = append(results, dbToAPIRole(r))
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(results); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list roles")
	}
}

func (a *Api) GetRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	roleId := vars["id"]
	if roleId == "" {
		sendApiError(w, http.StatusBadRequest, "Path parameter \"id\" is required")
	}

	var role Role
	if err := a.db.Preload("Groups.Users").Preload(clause.Associations).First(&role, &Role{Id: roleId}).Error; err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list roles")

		return
	}

	result := dbToAPIRole(role)

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list roles")
	}
}

var signatureAlgFromKeyAlgorithm = map[string]string{
	"ED25519": "EdDSA", // elliptic curve 25519
}

func (a *Api) createJWT(destination, email string) (rawJWT string, expiry time.Time, err error) {
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

func (a *Api) CreateToken(w http.ResponseWriter, r *http.Request) {
	token, err := extractToken(r.Context())
	if err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusUnauthorized, "unauthorized")

		return
	}

	var body api.TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendApiError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendApiError(w, http.StatusBadRequest, err.Error())
		return
	}

	jwt, expiry, err := a.createJWT(*body.Destination, token.User.Email)
	if err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not generate cred")

		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(api.Token{Token: jwt, Expires: expiry.Unix()}); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not create cred")
	}
}

func (a *Api) Login(w http.ResponseWriter, r *http.Request) {
	var body api.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendApiError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendApiError(w, http.StatusBadRequest, err.Error())
		return
	}

	var user User

	var token Token

	switch {
	case body.Okta != nil:
		var source Source
		if err := a.db.Where(&Source{Type: SourceTypeOkta, Domain: body.Okta.Domain}).First(&source).Error; err != nil {
			logging.L.Debug("Could not retrieve okta source from db: " + err.Error())
			sendApiError(w, http.StatusBadRequest, "invalid okta login information")

			return
		}

		clientSecret, err := a.k8s.GetSecret(source.ClientSecret)
		if err != nil {
			logging.L.Error("Could not retrieve okta client secret from kubernetes: " + err.Error())
			sendApiError(w, http.StatusInternalServerError, "invalid okta login information")

			return
		}

		email, err := a.okta.EmailFromCode(
			body.Okta.Code,
			source.Domain,
			source.ClientId,
			clientSecret,
		)
		if err != nil {
			logging.L.Debug("Could not extract email from okta info: " + err.Error())
			sendApiError(w, http.StatusUnauthorized, "invalid okta login information")

			return
		}

		err = a.db.Where("email = ?", email).First(&user).Error
		if err != nil {
			logging.L.Debug("Could not get user from database: " + err.Error())
			sendApiError(w, http.StatusUnauthorized, "invalid okta login information")

			return
		}
	default:
		sendApiError(w, http.StatusBadRequest, "invalid login information provided")
		return
	}

	secret, err := NewToken(a.db, user.Id, SessionDuration, &token)
	if err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not create token")

		return
	}

	tokenString := token.Id + secret

	setAuthCookie(w, tokenString)

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(api.LoginResponse{Name: user.Email, Token: tokenString}); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not login")
	}
}

func (a *Api) Logout(w http.ResponseWriter, r *http.Request) {
	token, err := extractToken(r.Context())
	if err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusBadRequest, "invalid token")

		return
	}

	if err := a.db.Where(&Token{UserId: token.UserId}).Delete(&Token{}).Error; err != nil {
		sendApiError(w, http.StatusInternalServerError, "could not log out user")
		logging.L.Error(err.Error())

		return
	}

	deleteAuthCookie(w)

	w.WriteHeader(http.StatusOK)
}

func (a *Api) Version(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(api.Version{Version: internal.Version}); err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not get version")
	}
}

func dbToAPISource(s Source) api.Source {
	res := api.Source{
		Id:      s.Id,
		Created: s.Created,
		Updated: s.Updated,
	}

	if s.Type == SourceTypeOkta {
		res.Okta = &api.SourceOkta{
			ClientId: s.ClientId,
			Domain:   s.Domain,
		}
	}

	return res
}

func dbToAPIdestination(d Destination) api.Destination {
	res := api.Destination{
		Name:    d.Name,
		Id:      d.Id,
		Created: d.Created,
		Updated: d.Updated,
	}

	if d.Type == DestinationTypeKubernetes {
		res.Kubernetes = &api.DestinationKubernetes{
			Ca:       d.KubernetesCa,
			Endpoint: d.KubernetesEndpoint,
		}
	}

	return res
}

func dbToAPIKey(k ApiKey) api.InfraAPIKey {
	res := api.InfraAPIKey{
		Name:    k.Name,
		Id:      k.Id,
		Created: k.Created,
	}
	res.Permissions = dbToInfraAPIPermissions(k.Permissions)

	return res
}

// This function returns the secret key, it should only be used after the initial key creation
func dbToApiKeyWithSecret(k *ApiKey) api.InfraAPIKeyCreateResponse {
	res := api.InfraAPIKeyCreateResponse{
		Name:    k.Name,
		Id:      k.Id,
		Created: k.Created,
		Key:     k.Key,
	}
	res.Permissions = dbToInfraAPIPermissions(k.Permissions)

	return res
}

func dbToInfraAPIPermissions(permissions string) []api.InfraAPIPermission {
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

func dbToAPIRole(r Role) api.Role {
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
		res.Users = append(res.Users, dbToAPIUser(u))
	}

	for _, g := range r.Groups {
		res.Groups = append(res.Groups, dbToAPIGroup(g))
	}

	res.Destination = dbToAPIdestination(r.Destination)

	return res
}

func dbToAPIUser(u User) api.User {
	res := api.User{
		Id:      u.Id,
		Email:   u.Email,
		Created: u.Created,
		Updated: u.Updated,
	}

	for _, g := range u.Groups {
		res.Groups = append(res.Groups, dbToAPIGroup(g))
	}

	for _, r := range u.Roles {
		res.Roles = append(res.Roles, dbToAPIRole(r))
	}

	return res
}

func dbToAPIGroup(g Group) api.Group {
	res := api.Group{
		Id:      g.Id,
		Created: g.Created,
		Updated: g.Updated,
		Name:    g.Name,
	}

	for _, u := range g.Users {
		res.Users = append(res.Users, dbToAPIUser(u))
	}

	for _, r := range g.Roles {
		res.Roles = append(res.Roles, dbToAPIRole(r))
	}

	return res
}

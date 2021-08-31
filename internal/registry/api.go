package registry

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/version"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gorm.io/gorm"
)

type Api struct {
	db   *gorm.DB
	k8s  *kubernetes.Kubernetes
	okta Okta
}

var validate *validator.Validate = validator.New()

func NewApiMux(db *gorm.DB, k8s *kubernetes.Kubernetes, okta Okta) *mux.Router {
	a := Api{
		db:   db,
		k8s:  k8s,
		okta: okta,
	}

	r := mux.NewRouter()
	v1 := r.PathPrefix("/v1").Subrouter()
	v1.Handle("/users", a.bearerAuthMiddleware(http.HandlerFunc(a.ListUsers))).Methods("GET")
	v1.Handle("/groups", a.bearerAuthMiddleware(http.HandlerFunc(a.ListGroups))).Methods("GET")
	v1.Handle("/sources", http.HandlerFunc(a.ListSources)).Methods("GET")
	v1.Handle("/destinations", a.bearerAuthMiddleware(http.HandlerFunc(a.ListDestinations))).Methods("GET")
	v1.Handle("/destinations", a.bearerAuthMiddleware(http.HandlerFunc(a.CreateDestination))).Methods("POST")
	v1.Handle("/creds", a.bearerAuthMiddleware(http.HandlerFunc(a.CreateCred))).Methods("POST")
	v1.Handle("/roles", a.bearerAuthMiddleware(http.HandlerFunc(a.ListRoles))).Methods("GET")
	v1.Handle("/apikeys", a.bearerAuthMiddleware(http.HandlerFunc(a.ListApiKeys))).Methods("GET")
	v1.Handle("/login", http.HandlerFunc(a.Login)).Methods("POST")
	v1.Handle("/logout", a.bearerAuthMiddleware(http.HandlerFunc(a.Logout))).Methods("POST")
	v1.Handle("/signup", http.HandlerFunc(a.Signup)).Methods("POST")
	v1.Handle("/version", http.HandlerFunc(a.Version)).Methods("GET")
	v1.Handle("/status", http.HandlerFunc(a.Status)).Methods("GET")
	return r
}

func sendApiError(w http.ResponseWriter, code int, message string) {
	err := api.Error{
		Code:    int32(code),
		Message: message,
	}
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(err)
}

type tokenContextKey struct{}
type apiKeyContextKey struct{}

func (a *Api) bearerAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")
		raw := strings.Replace(authorization, "Bearer ", "", -1)

		if raw == "" {
			// Backfall to checking cookies if the bearer header is not provided
			cookie, err := r.Cookie(CookieTokenName)
			if err != nil {
				logging.L.Debug("could not read token from cookie")
				sendApiError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			raw = cookie.Value
		}

		switch len(raw) {
		case TOKEN_LEN:
			token, err := ValidateAndGetToken(a.db, raw)
			if err != nil {
				logging.L.Debug(err.Error())
				sendApiError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), tokenContextKey{}, token)))
			return
		case API_KEY_LEN:
			var apiKey ApiKey
			if err := a.db.First(&apiKey, &ApiKey{Key: raw}).Error; err != nil {
				logging.L.Debug(err.Error())
				sendApiError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), apiKeyContextKey{}, &apiKey)))
			return
		}

		logging.L.Debug("invalid token length provided")
		sendApiError(w, http.StatusUnauthorized, "unauthorized")
	})
}

func extractToken(context context.Context) (*Token, error) {
	token, ok := context.Value(tokenContextKey{}).(*Token)
	if !ok {
		return nil, errors.New("token not found in context")
	}

	return token, nil
}

func (a *Api) ListUsers(w http.ResponseWriter, r *http.Request) {
	var users []User
	if err := a.db.Find(&users).Error; err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list users")
		return
	}

	results := make([]api.User, 0)
	for _, u := range users {
		results = append(results, dbToApiUser(&u))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (a *Api) ListGroups(w http.ResponseWriter, r *http.Request) {
	_, err := extractToken(r.Context())
	if err != nil {
		logging.L.Debug(err.Error())
		sendApiError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var groups []Group
	if err := a.db.Find(&groups).Error; err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list users")
		return
	}

	results := make([]api.Group, 0)
	for _, g := range groups {
		results = append(results, dbToApiGroup(&g))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (a *Api) ListSources(w http.ResponseWriter, r *http.Request) {
	var sources []Source
	err := a.db.Find(&sources).Error
	if err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list sources")
		return
	}

	var results []api.Source
	for _, s := range sources {
		results = append(results, dbToApiSource(&s))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (a *Api) ListDestinations(w http.ResponseWriter, r *http.Request) {
	_, err := extractToken(r.Context())
	if err != nil {
		logging.L.Debug(err.Error())
		sendApiError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var destinations []Destination
	if err := a.db.Find(&destinations).Error; err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list destinations")
		return
	}

	results := make([]api.Destination, 0)
	for _, d := range destinations {
		results = append(results, dbToApiDestination(&d))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (a *Api) CreateDestination(w http.ResponseWriter, r *http.Request) {
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
	err := a.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where(&Destination{Name: body.Name}).FirstOrCreate(&destination)
		if result.Error != nil {
			return result.Error
		}
		destination.Name = body.Name
		destination.Type = DESTINATION_TYPE_KUBERNERNETES
		destination.KubernetesCa = body.Kubernetes.Ca
		destination.KubernetesEndpoint = body.Kubernetes.Endpoint
		destination.KubernetesNamespace = body.Kubernetes.Namespace
		destination.KubernetesSaToken = body.Kubernetes.SaToken
		return tx.Save(&destination).Error
	})
	if err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dbToApiDestination(&destination))
	w.WriteHeader(http.StatusCreated)
}

func (a *Api) ListRoles(w http.ResponseWriter, r *http.Request) {
	destinationId := r.URL.Query().Get("destinationId")

	var roles []Role
	err := a.db.Preload("Destination").Preload("Groups").Preload("Users").Find(&roles, &Role{DestinationId: destinationId}).Error
	if err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list roles")
		return
	}

	// build the response which unifies the relation of group and directly related users to the role
	results := make([]api.Role, 0)
	err = a.db.Transaction(func(tx *gorm.DB) error {
		for _, r := range roles {
			// avoid duplicate users being added to the response by mapping based on user ID
			rUsers := make(map[string]User)
			for _, rUser := range r.Users {
				rUsers[rUser.Id] = rUser
			}

			// add any group users associated with the role now
			for _, g := range r.Groups {
				var gUsers []User
				err := tx.Model(&g).Association("Users").Find(&gUsers)
				if err != nil {
					return err
				}

				for _, gUser := range gUsers {
					rUsers[gUser.Id] = gUser
				}
			}

			// set the role users to the unified role/group users
			var users []User
			for _, u := range rUsers {
				users = append(users, u)
			}
			r.Users = users
			results = append(results, dbToApiRole(&r))
		}

		return nil
	})
	if err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list roles")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (a *Api) ListApiKeys(w http.ResponseWriter, r *http.Request) {
	token, err := extractToken(r.Context())
	if err != nil {
		logging.L.Debug(err.Error())
		sendApiError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if !token.User.Admin {
		logging.L.Debug("user is not an admin")
		sendApiError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var apikeys []ApiKey
	if err := a.db.Find(&apikeys).Error; err != nil {
		logging.L.Error(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not list apikeys")
		return
	}

	results := make([]api.ApiKey, 0)
	for _, a := range apikeys {
		results = append(results, dbToApiApiKey(&a))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (a *Api) createJWT(email string) (string, time.Time, error) {
	var settings Settings
	err := a.db.First(&settings).Error
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

func (a *Api) CreateCred(w http.ResponseWriter, r *http.Request) {
	token, err := extractToken(r.Context())
	if err != nil {
		logging.L.Debug(err.Error())
		sendApiError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	jwt, expiry, err := a.createJWT(token.User.Email)
	if err != nil {
		sendApiError(w, http.StatusInternalServerError, "could not generate cred")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(api.Cred{
		Token:   jwt,
		Expires: expiry.Unix(),
	})
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
		if err := a.db.Where(&Source{Type: SOURCE_TYPE_OKTA, Domain: body.Okta.Domain}).First(&source).Error; err != nil {
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

	case body.Infra != nil:
		if err := a.db.Where("email = ?", body.Infra.Email).First(&user).Error; err != nil {
			logging.L.Debug("User failed to login with unknown email")
			sendApiError(w, http.StatusUnauthorized, "invalid login information")
			return
		}

		if err := bcrypt.CompareHashAndPassword(user.Password, []byte(body.Infra.Password)); err != nil {
			logging.L.Debug("User failed to login due to invalid password")
			sendApiError(w, http.StatusUnauthorized, "invalid login information")
			return
		}
	default:
		sendApiError(w, http.StatusBadRequest, "invalid login information provided")
		return
	}

	secret, err := NewToken(a.db, user.Id, &token)
	if err != nil {
		logging.L.Debug(err.Error())
		sendApiError(w, http.StatusInternalServerError, "could not create token")
		return
	}

	tokenString := token.Id + secret

	setAuthCookie(w, tokenString)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Token string `json:"token"`
	}{
		Token: tokenString,
	})
}

func (a *Api) Logout(w http.ResponseWriter, r *http.Request) {
	token, err := extractToken(r.Context())
	if err != nil {
		logging.L.Debug(err.Error())
		sendApiError(w, http.StatusBadRequest, "invalid token")
		return
	}

	if err := a.db.Where(&Token{UserId: token.UserId}).Delete(&Token{}).Error; err != nil {
		sendApiError(w, http.StatusInternalServerError, "could not log out user")
		logging.L.Debug(err.Error())
		return
	}

	deleteAuthCookie(w)

	w.WriteHeader(http.StatusOK)
}

func (a *Api) Signup(w http.ResponseWriter, r *http.Request) {
	var body api.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendApiError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendApiError(w, http.StatusBadRequest, err.Error())
		return
	}

	var token Token
	var secret string
	err := a.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		err := tx.Where(&User{Admin: true}).Find(&[]User{}).Count(&count).Error
		if err != nil {
			logging.L.Debug("Could not lookup admin users in the database")
			return err
		}

		if count > 0 {
			return errors.New("admin already exists")
		}

		var infraSource Source
		if err := tx.Where(&Source{Type: SOURCE_TYPE_INFRA}).First(&infraSource).Error; err != nil {
			return err
		}

		var user User
		if err := infraSource.CreateUser(tx, &user, body.Email, body.Password, true); err != nil {
			return err
		}

		secret, err = NewToken(tx, user.Id, &token)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		logging.L.Debug(err.Error())
		sendApiError(w, http.StatusBadRequest, "could not create user")
		return
	}

	tokenString := token.Id + secret
	setAuthCookie(w, tokenString)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Token string `json:"token"`
	}{
		Token: tokenString,
	})
}

func (a *Api) Version(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(api.Version{Version: version.Version})
}

func (a *Api) Status(w http.ResponseWriter, r *http.Request) {
	var count int64
	err := a.db.Where(&User{Admin: true}).Find(&[]User{}).Count(&count).Error
	if err != nil {
		sendApiError(w, http.StatusInternalServerError, "could not retrieve status")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(api.Status{Admin: count > 0})
}

func dbToApiSource(s *Source) api.Source {
	res := api.Source{
		Id: s.Id,
	}

	switch s.Type {
	case SOURCE_TYPE_OKTA:
		res.Okta = &api.SourceOkta{
			ClientId: s.ClientId,
			Domain:   s.Domain,
		}
	}

	return res
}

func dbToApiDestination(d *Destination) api.Destination {
	res := api.Destination{
		Name:    d.Name,
		Id:      d.Id,
		Created: d.Created,
		Updated: d.Updated,
	}

	switch d.Type {
	case DESTINATION_TYPE_KUBERNERNETES:
		res.Kubernetes = &api.DestinationKubernetes{
			Ca:        d.KubernetesCa,
			Endpoint:  d.KubernetesEndpoint,
			Namespace: d.KubernetesNamespace,
			SaToken:   d.KubernetesSaToken,
		}
	}

	return res
}

func dbToApiRole(r *Role) api.Role {
	res := api.Role{
		Id:          r.Id,
		Created:     r.Created,
		Updated:     r.Updated,
		Name:        r.Name,
		Destination: dbToApiDestination(&r.Destination),
	}

	switch r.Kind {
	case ROLE_KIND_K8S_ROLE:
		res.Kind = api.ROLE
	case ROLE_KIND_K8S_CLUSTER_ROLE:
		res.Kind = api.CLUSTER_ROLE
	}

	for _, u := range r.Users {
		res.Users = append(res.Users, dbToApiUser(&u))
	}

	return res
}

func dbToApiUser(u *User) api.User {
	return api.User{
		Id:      u.Id,
		Email:   u.Email,
		Created: u.Created,
		Updated: u.Updated,
		Admin:   u.Admin,
	}
}

func dbToApiGroup(g *Group) api.Group {
	return api.Group{
		Id:      g.Id,
		Created: g.Created,
		Updated: g.Updated,
		Name:    g.Name,
	}
}

func dbToApiApiKey(a *ApiKey) api.ApiKey {
	return api.ApiKey{
		Id:      a.Id,
		Created: a.Created,
		Updated: a.Updated,
		Name:    a.Name,
		Key:     a.Key,
	}
}

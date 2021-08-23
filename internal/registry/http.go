package registry

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/version"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gorm.io/gorm"
)

var (
	CookieTokenName = "token"
	CookieLoginName = "login"
)

var validate *validator.Validate = validator.New()

type ApiServer struct {
	api.ServerInterface
	okta   Okta
	db     *gorm.DB
	logger *zap.Logger
	k8s    *kubernetes.Kubernetes
}

func sendApiError(w http.ResponseWriter, code int, message string) {
	err := api.Error{
		Code:    int32(code),
		Message: message,
	}
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(err)
}

func ZapLoggerHttpMiddleware(logger *zap.Logger, next http.Handler) http.HandlerFunc {
	if logger == nil {
		return next.ServeHTTP
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		t1 := time.Now()
		next.ServeHTTP(ww, r)
		logger.Info("finished http method call",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", ww.Status()),
			zap.String("proto", r.Proto),
			zap.Duration("time_ms", time.Since(t1)),
		)
	}
}

func (as *ApiServer) loginRedirectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ext := filepath.Ext(r.URL.Path)
		if ext != "" && ext != ".html" {
			next.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/_next") {
			next.ServeHTTP(w, r)
			return
		}

		token, tokenCookieErr := r.Cookie(CookieTokenName)
		if tokenCookieErr != nil && !errors.Is(tokenCookieErr, http.ErrNoCookie) {
			as.logger.Error(tokenCookieErr.Error())
			return
		}

		login, loginCookieErr := r.Cookie(CookieLoginName)
		if loginCookieErr != nil && !errors.Is(loginCookieErr, http.ErrNoCookie) {
			as.logger.Error(loginCookieErr.Error())
			return
		}

		// If the login or token cookie are missing, then redirect to /login or /signup based on the current status
		if errors.Is(loginCookieErr, http.ErrNoCookie) || errors.Is(tokenCookieErr, http.ErrNoCookie) {
			deleteAuthCookie(w)

			adminExists := as.db.Where(&User{Admin: true}).Find(&[]User{}).RowsAffected > 0
			if !adminExists && !strings.HasPrefix(r.URL.Path, "/signup") {
				http.Redirect(w, r, "/signup", http.StatusTemporaryRedirect)
				return
			} else if adminExists && !strings.HasPrefix(r.URL.Path, "/login") {
				params := url.Values{}
				path := "/login"

				next := ""
				if r.URL.Path != "/" {
					next += r.URL.Path
				}
				if r.URL.RawQuery != "" {
					next += "?" + r.URL.RawQuery
				}

				if next != "" {
					params.Add("next", next)
					path = "/login?" + params.Encode()
				}

				http.Redirect(w, r, path, http.StatusTemporaryRedirect)
				return
			}
		}

		// If the cookies exist, then validate their values and redirect to / or follow any ?next= query parameter
		if token != nil && login != nil {
			_, err := ValidateAndGetToken(as.db, token.Value)
			if err != nil {
				deleteAuthCookie(w)
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
				return
			}

			if login.Value != "1" {
				deleteAuthCookie(w)
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
				return
			}

			if strings.HasPrefix(r.URL.Path, "/login") || strings.HasPrefix(r.URL.Path, "/signup") {
				keys, ok := r.URL.Query()["next"]
				if !ok || len(keys[0]) < 1 {
					http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
					return
				} else {
					http.Redirect(w, r, keys[0], http.StatusTemporaryRedirect)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (as *ApiServer) createJWT(email string) (string, time.Time, error) {
	var settings Settings
	err := as.db.First(&settings).Error
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

func (as *ApiServer) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (as *ApiServer) WellKnownJWKs(w http.ResponseWriter, r *http.Request) {
	var settings Settings
	err := as.db.First(&settings).Error
	if err != nil {
		http.Error(w, "could not get JWKs", http.StatusInternalServerError)
		return
	}

	var pubKey jose.JSONWebKey
	err = pubKey.UnmarshalJSON(settings.PublicJWK)
	if err != nil {
		http.Error(w, "could not get JWKs", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Keys []jose.JSONWebKey `json:"keys"`
	}{
		[]jose.JSONWebKey{pubKey},
	})
}

func (as *ApiServer) ListUsers(w http.ResponseWriter, r *http.Request) {
	user, err := as.VerifyToken(r)
	if err != nil {
		sendApiError(w, http.StatusUnauthorized, "unauthorized")
		as.logger.Debug(err.Error())
		return
	}

	if !user.Admin {
		sendApiError(w, http.StatusUnauthorized, "unauthorized")
		as.logger.Debug("user is not an admin")
		return
	}

	var users []User
	err = as.db.Find(&users).Error
	if err != nil {
		sendApiError(w, 502, err.Error())
		return
	}

	var results []api.User
	for _, u := range users {
		results = append(results, dbToApiUser(&u))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (as *ApiServer) ListSources(w http.ResponseWriter, r *http.Request) {
	var sources []Source
	err := as.db.Find(&sources).Error
	if err != nil {
		sendApiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var results []api.Source
	for _, s := range sources {
		results = append(results, dbToApiSource(&s))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (as *ApiServer) ListDestinations(w http.ResponseWriter, r *http.Request) {
	_, err := as.VerifyToken(r)
	if err != nil {
		sendApiError(w, http.StatusUnauthorized, "unauthorized")
		as.logger.Debug(err.Error())
	}

	var destinations []Destination
	err = as.db.Find(&destinations).Error
	if err != nil {
		sendApiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var results []api.Destination
	for _, d := range destinations {
		results = append(results, dbToApiDestination(&d))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (as *ApiServer) CreateDestination(w http.ResponseWriter, r *http.Request) {
	_, tokenErr := as.VerifyToken(r)
	apiKeyErr := as.VerifyApiKey(r)
	if tokenErr != nil && apiKeyErr != nil {
		sendApiError(w, http.StatusUnauthorized, "unauthorized")
		as.logger.Debug("could not authenticate user or api key")
	}

	var body api.CreateDestinationJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendApiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var destination Destination
	err := as.db.Transaction(func(tx *gorm.DB) error {
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
		sendApiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dbToApiDestination(&destination))
	w.WriteHeader(http.StatusCreated)
}

func (as *ApiServer) ListDestinationRoles(w http.ResponseWriter, r *http.Request, destinationId string) {
	_, tokenErr := as.VerifyToken(r)
	apiKeyErr := as.VerifyApiKey(r)
	if tokenErr != nil && apiKeyErr != nil {
		sendApiError(w, http.StatusUnauthorized, "unauthorized")
		as.logger.Debug("could not authenticate user or api key")
	}

	var roles []Role
	err := as.db.Preload("Destination").Preload("Groups").Preload("Users").Find(&roles, &Role{DestinationId: destinationId}).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		sendApiError(w, http.StatusBadRequest, "destination not found")
		return
	}

	if err != nil {
		sendApiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// build the response which unifies the relation of group and directly related users to the role
	var results []api.Role
	err = as.db.Transaction(func(tx *gorm.DB) error {
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
		sendApiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (as *ApiServer) CreateCred(w http.ResponseWriter, r *http.Request) {
	user, err := as.VerifyToken(r)
	if err != nil {
		sendApiError(w, http.StatusUnauthorized, "unauthorized")
		as.logger.Debug(err.Error())
	}

	token, expiry, err := as.createJWT(user.Email)
	if err != nil {
		sendApiError(w, http.StatusInternalServerError, "could not generate cred")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(api.Cred{
		Token:   token,
		Expires: expiry.Unix(),
	})
}

func (as *ApiServer) Login(w http.ResponseWriter, r *http.Request) {
	var body api.LoginJSONBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendApiError(w, http.StatusInternalServerError, err.Error())
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
		if err := as.db.Where(&Source{Type: SOURCE_TYPE_OKTA, Domain: body.Okta.Domain}).First(&source).Error; err != nil {
			as.logger.Debug("Could not retrieve okta source from db: " + err.Error())
			sendApiError(w, http.StatusBadRequest, "invalid okta login information")
			return
		}

		clientSecret, err := as.k8s.GetSecret(source.ClientSecret)
		if err != nil {
			as.logger.Error("Could not retrieve okta client secret from kubernetes: " + err.Error())
			sendApiError(w, http.StatusInternalServerError, "invalid okta login information")
			return
		}

		email, err := as.okta.EmailFromCode(
			body.Okta.Code,
			source.Domain,
			source.ClientId,
			clientSecret,
		)
		if err != nil {
			as.logger.Debug("Could not extract email from okta info: " + err.Error())
			sendApiError(w, http.StatusUnauthorized, "invalid okta login information")
			return
		}

		err = as.db.Where("email = ?", email).First(&user).Error
		if err != nil {
			as.logger.Debug("Could not get user from database: " + err.Error())
			sendApiError(w, http.StatusUnauthorized, "invalid okta login information")
			return
		}

	case body.Infra != nil:
		if err := as.db.Where("email = ?", body.Infra.Email).First(&user).Error; err != nil {
			as.logger.Debug("User failed to login with unknown email")
			sendApiError(w, http.StatusUnauthorized, "invalid login information")
			return
		}

		if err := bcrypt.CompareHashAndPassword(user.Password, []byte(body.Infra.Password)); err != nil {
			as.logger.Debug("User failed to login due to invalid password")
			sendApiError(w, http.StatusUnauthorized, "invalid login information")
			return
		}
	default:
		sendApiError(w, http.StatusUnauthorized, "invalid login information provided")
		return
	}

	secret, err := NewToken(as.db, user.Id, &token)
	if err != nil {
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

func (as *ApiServer) Logout(w http.ResponseWriter, r *http.Request) {
	user, err := as.VerifyToken(r)
	if err != nil {
		sendApiError(w, http.StatusUnauthorized, "unauthorized")
		as.logger.Debug(err.Error())
	}

	if err := as.db.Where(&Token{UserId: user.Id}).Delete(&Token{}).Error; err != nil {
		sendApiError(w, http.StatusInternalServerError, "could not log out user")
		as.logger.Debug(err.Error())
		return
	}

	deleteAuthCookie(w)

	w.WriteHeader(http.StatusOK)
}

func (as *ApiServer) Signup(w http.ResponseWriter, r *http.Request) {
	var body api.SignupJSONBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendApiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := validate.Struct(body); err != nil {
		sendApiError(w, http.StatusBadRequest, err.Error())
		return
	}

	var token Token
	var secret string

	err := as.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		err := tx.Where(&User{Admin: true}).Find(&[]User{}).Count(&count).Error
		if err != nil {
			as.logger.Debug("Could not lookup admin users in the database")
			return status.Errorf(codes.Unauthenticated, "unauthorized")
		}

		if count > 0 {
			return status.Errorf(codes.InvalidArgument, "admin user already exists")
		}

		var infraSource Source
		if err := tx.Where(&Source{Type: SOURCE_TYPE_INFRA}).First(&infraSource).Error; err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}

		var user User
		if err := infraSource.CreateUser(tx, &user, body.Email, body.Password, true); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}

		secret, err = NewToken(tx, user.Id, &token)
		if err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}

		return nil
	})
	if err != nil {
		sendApiError(w, http.StatusInternalServerError, "could not create user")
		as.logger.Debug(err.Error())
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

func (as *ApiServer) Version(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Version string `json:"version"`
	}{
		Version: version.Version,
	})
}

func (as *ApiServer) Status(w http.ResponseWriter, r *http.Request) {
	var count int64
	err := as.db.Where(&User{Admin: true}).Find(&[]User{}).Count(&count).Error
	if err != nil {
		sendApiError(w, http.StatusInternalServerError, "could not retrieve status")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Admin bool `json:"admin"`
	}{
		Admin: count > 0,
	})
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
		res.Kind = api.RoleKindRole
	case ROLE_KIND_K8S_CLUSTER_ROLE:
		res.Kind = api.RoleKindClusterRole
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

func (as *ApiServer) VerifyToken(r *http.Request) (user *User, err error) {
	authorization := r.Header.Get("Authorization")
	raw := strings.Replace(authorization, "Bearer ", "", -1)

	if raw == "" {
		return nil, errors.New("missing or invalid Authorization header")
	}

	if len(raw) != TOKEN_LEN {
		return nil, errors.New("invalid token length")
	}

	token, err := ValidateAndGetToken(as.db, raw)
	if err != nil {
		return nil, errors.New("Could not validate token: " + err.Error())
	}

	return &token.User, nil
}

func (as *ApiServer) VerifyApiKey(r *http.Request) error {
	authorization := r.Header.Get("Authorization")
	raw := strings.Replace(authorization, "Bearer ", "", -1)

	if raw == "" {
		return errors.New("missing or invalid Authorization header")
	}

	if len(raw) != API_KEY_LEN {
		return errors.New("invalid api key length")
	}

	var apiKey ApiKey
	if as.db.First(&apiKey, &ApiKey{Key: raw}).Error != nil {
		return errors.New("could not find api key")
	}

	return nil
}

func setAuthCookie(w http.ResponseWriter, token string) {
	expires := time.Now().Add(SessionDuration)
	http.SetCookie(w, &http.Cookie{
		Name:     CookieTokenName,
		Value:    token,
		Expires:  expires,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:    CookieLoginName,
		Value:   "1",
		Expires: expires,
		Path:    "/",
	})
}
func deleteAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieTokenName,
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:    CookieLoginName,
		Value:   "",
		Expires: time.Unix(0, 0),
		Path:    "/",
	})
}

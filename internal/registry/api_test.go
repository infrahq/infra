package registry

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/registry/mocks"
	"github.com/infrahq/infra/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
	kubernetesClient "k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
)

type mockSecretReader struct{}

func NewMockSecretReader() kubernetes.SecretReader {
	return &mockSecretReader{}
}
func (msr *mockSecretReader) Get(secretName string, client *kubernetesClient.Clientset) (string, error) {
	return "foo", nil
}

func addUser(db *gorm.DB, email string, password string, admin bool) (tokenId string, tokenSecret string, err error) {
	var token Token
	var secret string
	err = db.Transaction(func(tx *gorm.DB) error {
		var infraSource Source
		if err := tx.Where(&Source{Type: SOURCE_TYPE_INFRA}).First(&infraSource).Error; err != nil {
			return err
		}
		var user User

		err := infraSource.CreateUser(tx, &user, email, password, admin)
		if err != nil {
			return err
		}

		secret, err = NewToken(tx, user.Id, &token)
		if err != nil {
			return errors.New("could not create token")
		}

		return nil
	})
	if err != nil {
		return "", "", err
	}

	return token.Id, secret, nil
}

func TestBearerTokenMiddlewareDefault(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	a.bearerAuthMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareEmptyHeader(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "")

	w := httptest.NewRecorder()
	a.bearerAuthMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareEmptyHeaderBearer(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "Bearer")

	w := httptest.NewRecorder()
	a.bearerAuthMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareInvalidLength(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "Bearer hello")

	w := httptest.NewRecorder()
	a.bearerAuthMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareInvalidToken(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "Bearer "+generate.RandString(TOKEN_LEN))

	w := httptest.NewRecorder()
	a.bearerAuthMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareValidToken(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{
		db: db,
	}

	id, secret, err := addUser(db, "test@test.com", "passw0rd", false)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "Bearer "+id+secret)

	w := httptest.NewRecorder()
	a.bearerAuthMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "hello world", w.Body.String())
}

func TestBearerTokenMiddlewareInvalidApiKey(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{
		db: db,
	}

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Add("Authorization", "Bearer "+generate.RandString(API_KEY_LEN))

	w := httptest.NewRecorder()
	a.bearerAuthMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareValidApiKey(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{
		db: db,
	}

	var apiKey ApiKey
	err = db.FirstOrCreate(&apiKey, &ApiKey{Name: "default"}).Error
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "Bearer "+apiKey.Key)

	w := httptest.NewRecorder()
	a.bearerAuthMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "hello world", w.Body.String())
}

func TestLoginHandlerEmptyRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{db: db}

	r := httptest.NewRequest(http.MethodPost, "http://test.com/v1/login", nil)
	w := httptest.NewRecorder()
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginNilInfraRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{db: db}

	loginRequest := api.LoginRequest{
		Infra: nil,
	}

	bts, err := loginRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginEmptyInfraRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{db: db}

	loginRequest := api.LoginRequest{
		Infra: &api.LoginRequestInfra{},
	}

	bts, err := loginRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginInfraEmptyPassword(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{db: db}

	loginRequest := api.LoginRequest{
		Infra: &api.LoginRequestInfra{
			Email: "test@test.com",
		},
	}

	bts, err := loginRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestLoginInfraEmptyEmail(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{db: db}

	loginRequest := api.LoginRequest{
		Infra: &api.LoginRequestInfra{
			Password: "passw0rd",
		},
	}

	bts, err := loginRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginInfraSuccess(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{db: db}

	_, _, err = addUser(db, "test@test.com", "passw0rd", false)
	if err != nil {
		t.Fatal(err)
	}

	loginRequest := api.LoginRequest{
		Infra: &api.LoginRequestInfra{
			Email:    "test@test.com",
			Password: "passw0rd",
		},
	}

	bts, err := loginRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginNilOktaRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{db: db}

	loginRequest := api.LoginRequest{
		Okta: nil,
	}

	bts, err := loginRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginEmptyOktaRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{db: db}

	loginRequest := api.LoginRequest{
		Okta: &api.LoginRequestOkta{},
	}

	bts, err := loginRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginOktaMissingDomainRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{db: db}

	loginRequest := api.LoginRequest{
		Okta: &api.LoginRequestOkta{
			Code: "testcode",
		},
	}

	bts, err := loginRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginMethodOktaMissingCodeRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{db: db}

	loginRequest := api.LoginRequest{
		Okta: &api.LoginRequestOkta{
			Domain: "test.okta.com",
		},
	}

	bts, err := loginRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginMethodOkta(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	var source Source
	source.Type = "okta"
	source.ApiToken = "test-api-token/apiToken"
	source.Domain = "test.okta.com"
	source.ClientId = "test-client-id"
	source.ClientSecret = "test-client-secret/clientSecret"
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}

	var user User
	source.CreateUser(db, &user, "test@test.com", "", false)
	if err != nil {
		t.Fatal(err)
	}

	testOkta := new(mocks.Okta)
	testOkta.On("EmailFromCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("test@test.com", nil)

	testSecretReader := NewMockSecretReader()
	testConfig := &rest.Config{
		Host: "https://localhost",
	}
	testK8s := &kubernetes.Kubernetes{Config: testConfig, SecretReader: testSecretReader}

	a := &Api{db: db, okta: testOkta, k8s: testK8s}

	loginRequest := api.LoginRequest{
		Okta: &api.LoginRequestOkta{
			Domain: "test.okta.com",
			Code:   "testcode",
		},
	}

	bts, err := loginRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestSignup(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{db: db}

	signupRequest := api.SignupRequest{
		Email:    "test@test.com",
		Password: "passw0rd",
	}

	bts, err := signupRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/signup", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.Signup).ServeHTTP(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)

	var user User
	err = db.First(&user).Error
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, user.Admin, true)
	assert.Equal(t, user.Email, "test@test.com")
}

func TestSignupWithExistingAdmin(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	addUser(db, "existing@user.com", "passw0rd", true)

	a := &Api{db: db}

	signupRequest := api.SignupRequest{
		Email:    "admin@test.com",
		Password: "adminpassw0rd",
	}

	bts, err := signupRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/signup", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.Signup).ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVersion(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{db: db}

	r := httptest.NewRequest(http.MethodGet, "/v1/signup", nil)
	w := httptest.NewRecorder()
	http.HandlerFunc(a.Version).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var body api.Version
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, version.Version, body.Version)
}

func TestListRolesForClusterReturnsRolesFromConfig(t *testing.T) {
	// this in memory DB is setup in the config_test.go
	a := &Api{db: db}

	r := httptest.NewRequest(http.MethodGet, "/v1/roles", nil)
	q := r.URL.Query()
	q.Add("destinationId", clusterA.Id)
	r.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	http.HandlerFunc(a.ListRoles).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var roles []api.Role
	if err := json.NewDecoder(w.Body).Decode(&roles); err != nil {
		t.Fatal(err)
	}

	returnedUserRoles := make(map[string][]api.User)
	for _, r := range roles {
		returnedUserRoles[r.Name] = r.Users
	}

	// check default roles granted on user create
	assert.Equal(t, 3, len(returnedUserRoles["view"]))
	assert.True(t, containsUser(returnedUserRoles["view"], iosDevUser.Email))
	assert.True(t, containsUser(returnedUserRoles["view"], standardUser.Email))
	assert.True(t, containsUser(returnedUserRoles["view"], adminUser.Email))

	// roles from groups
	assert.Equal(t, 2, len(returnedUserRoles["writer"]))
	assert.True(t, containsUser(returnedUserRoles["writer"], iosDevUser.Email))
	assert.True(t, containsUser(returnedUserRoles["writer"], standardUser.Email))

	// roles from direct user assignment
	assert.Equal(t, 1, len(returnedUserRoles["admin"]))
	assert.True(t, containsUser(returnedUserRoles["admin"], adminUser.Email))
	assert.Equal(t, 1, len(returnedUserRoles["reader"]))
	assert.True(t, containsUser(returnedUserRoles["reader"], standardUser.Email))
}

func TestListRolesOnlyFindsForSpecificCluster(t *testing.T) {
	// this in memory DB is setup in the config_test.go
	a := &Api{db: db}

	r := httptest.NewRequest(http.MethodGet, "/v1/roles", nil)
	q := r.URL.Query()
	q.Add("destinationId", clusterA.Id)
	r.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	http.HandlerFunc(a.ListRoles).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var roles []api.Role
	if err := json.NewDecoder(w.Body).Decode(&roles); err != nil {
		t.Fatal(err)
	}

	unexpectedClusterIds := make(map[string]bool)
	for _, r := range roles {
		if r.Destination.Id != clusterA.Id {
			unexpectedClusterIds[r.Destination.Id] = true
		}
	}
	if len(unexpectedClusterIds) != 0 {
		var unexpectedClusters []string
		for id := range unexpectedClusterIds {
			unexpectedClusters = append(unexpectedClusters, id)
		}
		t.Errorf("ListRoles response should only contain roles for the specified cluster ID. Only expected " + clusterA.Id + " but found " + strings.Join(unexpectedClusters, ", "))
	}
}

func TestListRolesForUnknownCluster(t *testing.T) {
	// this in memory DB is setup in config_test.go
	a := &Api{db: db}

	r := httptest.NewRequest(http.MethodGet, "/v1/roles", nil)
	q := r.URL.Query()
	q.Add("destinationId", "Unknown-Cluster-ID")
	r.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	http.HandlerFunc(a.ListRoles).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var roles []api.Role
	if err := json.NewDecoder(w.Body).Decode(&roles); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 0, len(roles))
}

func containsUser(users []api.User, email string) bool {
	for _, u := range users {
		if u.Email == email {
			return true
		}
	}
	return false
}

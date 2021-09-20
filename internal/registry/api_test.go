package registry

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/registry/mocks"
	"github.com/infrahq/infra/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func addUser(db *gorm.DB, sessionDuration time.Duration) (tokenId string, tokenSecret string, err error) {
	var token Token
	var secret string
	err = db.Transaction(func(tx *gorm.DB) error {
		user := &User{Email: "test@test.com"}
		err := tx.Create(user).Error
		if err != nil {
			return err
		}

		secret, err = NewToken(tx, user.Id, sessionDuration, &token)
		if err != nil {
			return err
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
	require.NoError(t, err)

	a := &Api{
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	bearerToken, err := generate.RandString(TOKEN_LEN)
	require.NoError(t, err)
	r.Header.Add("Authorization", "Bearer "+bearerToken)

	w := httptest.NewRecorder()
	a.bearerAuthMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareExpiredToken(t *testing.T) {
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

	id, secret, err := addUser(db, time.Millisecond*1)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "Bearer "+id+secret)

	w := httptest.NewRecorder()
	a.bearerAuthMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
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

	id, secret, err := addUser(db, time.Hour*24)
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
	bearerToken, err := generate.RandString(TOKEN_LEN)
	require.NoError(t, err)
	r.Header.Add("Authorization", "Bearer "+bearerToken)

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

func TestCreateDestinationNoApiKey(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		require.NoError(t, err)
	}

	a := &Api{db: db}

	req := api.DestinationCreateRequest{
		Kubernetes: &api.DestinationKubernetes{
			Ca:        "CA",
			Endpoint:  "endpoint.net",
			Namespace: "default",
			SaToken:   "token",
		},
	}

	bts, err := req.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	a.bearerAuthMiddleware(http.HandlerFunc(a.Login)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateDestination(t *testing.T) {
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

	req := api.DestinationCreateRequest{
		Kubernetes: &api.DestinationKubernetes{
			Ca:        "CA",
			Endpoint:  "endpoint.net",
			Namespace: "default",
			SaToken:   "token",
		},
	}

	bts, err := req.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))
	r.Header.Add("Authorization", "Bearer "+apiKey.Key)

	w := httptest.NewRecorder()
	a.bearerAuthMiddleware(http.HandlerFunc(a.CreateDestination)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
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
	source.CreateUser(db, &user, "test@test.com")
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

	var body api.LoginResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test@test.com", body.Name)
	assert.NotEmpty(t, body.Token)
}

func TestVersion(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &Api{db: db}

	r := httptest.NewRequest(http.MethodGet, "/v1/version", nil)
	w := httptest.NewRecorder()
	http.HandlerFunc(a.Version).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var body api.Version
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, version.Version, body.Version)
}

func TestListRolesForDestinationReturnsRolesFromConfig(t *testing.T) {
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

func TestListRolesOnlyFindsForSpecificDestination(t *testing.T) {
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

	unexpectedDestinationIds := make(map[string]bool)
	for _, r := range roles {
		if r.Destination.Id != clusterA.Id {
			unexpectedDestinationIds[r.Destination.Id] = true
		}
	}
	if len(unexpectedDestinationIds) != 0 {
		var unexpectedDestinations []string
		for id := range unexpectedDestinationIds {
			unexpectedDestinations = append(unexpectedDestinations, id)
		}
		t.Errorf("ListRoles response should only contain roles for the specified Destination ID. Only expected " + clusterA.Id + " but found " + strings.Join(unexpectedDestinations, ", "))
	}
}

func TestListRolesForUnknownDestination(t *testing.T) {
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

func TestListGroups(t *testing.T) {
	a := &Api{db: db}

	r := httptest.NewRequest(http.MethodGet, "/v1/groups", nil)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.ListGroups).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var groups []api.Group
	if err := json.NewDecoder(w.Body).Decode(&groups); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 4, len(groups))

	groupSources := make(map[string]string)
	for _, g := range groups {
		groupSources[g.Name] = g.Source
	}
	assert.Equal(t, "okta", groupSources["heroes"])
	assert.Equal(t, "okta", groupSources["villains"])
}

func TestCreateMachineAPIKey(t *testing.T) {
	a := &Api{db: db}

	createMachineAPIKeyRequest := api.MachineAPIKeyCreateRequest{
		Name: "test-api-machine",
	}

	csr, err := createMachineAPIKeyRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/machines/apis", bytes.NewReader(csr))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.CreateMachineAPIKey).ServeHTTP(w, r)

	var body api.MachineAPIKey
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "test-api-machine", body.Name)
	assert.NotEmpty(t, body.ApiKey)
}

func TestDeleteMachineAPIKey(t *testing.T) {
	a := &Api{db: db}

	createMachineAPIKeyRequest := api.MachineAPIKeyCreateRequest{
		Name: "test-api-machine",
	}

	csr, err := createMachineAPIKeyRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/machines/apis", bytes.NewReader(csr))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.CreateMachineAPIKey).ServeHTTP(w, r)

	var body api.MachineAPIKey
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	delR := httptest.NewRequest(http.MethodDelete, "/v1/machines/"+body.Id, nil)
	delW := httptest.NewRecorder()
	http.HandlerFunc(a.DeleteMachine).ServeHTTP(delW, delR)

	assert.Equal(t, http.StatusNoContent, delW.Code)

	var machineApiKey ApiKey
	db.First(&machineApiKey, &ApiKey{Name: "test-api-machine"})
	assert.Empty(t, machineApiKey.Id, "API key associated with machine not deleted")
}

func containsUser(users []api.User, email string) bool {
	for _, u := range users {
		if u.Email == email {
			return true
		}
	}
	return false
}

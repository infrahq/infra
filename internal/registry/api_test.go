package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/registry/mocks"
	"github.com/infrahq/infra/secrets"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockSecretReader struct{}

func NewMockSecretReader() secrets.SecretStorage {
	return &mockSecretReader{}
}

func (msr *mockSecretReader) GetSecret(secretName string) ([]byte, error) {
	return []byte("abcdefghijklmnopqrstuvwx"), nil
}

func (msr *mockSecretReader) SetSecret(secretName string, secret []byte) error {
	return nil
}

func addUser(db *gorm.DB, sessionDuration time.Duration) (tokenId string, tokenSecret string, err error) {
	var (
		token  Token
		secret string
	)

	err = db.Transaction(func(tx *gorm.DB) error {
		user := &User{Email: "test@test.com"}
		err := tx.Create(user).Error
		if err != nil {
			return err
		}

		secret, err = NewToken(tx, user.Id, standardUserPermissions, sessionDuration, &token)
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
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.USERS_READ)(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareEmptyHeader(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.USERS_READ)(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareEmptyHeaderBearer(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "Bearer")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.USERS_READ)(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareInvalidLength(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "Bearer hello")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.USERS_READ)(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareInvalidToken(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	require.NoError(t, err)

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	bearerToken, err := generate.RandString(TokenLen)
	require.NoError(t, err)
	r.Header.Add("Authorization", "Bearer "+bearerToken)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.USERS_READ)(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareExpiredToken(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	id, secret, err := addUser(db, time.Millisecond*1)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "Bearer "+id+secret)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.USERS_READ)(c)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestBearerTokenMiddlewareValidTokenWrongPermissions(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	id, secret, err := addUser(db, time.Hour*24)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "Bearer "+id+secret)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.API_KEYS_CREATE)(c)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestBearerTokenMiddlewareValidToken(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	id, secret, err := addUser(db, time.Hour*24)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "Bearer "+id+secret)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.USERS_READ)(c)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestBearerTokenMiddlewareInvalidAPIKey(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	bearerToken, err := generate.RandString(TokenLen)
	require.NoError(t, err)
	r.Header.Add("Authorization", "Bearer "+bearerToken)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.USERS_READ)(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareValidAPIKey(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	var apiKey APIKey

	err = db.FirstOrCreate(&apiKey, &APIKey{Name: engineAPIKeyName, Permissions: string(api.USERS_READ)}).Error
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "Bearer "+apiKey.Key)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.USERS_READ)(c)
	require.Equal(t, w.Code, http.StatusOK)
}

func TestBearerTokenMiddlewareValidAPIKeyRootPermissions(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	var apiKey APIKey

	err = db.FirstOrCreate(&apiKey, &APIKey{Name: engineAPIKeyName, Permissions: string(api.STAR)}).Error
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "Bearer "+apiKey.Key)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.USERS_READ)(c)
	require.Equal(t, w.Code, http.StatusOK)
}

func TestBearerTokenMiddlewareValidAPIKeyWrongPermission(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	var apiKey APIKey

	err = db.FirstOrCreate(&apiKey, &APIKey{Name: engineAPIKeyName, Permissions: string(api.DESTINATIONS_READ)}).Error
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add("Authorization", "Bearer "+apiKey.Key)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.DESTINATIONS_CREATE)(c)

	var body api.Error
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, w.Code, http.StatusForbidden)
	require.Equal(t, string(api.DESTINATIONS_CREATE)+" permission is required", body.Message)
}

func TestCreateDestinationNoKind(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	var apiKey APIKey

	err = db.FirstOrCreate(&apiKey, &APIKey{Name: "default", Permissions: string(api.DESTINATIONS_CREATE)}).Error
	if err != nil {
		t.Fatal(err)
	}

	req := api.DestinationCreateRequest{
		Kubernetes: &api.DestinationKubernetes{
			Ca:       "CA",
			Endpoint: "endpoint.net",
		},
	}

	bts, err := req.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))
	r.Header.Add("Authorization", "Bearer "+apiKey.Key)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.DESTINATIONS_CREATE)(c)
	a.CreateDestination(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateDestinationBadKind(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	var apiKey APIKey

	err = db.FirstOrCreate(&apiKey, &APIKey{Name: "default", Permissions: string(api.DESTINATIONS_CREATE)}).Error
	if err != nil {
		t.Fatal(err)
	}

	req := api.DestinationCreateRequest{
		Kind: api.DestinationKind("nonexistent"),
		Kubernetes: &api.DestinationKubernetes{
			Ca:       "CA",
			Endpoint: "endpoint.net",
		},
	}

	bts, err := req.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))
	r.Header.Add("Authorization", "Bearer "+apiKey.Key)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.DESTINATIONS_CREATE)(c)
	a.CreateDestination(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateDestinationNoAPIKey(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		require.NoError(t, err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	req := api.DestinationCreateRequest{
		Kind: api.KUBERNETES,
		Kubernetes: &api.DestinationKubernetes{
			Ca:       "CA",
			Endpoint: "endpoint.net",
		},
	}

	bts, err := req.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.USERS_READ)(c)
	a.Login(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateDestination(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	var apiKey APIKey

	err = db.FirstOrCreate(&apiKey, &APIKey{Name: "default", Permissions: string(api.DESTINATIONS_CREATE)}).Error
	if err != nil {
		t.Fatal(err)
	}

	req := api.DestinationCreateRequest{
		Kind: api.KUBERNETES,
		Kubernetes: &api.DestinationKubernetes{
			Ca:       "CA",
			Endpoint: "endpoint.net",
		},
	}

	bts, err := req.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))
	r.Header.Add("Authorization", "Bearer "+apiKey.Key)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.DESTINATIONS_CREATE)(c)
	a.CreateDestination(c)
	require.Equal(t, http.StatusCreated, w.Code)
}

func TestInsertDestinationUpdatesField(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	var apiKey APIKey

	err = db.FirstOrCreate(&apiKey, &APIKey{Name: "default", Permissions: string(api.DESTINATIONS_CREATE)}).Error
	if err != nil {
		t.Fatal(err)
	}

	nodeID := "node1"

	existing := &Destination{
		NodeID:             nodeID,
		Name:               "test-destination",
		Kind:               DestinationKindKubernetes,
		KubernetesCa:       "--BEGIN CERTIFICATE--",
		KubernetesEndpoint: "example.com",
	}

	err = db.FirstOrCreate(&existing).Error
	require.NoError(t, err)

	newName := "updated-test-destination"
	newCA := "--BEGIN NEW CERTIFICATE--"
	newEndpoint := "new.example.com"

	req := api.DestinationCreateRequest{
		NodeID: nodeID,
		Name:   "updated-test-destination",
		Kind:   api.KUBERNETES,
		Kubernetes: &api.DestinationKubernetes{
			Ca:       newCA,
			Endpoint: newEndpoint,
		},
	}

	bts, err := req.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))
	r.Header.Add("Authorization", "Bearer "+apiKey.Key)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.DESTINATIONS_CREATE)(c)
	a.CreateDestination(c)
	require.Equal(t, http.StatusCreated, w.Code)

	var body api.Destination
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, nodeID, body.NodeID)
	// check that the updated fields have changed
	require.Equal(t, newName, body.Name)
	require.Equal(t, newCA, body.Kubernetes.Ca)
	require.Equal(t, newEndpoint, body.Kubernetes.Endpoint)
}

func TestLoginHandlerEmptyRequest(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodPost, "http://test.com/v1/login", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.Login(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginNilOktaRequest(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	loginRequest := api.LoginRequest{
		Okta: nil,
	}

	bts, err := loginRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.Login(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginEmptyOktaRequest(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	loginRequest := api.LoginRequest{
		Okta: &api.LoginRequestOkta{},
	}

	bts, err := loginRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.Login(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginOktaMissingDomainRequest(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

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
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.Login(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginMethodOktaMissingCodeRequest(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

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
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.Login(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginMethodOkta(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	var provider Provider
	provider.Kind = ProviderKindOkta
	provider.APIToken = "test-api-token/apiToken"
	provider.Domain = "test.okta.com"
	provider.ClientID = "test-client-id"
	provider.ClientSecret = "test-client-secret/clientSecret"

	if err := db.Create(&provider).Error; err != nil {
		t.Fatal(err)
	}

	var user User
	if err := provider.CreateUser(db, &user, "test@test.com"); err != nil {
		t.Fatal(err)
	}

	testOkta := new(mocks.Okta)
	testOkta.On("EmailFromCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("test@test.com", nil)

	telemetry, err := NewTelemetry(db)
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db:   db,
		okta: testOkta,
		t:    telemetry,
	}

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
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.Login(c)

	var body api.LoginResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "test@test.com", body.Name)
	require.NotEmpty(t, body.Token)
}

func TestVersion(t *testing.T) {
	db, err := NewSQLiteDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/version", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.Version(c)
	require.Equal(t, http.StatusOK, w.Code)

	var body api.Version
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, internal.Version, body.Version)
}

func TestListRoles(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/roles", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListRoles(c)
	require.Equal(t, http.StatusOK, w.Code)

	var roles []api.Role
	err := json.NewDecoder(w.Body).Decode(&roles)
	require.NoError(t, err)

	require.Equal(t, 8, len(roles))

	returnedUserRoles := make(map[string][]api.User)
	for _, r := range roles {
		returnedUserRoles[r.Name] = r.Users
	}

	// roles from direct user assignment
	require.Equal(t, 1, len(returnedUserRoles["admin"]))
	require.True(t, containsUser(returnedUserRoles["admin"], adminUser.Email))

	require.Equal(t, 1, len(returnedUserRoles["audit"]))
	require.True(t, containsUser(returnedUserRoles["audit"], adminUser.Email))

	require.Equal(t, 1, len(returnedUserRoles["pod-create"]))
	require.True(t, containsUser(returnedUserRoles["pod-create"], adminUser.Email))

	require.Equal(t, 0, len(returnedUserRoles["writer"]))

	require.Equal(t, 0, len(returnedUserRoles["view"]))

	returnedGroupRoles := make(map[string][]api.Group)
	for _, r := range roles {
		returnedGroupRoles[r.Name] = r.Groups
	}

	// roles from groups
	require.Equal(t, 0, len(returnedGroupRoles["admin"]))

	require.Equal(t, 1, len(returnedGroupRoles["audit"]))
	require.True(t, containsGroup(returnedGroupRoles["audit"], iosDevGroup.Name))

	require.Equal(t, 1, len(returnedGroupRoles["pod-create"]))
	require.True(t, containsGroup(returnedGroupRoles["pod-create"], iosDevGroup.Name))

	require.Equal(t, 1, len(returnedGroupRoles["writer"]))
	require.True(t, containsGroup(returnedGroupRoles["writer"], macAdminGroup.Name))
}

func TestListRolesByName(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/roles?name=admin", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListRoles(c)
	require.Equal(t, http.StatusOK, w.Code)

	var roles []api.Role
	if err := json.NewDecoder(w.Body).Decode(&roles); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 3, len(roles))

	returnedUserRoles := make(map[string][]api.User)
	for _, r := range roles {
		returnedUserRoles[r.Name] = r.Users
	}

	// roles from direct user assignment
	require.Equal(t, 1, len(returnedUserRoles["admin"]))
	require.True(t, containsUser(returnedUserRoles["admin"], adminUser.Email))

	returnedGroupRoles := make(map[string][]api.Group)
	for _, r := range roles {
		returnedGroupRoles[r.Name] = r.Groups
	}

	// roles from groups
	require.Equal(t, 0, len(returnedGroupRoles["admin"]))
}

func TestListRolesByKind(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/roles?kind=role", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListRoles(c)
	require.Equal(t, http.StatusOK, w.Code)

	var roles []api.Role
	if err := json.NewDecoder(w.Body).Decode(&roles); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 1, len(roles))

	returnedGroupRoles := make(map[string][]api.Group)
	for _, r := range roles {
		returnedGroupRoles[r.Name] = r.Groups
	}

	// roles from groups
	require.Equal(t, 1, len(returnedGroupRoles["pod-create"]))
	require.True(t, containsGroup(returnedGroupRoles["pod-create"], iosDevGroup.Name))
}

func TestListRolesByMultiple(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/roles?name=admin&kind=role", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListRoles(c)
	require.Equal(t, http.StatusOK, w.Code)

	var roles []api.Role
	if err := json.NewDecoder(w.Body).Decode(&roles); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 0, len(roles))
}

func TestListRolesForDestinationReturnsRolesFromConfig(t *testing.T) {
	// this in memory DB is setup in the config_test.go
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/roles", nil)
	q := r.URL.Query()
	q.Add("destination", clusterA.Id)
	r.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListRoles(c)
	require.Equal(t, http.StatusOK, w.Code)

	var roles []api.Role
	if err := json.NewDecoder(w.Body).Decode(&roles); err != nil {
		t.Fatal(err)
	}

	returnedUserRoles := make(map[string][]api.User)
	for _, r := range roles {
		returnedUserRoles[r.Name] = r.Users
	}

	// roles from direct user assignment
	require.Equal(t, 1, len(returnedUserRoles["admin"]))
	require.True(t, containsUser(returnedUserRoles["admin"], adminUser.Email))

	returnedGroupRoles := make(map[string][]api.Group)
	for _, r := range roles {
		returnedGroupRoles[r.Name] = r.Groups
	}

	// roles from groups
	require.Equal(t, 1, len(returnedGroupRoles["writer"]))
	require.True(t, containsGroup(returnedGroupRoles["writer"], iosDevGroup.Name))
}

func TestListRolesOnlyFindsForSpecificDestination(t *testing.T) {
	// this in memory DB is setup in the config_test.go
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/roles", nil)
	q := r.URL.Query()
	q.Add("destination", clusterA.Id)
	r.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListRoles(c)
	require.Equal(t, http.StatusOK, w.Code)

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
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/roles", nil)
	q := r.URL.Query()
	q.Add("destination", "Unknown-Cluster-ID")
	r.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListRoles(c)
	require.Equal(t, http.StatusOK, w.Code)

	var roles []api.Role
	if err := json.NewDecoder(w.Body).Decode(&roles); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 0, len(roles))
}

func TestGetRole(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	role := &Role{Name: "mpt-role"}
	if err := a.db.Create(role).Error; err != nil {
		t.Fatalf(err.Error())
	}

	defer a.db.Delete(role)

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/roles/%s", role.Id), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r
	c.Params = append(c.Params, gin.Param{Key: "id", Value: role.Id})

	a.GetRole(c)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestGetRoleEmptyID(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/roles/", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.GetRole(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetRoleNotFound(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "nonexistent"}}

	a.GetRole(c)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListGroups(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/groups", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListGroups(c)
	require.Equal(t, http.StatusOK, w.Code)

	// b, _ := ioutil.ReadAll(w.Body)
	// fmt.Println(string(b))
	// t.Fail()

	var groups []api.Group
	err := json.NewDecoder(w.Body).Decode(&groups)
	require.NoError(t, err)

	require.Equal(t, 2, len(groups))

	require.True(t, containsGroup(groups, "ios-developers"))
	require.True(t, containsGroup(groups, "mac-admins"))
}

func TestListGroupsByName(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/groups?name=ios-developers", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListGroups(c)
	require.Equal(t, http.StatusOK, w.Code)

	var groups []api.Group
	err := json.NewDecoder(w.Body).Decode(&groups)
	require.NoError(t, err)

	require.Equal(t, 1, len(groups))

	require.True(t, containsGroup(groups, "ios-developers"))
}

func TestGetGroup(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	group := &Group{Name: "mpt-group"}
	if err := a.db.Create(group).Error; err != nil {
		t.Fatalf(err.Error())
	}

	defer a.db.Delete(group)

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/groups/%s", group.Id), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r
	c.Params = append(c.Params, gin.Param{Key: "id", Value: group.Id})

	a.GetGroup(c)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestGetGroupEmptyID(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/groups/", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.GetGroup(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetGroupNotFound(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/groups/nonexistent", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "nonexistent"})

	a.GetGroup(c)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListUsers(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/users", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListUsers(c)
	require.Equal(t, http.StatusOK, w.Code)

	var users []api.User
	if err := json.NewDecoder(w.Body).Decode(&users); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 3, len(users))

	require.True(t, containsUser(users, adminUser.Email))
	require.True(t, containsUser(users, standardUser.Email))
	require.True(t, containsUser(users, iosDevUser.Email))
}

func TestListUsersByEmail(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/users?email=woz@example.com", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListUsers(c)
	require.Equal(t, http.StatusOK, w.Code)

	var users []api.User
	if err := json.NewDecoder(w.Body).Decode(&users); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 1, len(users))

	require.True(t, containsUser(users, iosDevUser.Email))
}

func TestListUsersEmpty(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/users?email=nonexistent@example.com", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListUsers(c)
	require.Equal(t, http.StatusOK, w.Code)

	var users []api.User
	if err := json.NewDecoder(w.Body).Decode(&users); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 0, len(users))
}

func TestGetUser(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	user := &User{Email: "mpt-user@infrahq.com"}
	if err := a.db.Create(user).Error; err != nil {
		t.Fatalf(err.Error())
	}

	defer a.db.Delete(user)

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/users/%s", user.Id), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r
	c.Params = append(c.Params, gin.Param{Key: "id", Value: user.Id})

	a.GetUser(c)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestGetUserEmptyID(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/users/", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.GetUser(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetUserNotFound(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/users/nonexistent", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "nonexistent"})

	a.GetUser(c)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListProviders(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/providers", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListProviders(c)
	require.Equal(t, http.StatusOK, w.Code)

	var providers []api.Provider
	if err := json.NewDecoder(w.Body).Decode(&providers); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 1, len(providers))
}

func TestListProvidersByType(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/providers?kind=okta", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListProviders(c)
	require.Equal(t, http.StatusOK, w.Code)

	var providers []api.Provider
	if err := json.NewDecoder(w.Body).Decode(&providers); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 1, len(providers))
}

func TestListProvidersEmpty(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/providers?kind=nonexistent", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListProviders(c)
	require.Equal(t, http.StatusOK, w.Code)

	var providers []api.Provider
	if err := json.NewDecoder(w.Body).Decode(&providers); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 0, len(providers))
}

func TestGetProvider(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	provider := &Provider{Kind: ProviderKindOkta}
	if err := a.db.Create(provider).Error; err != nil {
		t.Fatalf(err.Error())
	}

	defer a.db.Delete(provider)

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/providers/%s", provider.Id), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r
	c.Params = append(c.Params, gin.Param{Key: "id", Value: provider.Id})

	a.GetProvider(c)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestGetProviderEmptyID(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.GetProvider(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetProviderNotFound(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/providers/nonexistent", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "nonexistent"})

	a.GetProvider(c)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListDestinations(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/destinations", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListDestinations(c)
	require.Equal(t, http.StatusOK, w.Code)

	var destinations []api.Destination
	if err := json.NewDecoder(w.Body).Decode(&destinations); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 3, len(destinations))

	require.True(t, containsDestination(destinations, "cluster-AAA"))
	require.True(t, containsDestination(destinations, "cluster-BBB"))
	require.True(t, containsDestination(destinations, "cluster-CCC"))
}

func TestListDestinationsByName(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/destinations?name=cluster-AAA", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListDestinations(c)
	require.Equal(t, http.StatusOK, w.Code)

	var destinations []api.Destination
	if err := json.NewDecoder(w.Body).Decode(&destinations); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 1, len(destinations))

	require.True(t, containsDestination(destinations, "cluster-AAA"))
}

func TestListDestinationsByType(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/destinations?=kind", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListDestinations(c)
	require.Equal(t, http.StatusOK, w.Code)

	var destinations []api.Destination
	if err := json.NewDecoder(w.Body).Decode(&destinations); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 3, len(destinations))

	require.True(t, containsDestination(destinations, "cluster-AAA"))
	require.True(t, containsDestination(destinations, "cluster-BBB"))
	require.True(t, containsDestination(destinations, "cluster-CCC"))
}

func TestListDestinationsEmpty(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/destinations?name=nonexistent", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListDestinations(c)
	require.Equal(t, http.StatusOK, w.Code)

	var destinations []api.Destination
	if err := json.NewDecoder(w.Body).Decode(&destinations); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 0, len(destinations))
}

// This is a preventative security auditing test that checks for keys with names that seem sensitive on this response
func TestListProvidersHasNoSensitiveValues(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/providers", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.ListProviders(c)
	require.Equal(t, http.StatusOK, w.Code)

	var providers []api.Provider
	if err := json.NewDecoder(w.Body).Decode(&providers); err != nil {
		t.Fatal(err)
	}

	require.Greater(t, len(providers), 0, "no providers returned, could not check sensitive values")

	var providerKeys map[string]interface{}

	inProv, err := json.Marshal(providers[0])
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(inProv, &providerKeys)
	if err != nil {
		t.Fatal(err)
	}

	// check for suspicious key names
	for key := range providerKeys {
		if strings.Contains(strings.ToLower(key), "secret") {
			t.Fatalf("%s in list provider response appears to be sensitive, it should not be returned", key)
		}

		if strings.Contains(strings.ToLower(key), "key") {
			t.Fatalf("%s in list provider response appears to be sensitive, it should not be returned", key)
		}
	}
}

func TestGetDestination(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	destination := &Destination{Name: "mpt-destination"}
	if err := a.db.Create(destination).Error; err != nil {
		t.Fatalf(err.Error())
	}

	defer a.db.Delete(destination)

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/destinations/%s", destination.Id), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r
	c.Params = append(c.Params, gin.Param{Key: "id", Value: destination.Id})

	a.GetDestination(c)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestGetDestinationEmptyID(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/destinations/", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.GetDestination(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetDestinationNotFound(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/destinations/nonexistent", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "nonexistent"})

	a.GetDestination(c)

	require.Equal(t, http.StatusNotFound, w.Code)
}

// This is a preventative security auditing test that checks for keys with names that seem sensitive on this response
func TestGetProviderHasNoSensitiveValues(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	provider := &Provider{Kind: ProviderKindOkta}
	if err := a.db.Create(provider).Error; err != nil {
		t.Fatalf(err.Error())
	}

	defer a.db.Delete(provider)

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/providers/%s", provider.Id), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r
	c.Params = append(c.Params, gin.Param{Key: "id", Value: provider.Id})

	a.GetProvider(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp api.Provider
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	require.NotNil(t, resp, "no provider returned, could not check sensitive values")

	var providerKeys map[string]interface{}

	inProv, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(inProv, &providerKeys)
	if err != nil {
		t.Fatal(err)
	}

	// check for suspicious key names
	for key := range providerKeys {
		if strings.Contains(strings.ToLower(key), "secret") {
			t.Fatalf("%s in list provider response appears to be sensitive, it should not be returned", key)
		}

		if strings.Contains(strings.ToLower(key), "key") {
			t.Fatalf("%s in list provider response appears to be sensitive, it should not be returned", key)
		}
	}
}

func TestCreateAPIKey(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	var apiKey APIKey

	err := db.FirstOrCreate(&apiKey, &APIKey{Name: "create-api-key", Permissions: string(api.API_KEYS_CREATE)}).Error
	if err != nil {
		t.Fatal(err)
	}

	createAPIKeyRequest := api.InfraAPIKeyCreateRequest{
		Name:        "test-api-client",
		Permissions: []api.InfraAPIPermission{api.USERS_READ},
	}

	csr, err := createAPIKeyRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/api-keys", bytes.NewReader(csr))
	r.Header.Add("Authorization", "Bearer "+apiKey.Key)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.API_KEYS_CREATE)(c)
	a.CreateAPIKey(c)

	var body api.InfraAPIKeyCreateResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "test-api-client", body.Name)
	require.NotEmpty(t, body.Key)

	db.Delete(&APIKey{}, &APIKey{Name: "test-api-client"})
	db.Delete(&apiKey)
}

func TestDeleteAPIKey(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	k := &APIKey{Name: "test-delete-key", Permissions: string(api.API_KEYS_DELETE)}
	if err := a.db.Create(k).Error; err != nil {
		t.Fatalf(err.Error())
	}

	delR := httptest.NewRequest(http.MethodDelete, "/v1/api-keys/"+k.Id, nil)
	delR.Header.Add("Authorization", "Bearer "+k.Key)

	delW := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(delW)
	c.Request = delR
	c.Params = append(c.Params, gin.Param{Key: "id", Value: k.Id})
	a.bearerAuthMiddleware(api.API_KEYS_DELETE)(c)
	a.DeleteAPIKey(c)

	require.Equal(t, http.StatusNoContent, c.Writer.Status())

	var apiKey APIKey

	db.First(&apiKey, &APIKey{Name: "test-api-delete-key"})
	require.Empty(t, apiKey.Id, "API key not deleted from database")
}

func TestListAPIKeys(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	k := &APIKey{Name: "test-key", Permissions: string(api.API_KEYS_READ)}
	if err := a.db.Create(k).Error; err != nil {
		t.Fatalf(err.Error())
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/api-keys", nil)
	r.Header.Add("Authorization", "Bearer "+k.Key)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.bearerAuthMiddleware(api.API_KEYS_READ)(c)
	a.ListAPIKeys(c)
	require.Equal(t, http.StatusOK, w.Code)

	var keys []api.InfraAPIKey
	if err := json.NewDecoder(w.Body).Decode(&keys); err != nil {
		t.Fatal(err)
	}

	keyIDs := make(map[string]string)

	for _, k := range keys {
		keyIDs[k.Name] = k.Id
	}

	require.NotEmpty(t, keyIDs["test-key"])
}

func containsUser(users []api.User, email string) bool {
	for _, u := range users {
		if u.Email == email {
			return true
		}
	}

	return false
}

func containsGroup(groups []api.Group, name string) bool {
	for _, g := range groups {
		if g.Name == name {
			return true
		}
	}

	return false
}

func containsDestination(destinations []api.Destination, name string) bool {
	for _, d := range destinations {
		if d.Name == name {
			return true
		}
	}

	return false
}

func TestCredentials(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	err := db.FirstOrCreate(&Settings{}).Error
	require.NoError(t, err)

	jwt, expiry, err := a.createJWT("dest", "steven@infrahq.com")
	require.NoError(t, err)
	require.Greater(t, len(jwt), 1)
	require.Greater(t, expiry.Unix(), time.Now().Unix())
}

func TestCheckPermissionForEmptyPermissions(t *testing.T) {
	require.False(t, checkPermission(api.API_KEYS_CREATE, ""))
	require.False(t, checkPermission(api.API_KEYS_CREATE, " "))
}

func TestCheckPermissionForWrongPermissions(t *testing.T) {
	require.False(t, checkPermission(api.API_KEYS_CREATE, string(api.API_KEYS_DELETE)))

	multiPermissions := strings.Join([]string{
		string(api.USERS_READ),
		string(api.AUTH_DELETE),
		string(api.TOKENS_CREATE),
	}, " ")
	require.False(t, checkPermission(api.API_KEYS_CREATE, multiPermissions))
}

func TestCheckPermissionForCorrectPermissions(t *testing.T) {
	require.True(t, checkPermission(api.API_KEYS_CREATE, string(api.API_KEYS_CREATE)))

	multiPermissions := strings.Join([]string{
		string(api.USERS_READ),
		string(api.API_KEYS_CREATE),
		string(api.TOKENS_CREATE),
	}, " ")
	require.True(t, checkPermission(api.API_KEYS_CREATE, multiPermissions))
}

func TestCheckPermissionForRootPermissionsIsValid(t *testing.T) {
	require.True(t, checkPermission(api.API_KEYS_CREATE, string(api.STAR)))
}

func TestIssueSessionTokenCreatesTokenForSpecifiedUser(t *testing.T) {
	userID := "some-user-id"

	issued, err := issueSessionToken(db, userID, time.Minute)
	require.NoError(t, err)

	id := issued[0:IdLen]

	var token Token
	err = db.Where(&Token{Id: id}).Find(&token).Error
	require.NoError(t, err)

	require.Equal(t, userID, token.UserId)
}

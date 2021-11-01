package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/registry/mocks"
	"github.com/infrahq/infra/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

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
		if _, err := io.WriteString(w, "hello world"); err != nil {
			t.Fatal(err)
		}
	}

	db, err := NewDB("file::memory:")
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
	a.bearerAuthMiddleware(api.USERS_READ, http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareEmptyHeader(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, "hello world"); err != nil {
			t.Fatal(err)
		}
	}

	db, err := NewDB("file::memory:")
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
	a.bearerAuthMiddleware(api.USERS_READ, http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareEmptyHeaderBearer(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, "hello world"); err != nil {
			t.Fatal(err)
		}
	}

	db, err := NewDB("file::memory:")
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
	a.bearerAuthMiddleware(api.USERS_READ, http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareInvalidLength(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, "hello world"); err != nil {
			t.Fatal(err)
		}
	}

	db, err := NewDB("file::memory:")
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
	a.bearerAuthMiddleware(api.USERS_READ, http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareInvalidToken(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, "hello world"); err != nil {
			t.Fatal(err)
		}
	}

	db, err := NewDB("file::memory:")
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
	a.bearerAuthMiddleware(api.USERS_READ, http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareExpiredToken(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, "hello world"); err != nil {
			t.Fatal(err)
		}
	}

	db, err := NewDB("file::memory:")
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
	a.bearerAuthMiddleware(api.USERS_READ, http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestBearerTokenMiddlewareValidToken(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, "hello world"); err != nil {
			t.Fatal(err)
		}
	}

	db, err := NewDB("file::memory:")
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
	a.bearerAuthMiddleware(api.USERS_READ, http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "hello world", w.Body.String())
}

func TestBearerTokenMiddlewareInvalidAPIKey(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, "hello world"); err != nil {
			t.Fatal(err)
		}
	}

	db, err := NewDB("file::memory:")
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
	a.bearerAuthMiddleware(api.USERS_READ, http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerTokenMiddlewareValidAPIKey(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, "hello world"); err != nil {
			t.Fatal(err)
		}
	}

	db, err := NewDB("file::memory:")
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
	a.bearerAuthMiddleware(api.USERS_READ, http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "hello world", w.Body.String())
}

func TestBearerTokenMiddlewareValidAPIKeyRootPermissions(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, "hello world"); err != nil {
			t.Fatal(err)
		}
	}

	db, err := NewDB("file::memory:")
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
	a.bearerAuthMiddleware(api.USERS_READ, http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "hello world", w.Body.String())
}

func TestBearerTokenMiddlewareValidAPIKeyWrongPermission(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, "hello world"); err != nil {
			t.Fatal(err)
		}
	}

	db, err := NewDB("file::memory:")
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
	a.bearerAuthMiddleware(api.DESTINATIONS_CREATE, http.HandlerFunc(handler)).ServeHTTP(w, r)

	var body api.Error
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, w.Code, http.StatusForbidden)
	assert.Equal(t, string(api.DESTINATIONS_CREATE)+" permission is required", body.Message)
}

func TestCreateDestinationNoAPIKey(t *testing.T) {
	db, err := NewDB("file::memory:")
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
	a.bearerAuthMiddleware(api.USERS_READ, http.HandlerFunc(a.Login)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateDestination(t *testing.T) {
	db, err := NewDB("file::memory:")
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
	a.bearerAuthMiddleware(api.DESTINATIONS_CREATE, http.HandlerFunc(a.CreateDestination)).ServeHTTP(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestLoginHandlerEmptyRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
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
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginNilOktaRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
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
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginEmptyOktaRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
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
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginOktaMissingDomainRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
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
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginMethodOktaMissingCodeRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
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
	http.HandlerFunc(a.Login).ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginMethodOkta(t *testing.T) {
	db, err := NewDB("file::memory:")
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
	http.HandlerFunc(a.Version).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var body api.Version
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, internal.Version, body.Version)
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
	http.HandlerFunc(a.ListRoles).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var roles []api.Role
	if err := json.NewDecoder(w.Body).Decode(&roles); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 7, len(roles))

	returnedUserRoles := make(map[string][]api.User)
	for _, r := range roles {
		returnedUserRoles[r.Name] = r.Users
	}

	// roles from direct user assignment
	assert.Equal(t, 1, len(returnedUserRoles["admin"]))
	assert.True(t, containsUser(returnedUserRoles["admin"], adminUser.Email))

	assert.Equal(t, 1, len(returnedUserRoles["audit"]))
	assert.True(t, containsUser(returnedUserRoles["audit"], adminUser.Email))

	assert.Equal(t, 1, len(returnedUserRoles["pod-create"]))
	assert.True(t, containsUser(returnedUserRoles["pod-create"], adminUser.Email))

	assert.Equal(t, 0, len(returnedUserRoles["writer"]))

	returnedGroupRoles := make(map[string][]api.Group)
	for _, r := range roles {
		returnedGroupRoles[r.Name] = r.Groups
	}

	// roles from groups
	assert.Equal(t, 0, len(returnedGroupRoles["admin"]))

	assert.Equal(t, 1, len(returnedGroupRoles["audit"]))
	assert.True(t, containsGroup(returnedGroupRoles["audit"], iosDevGroup.Name))

	assert.Equal(t, 1, len(returnedGroupRoles["pod-create"]))
	assert.True(t, containsGroup(returnedGroupRoles["pod-create"], iosDevGroup.Name))

	assert.Equal(t, 1, len(returnedGroupRoles["writer"]))
	assert.True(t, containsGroup(returnedGroupRoles["writer"], macAdminGroup.Name))
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
	http.HandlerFunc(a.ListRoles).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var roles []api.Role
	if err := json.NewDecoder(w.Body).Decode(&roles); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(roles))

	returnedUserRoles := make(map[string][]api.User)
	for _, r := range roles {
		returnedUserRoles[r.Name] = r.Users
	}

	// roles from direct user assignment
	assert.Equal(t, 1, len(returnedUserRoles["admin"]))
	assert.True(t, containsUser(returnedUserRoles["admin"], adminUser.Email))

	returnedGroupRoles := make(map[string][]api.Group)
	for _, r := range roles {
		returnedGroupRoles[r.Name] = r.Groups
	}

	// roles from groups
	assert.Equal(t, 0, len(returnedGroupRoles["admin"]))
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
	http.HandlerFunc(a.ListRoles).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var roles []api.Role
	if err := json.NewDecoder(w.Body).Decode(&roles); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(roles))

	returnedGroupRoles := make(map[string][]api.Group)
	for _, r := range roles {
		returnedGroupRoles[r.Name] = r.Groups
	}

	// roles from groups
	assert.Equal(t, 1, len(returnedGroupRoles["pod-create"]))
	assert.True(t, containsGroup(returnedGroupRoles["pod-create"], iosDevGroup.Name))
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
	http.HandlerFunc(a.ListRoles).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var roles []api.Role
	if err := json.NewDecoder(w.Body).Decode(&roles); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 0, len(roles))
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

	// roles from direct user assignment
	assert.Equal(t, 1, len(returnedUserRoles["admin"]))
	assert.True(t, containsUser(returnedUserRoles["admin"], adminUser.Email))

	returnedGroupRoles := make(map[string][]api.Group)
	for _, r := range roles {
		returnedGroupRoles[r.Name] = r.Groups
	}

	// roles from groups
	assert.Equal(t, 1, len(returnedGroupRoles["writer"]))
	assert.True(t, containsGroup(returnedGroupRoles["writer"], iosDevGroup.Name))
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
	http.HandlerFunc(a.ListRoles).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var roles []api.Role
	if err := json.NewDecoder(w.Body).Decode(&roles); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 0, len(roles))
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
	vars := map[string]string{
		"id": role.Id,
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.GetRole).ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
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
	vars := map[string]string{
		"id": "",
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.GetRole).ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
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

	r := httptest.NewRequest(http.MethodGet, "/v1/roles/nonexistent", nil)
	vars := map[string]string{
		"id": "nonexistent",
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.GetRole).ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
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
	http.HandlerFunc(a.ListGroups).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var groups []api.Group
	if err := json.NewDecoder(w.Body).Decode(&groups); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(groups))

	assert.True(t, containsGroup(groups, "ios-developers"))
	assert.True(t, containsGroup(groups, "mac-admins"))
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
	http.HandlerFunc(a.ListGroups).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var groups []api.Group
	if err := json.NewDecoder(w.Body).Decode(&groups); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(groups))

	assert.True(t, containsGroup(groups, "ios-developers"))
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
	vars := map[string]string{
		"id": group.Id,
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.GetGroup).ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
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
	vars := map[string]string{
		"id": "",
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.GetGroup).ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
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
	vars := map[string]string{
		"id": "nonexistent",
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.GetGroup).ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
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
	http.HandlerFunc(a.ListUsers).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var users []api.User
	if err := json.NewDecoder(w.Body).Decode(&users); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 3, len(users))

	assert.True(t, containsUser(users, adminUser.Email))
	assert.True(t, containsUser(users, standardUser.Email))
	assert.True(t, containsUser(users, iosDevUser.Email))
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
	http.HandlerFunc(a.ListUsers).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var users []api.User
	if err := json.NewDecoder(w.Body).Decode(&users); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(users))

	assert.True(t, containsUser(users, iosDevUser.Email))
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
	http.HandlerFunc(a.ListUsers).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var users []api.User
	if err := json.NewDecoder(w.Body).Decode(&users); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 0, len(users))
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
	vars := map[string]string{
		"id": user.Id,
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.GetUser).ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
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
	vars := map[string]string{
		"id": "",
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.GetUser).ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
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
	vars := map[string]string{
		"id": "nonexistent",
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.GetUser).ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
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
	http.HandlerFunc(a.ListProviders).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var providers []api.Provider
	if err := json.NewDecoder(w.Body).Decode(&providers); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(providers))
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
	http.HandlerFunc(a.ListProviders).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var providers []api.Provider
	if err := json.NewDecoder(w.Body).Decode(&providers); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(providers))
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
	http.HandlerFunc(a.ListProviders).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var providers []api.Provider
	if err := json.NewDecoder(w.Body).Decode(&providers); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 0, len(providers))
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
	vars := map[string]string{
		"id": provider.Id,
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.GetProvider).ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
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
	vars := map[string]string{
		"id": "",
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.GetProvider).ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
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
	vars := map[string]string{
		"id": "nonexistent",
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.GetProvider).ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateProvider(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	domain := "test.create.example.com"
	clientID := "testClientID"
	clientSecret := "testClientSecret"
	apiToken := "apiToken"

	provider := api.Provider{
		Domain:       domain,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Kind:         ProviderKindOkta,
		Okta:         &api.ProviderOkta{APIToken: apiToken},
	}

	cpr, err := provider.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/providers", bytes.NewReader(cpr))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.CreateProvider).ServeHTTP(w, r)

	var body api.Provider
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, domain, body.Domain)
	assert.Equal(t, clientID, body.ClientID)
	assert.Equal(t, clientSecret, body.ClientSecret)
	assert.Equal(t, ProviderKindOkta, body.Kind)
	assert.Equal(t, apiToken, body.Okta.APIToken)

	// clean up
	var created Provider

	db.First(&created, &Provider{Domain: domain})
	db.Delete(&created)
}

func TestCreateProviderInvalidKind(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	domain := "test.create.example.com"
	clientID := "testClientID"
	clientSecret := "testClientSecret"
	apiToken := "apiToken"
	kind := "not-okta"

	provider := api.Provider{
		Domain:       domain,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Kind:         kind,
		Okta:         &api.ProviderOkta{APIToken: apiToken},
	}

	cpr, err := provider.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/providers", bytes.NewReader(cpr))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.CreateProvider).ServeHTTP(w, r)

	var body api.Provider
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateProvider(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	existing := &Provider{Domain: "before.update.example.com", Kind: ProviderKindOkta}
	if err := a.db.Create(existing).Error; err != nil {
		t.Fatalf(err.Error())
	}

	domain := "test.update.example.com"
	clientID := "testUpdateClientID"
	clientSecret := "testUpdateClientSecret"
	apiToken := "apiTokenUpdate"

	provider := api.Provider{
		Domain:       domain,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Kind:         ProviderKindOkta,
		Okta:         &api.ProviderOkta{APIToken: apiToken},
	}

	cpr, err := provider.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/providers/%s", existing.Id), bytes.NewReader(cpr))
	vars := map[string]string{
		"id": existing.Id,
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.UpdateProvider).ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var updated Provider
	db.First(&updated, &Provider{Id: existing.Id})

	assert.Equal(t, domain, updated.Domain)
	assert.Equal(t, clientID, updated.ClientID)
	assert.Equal(t, clientSecret, updated.ClientSecret)
	assert.Equal(t, ProviderKindOkta, updated.Kind)
	assert.Equal(t, apiToken, updated.APIToken)

	// clean up
	db.Delete(&updated)
}

func TestUpdateProviderNoID(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	domain := "test.update.example.com"
	clientID := "testUpdateClientID"
	clientSecret := "testUpdateClientSecret"
	apiToken := "apiTokenUpdate"

	provider := api.Provider{
		Domain:       domain,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Kind:         ProviderKindOkta,
		Okta:         &api.ProviderOkta{APIToken: apiToken},
	}

	cpr, err := provider.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPut, "/v1/providers/", bytes.NewReader(cpr))

	w := httptest.NewRecorder()
	http.HandlerFunc(a.UpdateProvider).ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateProviderInvalidID(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	nonexistantID := "invalid-ID"

	domain := "test.update.example.com"
	clientID := "testUpdateClientID"
	clientSecret := "testUpdateClientSecret"
	apiToken := "apiTokenUpdate"

	provider := api.Provider{
		Domain:       domain,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Kind:         ProviderKindOkta,
		Okta:         &api.ProviderOkta{APIToken: apiToken},
	}

	cpr, err := provider.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/providers/%s", nonexistantID), bytes.NewReader(cpr))
	vars := map[string]string{
		"id": nonexistantID,
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.UpdateProvider).ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateProviderInvalidKind(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	existing := &Provider{Domain: "before.update.example.com", Kind: ProviderKindOkta}
	if err := a.db.Create(existing).Error; err != nil {
		t.Fatalf(err.Error())
	}

	domain := "test.update.example.com"
	clientID := "testUpdateClientID"
	clientSecret := "testUpdateClientSecret"
	apiToken := "apiTokenUpdate"

	provider := api.Provider{
		Domain:       domain,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Kind:         "not-okta",
		Okta:         &api.ProviderOkta{APIToken: apiToken},
	}

	cpr, err := provider.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/providers/%s", existing.Id), bytes.NewReader(cpr))
	vars := map[string]string{
		"id": existing.Id,
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.UpdateProvider).ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteProvider(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	domain := "test-delete-provider-domain.example.com"
	provider := &Provider{Domain: domain, Kind: ProviderKindOkta}
	if err := a.db.Create(provider).Error; err != nil {
		t.Fatalf(err.Error())
	}

	r := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/providers/%s", provider.Id), nil)
	vars := map[string]string{
		"id": provider.Id,
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.DeleteProvider).ServeHTTP(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)

	var deleteProvider Provider

	db.First(&deleteProvider, &Provider{Id: provider.Id})
	assert.Empty(t, deleteProvider.Id, "Provider not deleted from database")
}

func TestDeleteProviderDoesNotExist(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	doesNotExistID := "does-not-exist"
	r := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/providers/%s", doesNotExistID), nil)
	vars := map[string]string{
		"id": doesNotExistID,
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.DeleteProvider).ServeHTTP(w, r)

	// 204 is returned, this didn't exist but that is ok
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteProviderNoID(t *testing.T) {
	a := &API{
		registry: &Registry{
			db: db,
			secrets: map[string]secrets.SecretStorage{
				"kubernetes": NewMockSecretReader(),
			},
		},
		db: db,
	}

	r := httptest.NewRequest(http.MethodDelete, "/v1/providers/", nil)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.DeleteProvider).ServeHTTP(w, r)

	// 204 is returned, this didn't exist but that is ok
	assert.Equal(t, http.StatusBadRequest, w.Code)
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
	http.HandlerFunc(a.ListDestinations).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var destinations []api.Destination
	if err := json.NewDecoder(w.Body).Decode(&destinations); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 3, len(destinations))

	assert.True(t, containsDestination(destinations, "cluster-AAA"))
	assert.True(t, containsDestination(destinations, "cluster-BBB"))
	assert.True(t, containsDestination(destinations, "cluster-CCC"))
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
	http.HandlerFunc(a.ListDestinations).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var destinations []api.Destination
	if err := json.NewDecoder(w.Body).Decode(&destinations); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(destinations))

	assert.True(t, containsDestination(destinations, "cluster-AAA"))
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
	http.HandlerFunc(a.ListDestinations).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var destinations []api.Destination
	if err := json.NewDecoder(w.Body).Decode(&destinations); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 3, len(destinations))

	assert.True(t, containsDestination(destinations, "cluster-AAA"))
	assert.True(t, containsDestination(destinations, "cluster-BBB"))
	assert.True(t, containsDestination(destinations, "cluster-CCC"))
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
	http.HandlerFunc(a.ListDestinations).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var destinations []api.Destination
	if err := json.NewDecoder(w.Body).Decode(&destinations); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 0, len(destinations))
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
	vars := map[string]string{
		"id": destination.Id,
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.GetDestination).ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
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
	vars := map[string]string{
		"id": "",
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.GetDestination).ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
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
	vars := map[string]string{
		"id": "nonexistent",
	}
	r = mux.SetURLVars(r, vars)

	w := httptest.NewRecorder()
	http.HandlerFunc(a.GetDestination).ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
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

	createAPIKeyRequest := api.InfraAPIKeyCreateRequest{
		Name:        "test-api-client",
		Permissions: []api.InfraAPIPermission{api.USERS_READ},
	}

	csr, err := createAPIKeyRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/api-keys", bytes.NewReader(csr))
	w := httptest.NewRecorder()
	http.HandlerFunc(a.CreateAPIKey).ServeHTTP(w, r)

	var body api.InfraAPIKeyCreateResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "test-api-client", body.Name)
	assert.NotEmpty(t, body.Key)

	// clean up
	var apiKey APIKey

	db.First(&apiKey, &APIKey{Name: "test-api-client"})
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
	vars := map[string]string{
		"id": k.Id,
	}
	delR = mux.SetURLVars(delR, vars)
	delW := httptest.NewRecorder()
	http.HandlerFunc(a.DeleteAPIKey).ServeHTTP(delW, delR)

	assert.Equal(t, http.StatusNoContent, delW.Code)

	var apiKey APIKey

	db.First(&apiKey, &APIKey{Name: "test-api-delete-key"})
	assert.Empty(t, apiKey.Id, "API key not deleted from database")
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
	w := httptest.NewRecorder()
	http.HandlerFunc(a.ListAPIKeys).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var keys []api.InfraAPIKey
	if err := json.NewDecoder(w.Body).Decode(&keys); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(keys))

	keyIDs := make(map[string]string)

	for _, k := range keys {
		keyIDs[k.Name] = k.Id
	}

	assert.NotEmpty(t, keyIDs["test-key"])
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

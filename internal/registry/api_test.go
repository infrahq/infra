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
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/data"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/registry/mocks"
	"github.com/infrahq/infra/secrets"
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

func issueAPIKey(t *testing.T, db *gorm.DB, permissions string) *data.APIKey {
	secret, err := generate.CryptoRandom(data.APIKeyLength)
	require.NoError(t, err)

	apiKey := &data.APIKey{
		Name:        "test",
		Key:         secret,
		Permissions: permissions,
	}

	apiKey, err = data.CreateAPIKey(db, apiKey)
	require.NoError(t, err)

	return apiKey
}

func TestCreateDestination(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"OK": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionDestinationCreate))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T) *api.DestinationCreateRequest {
				return &api.DestinationCreateRequest{
					Kind:   api.DESTINATIONKIND_KUBERNETES,
					NodeID: "test",
					Name:   "test",
					Kubernetes: &api.DestinationKubernetes{
						Ca:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, w.Code)
			},
		},
		"NoKind": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionDestinationCreate))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T) *api.DestinationCreateRequest {
				return &api.DestinationCreateRequest{
					Kubernetes: &api.DestinationKubernetes{
						Ca:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		"UnknownKind": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionDestinationCreate))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T) *api.DestinationCreateRequest {
				return &api.DestinationCreateRequest{
					Kind: api.DestinationKind("unknown"),
					Kubernetes: &api.DestinationKubernetes{
						Ca:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		"NoAuthorization": {
			"requestFunc": func(t *testing.T) *api.DestinationCreateRequest {
				return &api.DestinationCreateRequest{
					Kind: api.DESTINATIONKIND_KUBERNETES,
					Kubernetes: &api.DestinationKubernetes{
						Ca:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, w.Code)
			},
		},
		"BadPermissions": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, "infra.bad.permissions")
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T) *api.DestinationCreateRequest {
				return &api.DestinationCreateRequest{
					Kind: api.DESTINATIONKIND_KUBERNETES,
					Kubernetes: &api.DestinationKubernetes{
						Ca:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			db := configure(t, nil)

			requestFunc, ok := v["requestFunc"].(func(*testing.T) *api.DestinationCreateRequest)
			require.True(t, ok)

			request := requestFunc(t)
			bts, err := request.MarshalJSON()
			require.NoError(t, err)

			r := httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set("db", db)
			c.Request = r

			authFunc, ok := v["authFunc"].(func(*testing.T, *gorm.DB, *gin.Context))
			if ok {
				authFunc(t, db, c)
			}

			a := API{}

			a.CreateDestination(c)

			verifyFunc, ok := v["verifyFunc"].(func(*testing.T, *http.Request, *httptest.ResponseRecorder))
			require.True(t, ok)

			verifyFunc(t, r, w)
		})
	}
}

func TestCreateDestinationUpdatesField(t *testing.T) {
	db := configure(t, nil)

	destination, err := data.CreateDestination(db, &data.Destination{
		Kind:     data.DestinationKindKubernetes,
		NodeID:   "node-id",
		Name:     "name",
		Endpoint: "endpoint",
		Kubernetes: data.DestinationKubernetes{
			CA: "ca",
		},
	})

	require.NoError(t, err)
	require.Equal(t, "node-id", destination.NodeID)
	require.Equal(t, "name", destination.Name)
	require.Equal(t, "endpoint", destination.Endpoint)
	require.Equal(t, "ca", destination.Kubernetes.CA)

	request := api.DestinationCreateRequest{
		Kind:   api.DESTINATIONKIND_KUBERNETES,
		NodeID: destination.NodeID,
		Name:   "updated-name",
		Kubernetes: &api.DestinationKubernetes{
			Ca:       "updated-ca",
			Endpoint: "updated-endpoint",
		},
	}

	bts, err := request.MarshalJSON()
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("db", db)
	c.Request = r

	apiKey := issueAPIKey(t, db, string(access.PermissionDestinationCreate))
	c.Set("authorization", apiKey.Key)

	a := API{}

	a.CreateDestination(c)

	require.Equal(t, http.StatusCreated, w.Code)

	var body api.Destination
	err = json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	require.Equal(t, "node-id", body.NodeID)
	require.Equal(t, "updated-name", body.Name)
	require.Equal(t, "updated-ca", body.Kubernetes.Ca)
	require.Equal(t, "updated-endpoint", body.Kubernetes.Endpoint)

	destinations, err := data.ListDestinations(db, &data.Destination{NodeID: "node-id"})
	require.NoError(t, err)
	require.Len(t, destinations, 1)
	require.Equal(t, body.Id, destinations[0].ID.String())
	require.Equal(t, body.NodeID, destinations[0].NodeID)
	require.Equal(t, body.Name, destinations[0].Name)
	require.Equal(t, body.Kubernetes.Endpoint, destinations[0].Endpoint)
	require.Equal(t, body.Kubernetes.Ca, destinations[0].Kubernetes.CA)
}

func TestLogin(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"NilRequest": {
			"requestFunc": func(t *testing.T) *http.Request {
				return httptest.NewRequest(http.MethodPost, "/v1/login", nil)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		"OktaNil": {
			"requestFunc": func(t *testing.T) *http.Request {
				r := &api.LoginRequest{
					Okta: nil,
				}

				bts, err := r.MarshalJSON()
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		"OktaEmpty": {
			"requestFunc": func(t *testing.T) *http.Request {
				r := &api.LoginRequest{
					Okta: &api.LoginRequestOkta{},
				}

				bts, err := r.MarshalJSON()
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		"OktaMissingDomain": {
			"requestFunc": func(t *testing.T) *http.Request {
				r := &api.LoginRequest{
					Okta: &api.LoginRequestOkta{
						Code: "code",
					},
				}

				bts, err := r.MarshalJSON()
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		"OktaMissingCodeRequest": {
			"requestFunc": func(t *testing.T) *http.Request {
				r := &api.LoginRequest{
					Okta: &api.LoginRequestOkta{
						Domain: "test.okta.com",
					},
				}

				bts, err := r.MarshalJSON()
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			db := configure(t, nil)

			requestFunc, ok := v["requestFunc"].(func(*testing.T) *http.Request)
			require.True(t, ok)

			r := requestFunc(t)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set("db", db)
			c.Request = r

			a := &API{}

			a.Login(c)

			verifyFunc, ok := v["verifyFunc"].(func(*testing.T, *http.Request, *httptest.ResponseRecorder))
			require.True(t, ok)

			verifyFunc(t, r, w)
		})
	}
}

func TestLoginOkta(t *testing.T) {
	db := configure(t, nil)

	testOkta := new(mocks.Okta)
	testOkta.On("EmailFromCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("jbond@infrahq.com", nil)

	request := api.LoginRequest{
		Okta: &api.LoginRequestOkta{
			Domain: "test.okta.com",
			Code:   "code",
		},
	}

	bts, err := request.MarshalJSON()
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("okta", testOkta)
	c.Set("db", db)
	c.Request = r

	a := API{
		registry: &Registry{
			secrets: map[string]secrets.SecretStorage{
				"base64": NewMockSecretReader(),
			},
		},
	}

	a.Login(c)

	require.Equal(t, http.StatusOK, w.Code)

	var body api.LoginResponse
	err = json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	require.Equal(t, "jbond@infrahq.com", body.Name)
	require.NotEmpty(t, body.Token)
}

func TestLogout(t *testing.T) {
}

func TestVersion(t *testing.T) {
	a := API{}

	r := httptest.NewRequest(http.MethodGet, "/v1/version", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r

	a.Version(c)

	require.Equal(t, http.StatusOK, w.Code)

	var body api.Version
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	require.Equal(t, internal.Version, body.Version)
}

func TestT(t *testing.T) {
	cases := map[string]map[string]interface{}{
		// /v1/users
		"GetUser": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionUserRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				c.Params = append(c.Params, gin.Param{Key: "id", Value: userBond.ID.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/users/%s", userBond.ID), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetUser(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var user api.User
				err := json.NewDecoder(w.Body).Decode(&user)
				require.NoError(t, err)
				require.Equal(t, "jbond@infrahq.com", user.Email)
			},
		},
		"GetUserEmptyID": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionUserRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/users/", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetUser(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		"GetUserUnknownUser": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionUserRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/users/%s", id), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetUser(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		"ListUsers": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionUserRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/users", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListUsers(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var users []api.User
				err := json.NewDecoder(w.Body).Decode(&users)
				require.NoError(t, err)
				require.Len(t, users, 2)
				require.ElementsMatch(t, []string{"jbond@infrahq.com", "jbourne@infrahq.com"}, []string{users[0].Email, users[1].Email})
			},
		},
		"ListUsersByEmail": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionUserRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/users?email=jbond@infrahq.com", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListUsers(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var users []api.User
				err := json.NewDecoder(w.Body).Decode(&users)
				require.NoError(t, err)
				require.Len(t, users, 1)
				require.Equal(t, "jbond@infrahq.com", users[0].Email)
			},
		},
		"ListUsersUnknownEmail": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionUserRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/users?email=unknown@infrahq.com", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListUsers(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var users []api.User
				err := json.NewDecoder(w.Body).Decode(&users)
				require.NoError(t, err)
				require.Len(t, users, 0)
			},
		},

		// /v1/groups
		"GetGroup": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionGroupRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				c.Params = append(c.Params, gin.Param{Key: "id", Value: groupEveryone.ID.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/groups/%s", groupEveryone.ID), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetGroup(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var group api.Group
				err := json.NewDecoder(w.Body).Decode(&group)
				require.NoError(t, err)
				require.Equal(t, "Everyone", group.Name)
			},
		},
		"GetGroupEmptyID": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionGroupRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/groups/", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetGroup(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		"GetGroupUnknownGroup": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionGroupRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/groups/%s", id), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetGroup(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		"ListGroups": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionGroupRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/groups", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListGroups(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var groups []api.Group
				err := json.NewDecoder(w.Body).Decode(&groups)
				require.NoError(t, err)
				require.Len(t, groups, 2)
				require.ElementsMatch(t, []string{"Everyone", "Engineering"}, []string{groups[0].Name, groups[1].Name})
			},
		},
		"ListGroupsByName": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionGroupRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/groups?name=Everyone", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListGroups(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var groups []api.Group
				err := json.NewDecoder(w.Body).Decode(&groups)
				require.NoError(t, err)
				require.Len(t, groups, 1)
				require.Equal(t, "Everyone", groups[0].Name)
			},
		},
		"ListGroupsUnknownName": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionGroupRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/groups?name=unknown", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListGroups(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var groups []api.Group
				err := json.NewDecoder(w.Body).Decode(&groups)
				require.NoError(t, err)
				require.Len(t, groups, 0)
			},
		},

		// /v1/roles
		// "GetRole": map[string]interface{} {
		// 	"authFunc": func (t *testing.T, db *gorm.DB, c *gin.Context) {
		// 		apiKey := issueAPIKey(t, db, string(access.PermissionRoleRead))
		// 		c.Set("authorization", apiKey.Key)
		// 	},
		// 	"requestFunc": func (t *testing.T, c *gin.Context) *http.Request {
		// 		c.Params = append(c.Params, gin.Param{Key: "id", Value: roleEveryone.ID.String()})
		// 		return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/roles/%s", roleEveryone.ID), nil)
		// 	},
		// 	"func": func (a *API, c *gin.Context) {
		// 		a.GetRole(c)
		// 	},
		// 	"verifyFunc": func (t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
		// 		require.Equal(t, http.StatusOK, w.Code)

		// 		var role api.Role
		// 		err := json.NewDecoder(w.Body).Decode(&role)
		// 		require.NoError(t, err)
		// 		require.Equal(t, "Everyone", role.Name)
		// 	},
		// },
		"GetRoleEmptyID": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionRoleRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/roles/", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetRole(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		"GetRoleUnknownRole": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionRoleRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/roles/%s", id), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetRole(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		"ListRoles": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionRoleRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/roles", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListRoles(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var roles []api.Role
				err := json.NewDecoder(w.Body).Decode(&roles)
				require.NoError(t, err)
				require.Len(t, roles, 8)
			},
		},
		"ListRolesByDestinationID": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionRoleRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/roles?destination=%s", destinationAAA.ID), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListRoles(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var roles []api.Role
				err := json.NewDecoder(w.Body).Decode(&roles)
				require.NoError(t, err)
				require.Len(t, roles, 2)
				require.ElementsMatch(t, []string{"AAA", "AAA"}, []string{
					roles[0].Destination.Name,
					roles[1].Destination.Name,
				})
				require.ElementsMatch(t, []string{"writer", "admin"}, []string{roles[0].Name, roles[1].Name})
				require.ElementsMatch(t, []api.RoleKind{api.ROLEKIND_CLUSTER_ROLE, api.ROLEKIND_CLUSTER_ROLE}, []api.RoleKind{roles[0].Kind, roles[1].Kind})
			},
		},
		"ListRolesByKind": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionRoleRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/roles?kind=role", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListRoles(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var roles []api.Role
				err := json.NewDecoder(w.Body).Decode(&roles)
				require.NoError(t, err)
				require.Len(t, roles, 4)

				for _, r := range roles {
					require.Equal(t, api.ROLEKIND_ROLE, r.Kind)
				}
			},
		},
		"ListRolesByName": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionRoleRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/roles?name=admin", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListRoles(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var roles []api.Role
				err := json.NewDecoder(w.Body).Decode(&roles)
				require.NoError(t, err)
				require.Len(t, roles, 3)

				for _, r := range roles {
					require.Equal(t, "admin", r.Name)
				}
			},
		},
		"ListRolesCombo": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionRoleRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/roles?kind=cluster-role&name=admin", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListRoles(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var roles []api.Role
				err := json.NewDecoder(w.Body).Decode(&roles)
				require.NoError(t, err)
				require.Len(t, roles, 3)

				for _, r := range roles {
					require.Equal(t, "admin", r.Name)
					require.Equal(t, api.ROLEKIND_CLUSTER_ROLE, r.Kind)
				}
			},
		},
		"ListRolesCombo3": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionRoleRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/roles?destination=%s&kind=role&name=audit", destinationCCC.ID), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListRoles(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var roles []api.Role
				err := json.NewDecoder(w.Body).Decode(&roles)
				require.NoError(t, err)
				require.Len(t, roles, 2)
				require.ElementsMatch(t, []string{"infrahq", "development"}, []string{roles[0].Namespace, roles[1].Namespace})

				for _, r := range roles {
					require.Equal(t, destinationCCC.ID.String(), r.Destination.Id)
					require.Equal(t, "CCC", r.Destination.Name)
					require.Equal(t, "audit", r.Name)
					require.Equal(t, api.ROLEKIND_ROLE, r.Kind)
				}
			},
		},
		"ListRolesNotFound": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionRoleRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/roles?destination=%s&kind=role&name=audit", id), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListRoles(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var roles []api.Role
				err := json.NewDecoder(w.Body).Decode(&roles)
				require.NoError(t, err)
				require.Len(t, roles, 0)
			},
		},

		// /v1/providers
		"GetProvider": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionProviderRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				c.Params = append(c.Params, gin.Param{Key: "id", Value: providerOkta.ID.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/providers/%s", providerOkta.ID), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetProvider(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var provider api.Provider
				err := json.NewDecoder(w.Body).Decode(&provider)
				require.NoError(t, err)
				require.Equal(t, "test.okta.com", provider.Domain)
				require.Equal(t, "plaintext:0oapn0qwiQPiMIyR35d6", provider.ClientID)
			},
		},
		"GetProviderEmptyID": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionProviderRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetProvider(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		"GetProviderUnknownProvider": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionProviderRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/providers/%s", id), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetProvider(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		"ListProviders": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionProviderRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/providers", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListProviders(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var providers []api.Provider
				err := json.NewDecoder(w.Body).Decode(&providers)
				require.NoError(t, err)
				require.Len(t, providers, 1)
				require.Equal(t, "test.okta.com", providers[0].Domain)
			},
		},
		"ListProvidersByKind": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionProviderRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/providers?kind=okta", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListProviders(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var providers []api.Provider
				err := json.NewDecoder(w.Body).Decode(&providers)
				require.NoError(t, err)
				require.Len(t, providers, 1)
				require.Equal(t, "test.okta.com", providers[0].Domain)
			},
		},
		"ListProvidersByDomain": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionProviderRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/providers?domain=test.okta.com", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListProviders(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var providers []api.Provider
				err := json.NewDecoder(w.Body).Decode(&providers)
				require.NoError(t, err)
				require.Len(t, providers, 1)
				require.Equal(t, "test.okta.com", providers[0].Domain)
			},
		},
		"ListProvidersNotFound": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionProviderRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/providers?domain=nonexistent.okta.com", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListProviders(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var providers []api.Provider
				err := json.NewDecoder(w.Body).Decode(&providers)
				require.NoError(t, err)
				require.Len(t, providers, 0)
			},
		},
		"ListProvidersSensitiveInformation": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionProviderRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/providers?domain=test.okta.com", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListProviders(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var providers []api.Provider
				err := json.NewDecoder(w.Body).Decode(&providers)
				require.NoError(t, err)
				require.Len(t, providers, 1)

				raw, err := json.Marshal(providers[0])
				require.NoError(t, err)

				var provider map[string]interface{}
				err = json.Unmarshal(raw, &provider)
				require.NoError(t, err)

				for key := range provider {
					leak := strings.Contains(strings.ToLower(key), "secret")
					require.False(t, leak)

					leak = strings.Contains(strings.ToLower(key), "key")
					require.False(t, leak)
				}
			},
		},

		// /v1/destinations
		"GetDestination": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionDestinationRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				c.Params = append(c.Params, gin.Param{Key: "id", Value: destinationAAA.ID.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/destinations/%s", destinationAAA.ID), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetDestination(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var destination api.Destination
				err := json.NewDecoder(w.Body).Decode(&destination)
				require.NoError(t, err)
				require.Equal(t, "AAA", destination.Name)
				require.Equal(t, "AAA", destination.NodeID)
				require.Equal(t, "develop.infrahq.com", destination.Kubernetes.Endpoint)
			},
		},
		"GetDestinationEmptyID": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionDestinationRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/destinations/", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetDestination(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		"GetDestinationUnknownDestination": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionDestinationRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/destinations/%s", id), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetDestination(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		"ListDestinations": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionDestinationRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/destinations", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListDestinations(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var destinations []api.Destination
				err := json.NewDecoder(w.Body).Decode(&destinations)
				require.NoError(t, err)
				require.Len(t, destinations, 3)
				require.ElementsMatch(t, []string{"AAA", "BBB", "CCC"}, []string{
					destinations[0].Name,
					destinations[1].Name,
					destinations[2].Name,
				})
				require.ElementsMatch(t, []string{"AAA", "BBB", "CCC"}, []string{
					destinations[0].NodeID,
					destinations[1].NodeID,
					destinations[2].NodeID,
				})
				require.ElementsMatch(t, []string{"develop.infrahq.com", "stage.infrahq.com", "production.infrahq.com"}, []string{
					destinations[0].Kubernetes.Endpoint,
					destinations[1].Kubernetes.Endpoint,
					destinations[2].Kubernetes.Endpoint,
				})
			},
		},
		"ListDestinationsByKind": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionDestinationRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/destinations?kind=kubernetes", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListDestinations(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var destinations []api.Destination
				err := json.NewDecoder(w.Body).Decode(&destinations)
				require.NoError(t, err)
				require.Len(t, destinations, 3)

				for _, d := range destinations {
					require.Equal(t, api.DESTINATIONKIND_KUBERNETES, d.Kind)
				}
			},
		},
		"ListDestinationsByName": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionDestinationRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/destinations?name=AAA", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListDestinations(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var destinations []api.Destination
				err := json.NewDecoder(w.Body).Decode(&destinations)
				require.NoError(t, err)
				require.Len(t, destinations, 1)
				require.Equal(t, "AAA", destinations[0].Name)
				require.Equal(t, "AAA", destinations[0].NodeID)
				require.Equal(t, "develop.infrahq.com", destinations[0].Kubernetes.Endpoint)
			},
		},
		"ListDestinationsCombo": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionDestinationRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/destinations?kind=kubernetes&name=AAA", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListDestinations(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var destinations []api.Destination
				err := json.NewDecoder(w.Body).Decode(&destinations)
				require.NoError(t, err)
				require.Len(t, destinations, 1)
				require.Equal(t, "AAA", destinations[0].Name)
				require.Equal(t, "AAA", destinations[0].NodeID)
				require.Equal(t, "develop.infrahq.com", destinations[0].Kubernetes.Endpoint)
			},
		},
		"ListDestinationsNotFound": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionDestinationRead))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/destinations?name=nonexistent", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListDestinations(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var destinations []api.Destination
				err := json.NewDecoder(w.Body).Decode(&destinations)
				require.NoError(t, err)
				require.Len(t, destinations, 0)
			},
		},

		// /v1/api-keys
		"ListAPIKeys": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				apiKey := issueAPIKey(t, db, string(access.PermissionAPIKeyList))
				c.Set("authorization", apiKey.Key)
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/api-keys", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListAPIKeys(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var apiKeys []api.InfraAPIKey
				err := json.NewDecoder(w.Body).Decode(&apiKeys)
				require.NoError(t, err)
				require.Len(t, apiKeys, 1)
			},
		},

		// /v1/tokens
		"CreateToken": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				_, token, err := access.IssueToken(c, "jbond@infrahq.com", time.Hour*1)
				require.NoError(t, err)

				c.Set("authorization", token.SessionToken())
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				request := api.TokenRequest{
					Destination: "AAA",
				}

				bts, err := request.MarshalJSON()
				require.NoError(t, err)
				return httptest.NewRequest(http.MethodPost, "/v1/token", bytes.NewReader(bts))
			},
			"func": func(a *API, c *gin.Context) {
				a.CreateToken(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var token api.Token
				err := json.NewDecoder(w.Body).Decode(&token)
				require.NoError(t, err)
				require.NotEmpty(t, token.Token)
				require.WithinDuration(t, time.Unix(token.Expires, 0), time.Now(), time.Hour*1)
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			db := configure(t, nil)

			_, err := data.InitializeSettings(db)
			require.NoError(t, err)

			requestFunc, ok := v["requestFunc"].(func(*testing.T, *gin.Context) *http.Request)
			require.True(t, ok)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			r := requestFunc(t, c)
			c.Set("db", db)
			c.Request = r

			authFunc, ok := v["authFunc"].(func(*testing.T, *gorm.DB, *gin.Context))
			if ok {
				authFunc(t, db, c)
			}

			fn, ok := v["func"].(func(*API, *gin.Context))
			require.True(t, ok)

			a := API{}

			fn(&a, c)

			verifyFunc, ok := v["verifyFunc"].(func(*testing.T, *http.Request, *httptest.ResponseRecorder))
			require.True(t, ok)

			verifyFunc(t, r, w)
		})
	}
}

func TestCreateAPIKey(t *testing.T) {
	db := configure(t, nil)

	apiKey := issueAPIKey(t, db, strings.Join([]string{
		string(access.PermissionAPIKeyIssue),
	}, " "))

	request := api.InfraAPIKeyCreateRequest{
		Name:        "tmp",
		Permissions: []string{"infra.*"},
	}

	bts, err := request.MarshalJSON()
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/v1/api-keys", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("db", db)
	c.Set("authorization", apiKey.Key)
	c.Request = r

	a := API{}

	a.CreateAPIKey(c)

	require.Equal(t, http.StatusCreated, w.Code)

	var body api.InfraAPIKeyCreateResponse
	err = json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)

	newr := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	neww := httptest.NewRecorder()
	newc, _ := gin.CreateTestContext(neww)
	newc.Set("db", db)
	newc.Set("authorization", body.Key)
	newc.Request = newr

	a.ListUsers(newc)

	require.Equal(t, http.StatusOK, neww.Code)
}

func TestDeleteAPIKey(t *testing.T) {
	db := configure(t, nil)

	apiKey := issueAPIKey(t, db, strings.Join([]string{
		string(access.PermissionUserRead),
		string(access.PermissionAPIKeyRevoke),
	}, " "))

	oldr := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	oldw := httptest.NewRecorder()
	oldc, _ := gin.CreateTestContext(oldw)
	oldc.Set("db", db)
	oldc.Set("authorization", apiKey.Key)
	oldc.Request = oldr

	a := API{}

	a.ListUsers(oldc)

	require.Equal(t, http.StatusOK, oldw.Code)

	r := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/api-keys/%s", apiKey.ID.String()), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("db", db)
	c.Set("authorization", apiKey.Key)
	c.Request = r
	c.Params = append(c.Params, gin.Param{Key: "id", Value: apiKey.ID.String()})

	a.DeleteAPIKey(c)

	require.Equal(t, http.StatusNoContent, w.Code)

	newr := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	neww := httptest.NewRecorder()
	newc, _ := gin.CreateTestContext(neww)
	newc.Set("db", db)
	newc.Set("authorization", apiKey.Key)
	newc.Request = newr

	a.ListUsers(newc)

	require.Equal(t, http.StatusUnauthorized, neww.Code)
}

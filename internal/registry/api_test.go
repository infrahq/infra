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
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/mocks"
	"github.com/infrahq/infra/internal/registry/models"
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

func issueAPIToken(t *testing.T, db *gorm.DB, permissions string) *models.APIToken {
	apiToken := &models.APIToken{
		Name:        "test",
		Permissions: permissions,
		TTL:         1 * time.Hour,
	}

	_, err := data.CreateAPIToken(db, apiToken, &models.Token{})
	require.NoError(t, err)

	return apiToken
}

func TestCreateDestination(t *testing.T) {
	cases := []struct {
		Name        string
		AuthFunc    func(t *testing.T, db *gorm.DB, c *gin.Context)
		RequestFunc func(t *testing.T) *api.DestinationRequest
		VerifyFunc  func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder)
	}{
		{
			Name: "OK",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationCreate))
			},
			RequestFunc: func(t *testing.T) *api.DestinationRequest {
				return &api.DestinationRequest{
					Kind:   api.DestinationKindKubernetes,
					NodeID: "test",
					Name:   "test",
					Kubernetes: &api.DestinationKubernetes{
						CA:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, w.Code)
			},
		},
		{
			Name: "NoKind",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationCreate))
			},
			RequestFunc: func(t *testing.T) *api.DestinationRequest {
				return &api.DestinationRequest{
					Kubernetes: &api.DestinationKubernetes{
						CA:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			Name: "UnknownKind",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationCreate))
			},
			RequestFunc: func(t *testing.T) *api.DestinationRequest {
				return &api.DestinationRequest{
					Kind: api.DestinationKind("unknown"),
					Kubernetes: &api.DestinationKubernetes{
						CA:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			Name: "NoAuthorization",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", "")
			},
			RequestFunc: func(t *testing.T) *api.DestinationRequest {
				return &api.DestinationRequest{
					Kind: api.DestinationKindKubernetes,
					Kubernetes: &api.DestinationKubernetes{
						CA:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
			},
		},
		{
			Name: "BadPermissions",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", "infra.bad.permissions")
			},
			RequestFunc: func(t *testing.T) *api.DestinationRequest {
				return &api.DestinationRequest{
					Kind: api.DestinationKindKubernetes,
					Kubernetes: &api.DestinationKubernetes{
						CA:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
			},
		},
	}

	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			_, db := configure(t, nil)

			request := test.RequestFunc(t)
			bts, err := json.Marshal(request)
			require.NoError(t, err)

			r := httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set("db", db)
			c.Request = r

			test.AuthFunc(t, db, c)

			a := API{}

			a.CreateDestination(c)

			test.VerifyFunc(t, r, w)
		})
	}
}

func TestLogin(t *testing.T) {
	cases := []struct {
		Name        string
		RequestFunc func(t *testing.T) *http.Request
		VerifyFunc  func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder)
	}{
		{
			Name: "NilRequest",
			RequestFunc: func(t *testing.T) *http.Request {
				return httptest.NewRequest(http.MethodPost, "/v1/login", nil)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			Name: "OktaNil",
			RequestFunc: func(t *testing.T) *http.Request {
				r := &api.LoginRequest{
					Okta: nil,
				}

				bts, err := json.Marshal(r)
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			Name: "OktaEmpty",
			RequestFunc: func(t *testing.T) *http.Request {
				r := &api.LoginRequest{
					Okta: &api.LoginRequestOkta{},
				}

				bts, err := json.Marshal(r)
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			Name: "OktaMissingDomain",
			RequestFunc: func(t *testing.T) *http.Request {
				r := &api.LoginRequest{
					Okta: &api.LoginRequestOkta{
						Code: "code",
					},
				}

				bts, err := json.Marshal(r)
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			Name: "OktaMissingCodeRequest",
			RequestFunc: func(t *testing.T) *http.Request {
				r := &api.LoginRequest{
					Okta: &api.LoginRequestOkta{
						Domain: "test.okta.com",
					},
				}

				bts, err := json.Marshal(r)
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(bts))
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
	}

	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			_, db := configure(t, nil)

			r := test.RequestFunc(t)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set("db", db)
			c.Request = r

			a := &API{}

			a.Login(c)

			test.VerifyFunc(t, r, w)
		})
	}
}

func TestLoginOkta(t *testing.T) {
	_, db := configure(t, nil)

	testOkta := new(mocks.Okta)
	testOkta.On("EmailFromCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("jbond@infrahq.com", nil)

	request := api.LoginRequest{
		Okta: &api.LoginRequestOkta{
			Domain: "test.okta.com",
			Code:   "code",
		},
	}

	bts, err := json.Marshal(request)
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
	cases := []struct {
		Name        string
		AuthFunc    func(t *testing.T, db *gorm.DB, c *gin.Context)
		RequestFunc func(t *testing.T, c *gin.Context) *http.Request
		HandlerFunc func(a *API, c *gin.Context)
		VerifyFunc  func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder)
	}{
		// /v1/users
		{
			Name: "GetUser",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionUserRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				c.Params = append(c.Params, gin.Param{Key: "id", Value: userBond.ID.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/users/%s", userBond.ID), nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.GetUser(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var user api.User
				err := json.NewDecoder(w.Body).Decode(&user)
				require.NoError(t, err)
				require.Equal(t, "jbond@infrahq.com", user.Email)
			},
		},
		{
			Name: "GetUserEmptyID",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionUserRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/users/", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.GetUser(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			Name: "GetUserUnknownUser",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionUserRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/users/%s", id), nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.GetUser(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			Name: "ListUsers",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionUserRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/users", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListUsers(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var users []api.User
				err := json.NewDecoder(w.Body).Decode(&users)
				require.NoError(t, err)
				require.Len(t, users, 2)
				require.ElementsMatch(t, []string{"jbond@infrahq.com", "jbourne@infrahq.com"}, []string{users[0].Email, users[1].Email})
			},
		},
		{
			Name: "ListUsersByEmail",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionUserRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/users?email=jbond@infrahq.com", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListUsers(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var users []api.User
				err := json.NewDecoder(w.Body).Decode(&users)
				require.NoError(t, err)
				require.Len(t, users, 1)
				require.Equal(t, "jbond@infrahq.com", users[0].Email)
			},
		},
		{
			Name: "ListUsersUnknownEmail",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionUserRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/users?email=unknown@infrahq.com", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListUsers(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var users []api.User
				err := json.NewDecoder(w.Body).Decode(&users)
				require.NoError(t, err)
				require.Len(t, users, 0)
			},
		},

		// /v1/groups
		{
			Name: "GetGroup",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGroupRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				c.Params = append(c.Params, gin.Param{Key: "id", Value: groupEveryone.ID.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/groups/%s", groupEveryone.ID), nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.GetGroup(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var group api.Group
				err := json.NewDecoder(w.Body).Decode(&group)
				require.NoError(t, err)
				require.Equal(t, "Everyone", group.Name)
			},
		},
		{
			Name: "GetGroupEmptyID",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGroupRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/groups/", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.GetGroup(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			Name: "GetGroupUnknownGroup",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGroupRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/groups/%s", id), nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.GetGroup(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			Name: "ListGroups",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGroupRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/groups", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListGroups(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var groups []api.Group
				err := json.NewDecoder(w.Body).Decode(&groups)
				require.NoError(t, err)
				require.Len(t, groups, 2)
				require.ElementsMatch(t, []string{"Everyone", "Engineering"}, []string{groups[0].Name, groups[1].Name})
			},
		},
		{
			Name: "ListGroupsByName",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGroupRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/groups?name=Everyone", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListGroups(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var groups []api.Group
				err := json.NewDecoder(w.Body).Decode(&groups)
				require.NoError(t, err)
				require.Len(t, groups, 1)
				require.Equal(t, "Everyone", groups[0].Name)
			},
		},
		{
			Name: "ListGroupsUnknownName",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGroupRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/groups?name=unknown", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListGroups(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var groups []api.Group
				err := json.NewDecoder(w.Body).Decode(&groups)
				require.NoError(t, err)
				require.Len(t, groups, 0)
			},
		},
		{
			Name: "GetGrantEmptyID",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGrantRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/grants/", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.GetGrant(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			Name: "GetGrantUnknownGrant",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGrantRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/grants/%s", id), nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.GetGrant(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			Name: "ListGrants",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGrantRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/grants", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListGrants(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var grants []api.Grant
				err := json.NewDecoder(w.Body).Decode(&grants)
				require.NoError(t, err)
				require.Len(t, grants, 8)
			},
		},
		{
			Name: "ListGrantsByDestinationID",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGrantRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/grants?destination=%s", destinationAAA.ID), nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListGrants(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var grants []api.Grant
				err := json.NewDecoder(w.Body).Decode(&grants)
				require.NoError(t, err)
				require.Len(t, grants, 2)
				require.ElementsMatch(t, []string{"AAA", "AAA"}, []string{
					grants[0].Destination.Name,
					grants[1].Destination.Name,
				})
				require.ElementsMatch(t, []string{"writer", "admin"}, []string{grants[0].Kubernetes.Name, grants[1].Kubernetes.Name})
				require.ElementsMatch(t, []api.GrantKubernetesKind{
					api.GrantKubernetesKindClusterRole,
					api.GrantKubernetesKindClusterRole,
				}, []api.GrantKubernetesKind{
					grants[0].Kubernetes.Kind,
					grants[1].Kubernetes.Kind,
				})
			},
		},
		{
			Name: "ListGrantsByKind",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGrantRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/grants?kind=kubernetes", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListGrants(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var grants []api.Grant
				err := json.NewDecoder(w.Body).Decode(&grants)
				require.NoError(t, err)
				require.Len(t, grants, 8)

				for _, r := range grants {
					require.Equal(t, api.GrantKindKubernetes, r.Kind)
				}
			},
		},
		{
			Name: "ListGrantsCombo",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGrantRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/grants?destination=%s&kind=kubernetes", destinationCCC.ID), nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListGrants(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var grants []api.Grant
				err := json.NewDecoder(w.Body).Decode(&grants)
				require.NoError(t, err)
				require.Len(t, grants, 4)

				for _, r := range grants {
					require.Equal(t, api.GrantKindKubernetes, r.Kind)
				}
			},
		},
		{
			Name: "ListGrantsBadGrantType",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGrantRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/grants?destination=%s&kind=grant&name=audit", id), nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListGrants(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)

				var errMsg api.Error
				err := json.NewDecoder(w.Body).Decode(&errMsg)
				require.NoError(t, err)
				require.EqualValues(t, http.StatusBadRequest, errMsg.Code)
				require.Equal(t, "bad request: unknown grant kind: \"grant\"", errMsg.Message)
			},
		},
		{
			Name: "ListGrantsNotFound",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGrantRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/grants?destination=%s&kind=kubernetes&name=audit", id), nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListGrants(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var grants []api.Grant
				err := json.NewDecoder(w.Body).Decode(&grants)
				require.NoError(t, err)
				require.Len(t, grants, 0)
			},
		},
		{
			Name: "ListAPITokens",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				issueAPIToken(t, db, string(access.PermissionAPITokenRead))
				c.Set("permissions", string(access.PermissionAPITokenRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/api-tokens", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListAPITokens(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var apiTokens []api.InfraAPIToken
				err := json.NewDecoder(w.Body).Decode(&apiTokens)
				require.NoError(t, err)
				require.Len(t, apiTokens, 1)
			},
		},
		{
			Name: "CreateToken",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				_, token, err := access.IssueUserToken(c, "jbond@infrahq.com", time.Hour*1)
				require.NoError(t, err)

				c.Set("authentication", token.SessionToken())
				c.Set("permissions", string(access.PermissionCredentialCreate))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.TokenRequest{
					Destination: "AAA",
				}

				bts, err := json.Marshal(request)
				require.NoError(t, err)
				return httptest.NewRequest(http.MethodPost, "/v1/token", bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.CreateToken(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var token api.Token
				err := json.NewDecoder(w.Body).Decode(&token)
				require.NoError(t, err)
				require.NotEmpty(t, token.Token)
				require.WithinDuration(t, time.Unix(token.Expires, 0), time.Now(), time.Hour*1)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			_, db := configure(t, nil)

			_, err := data.InitializeSettings(db)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			r := tc.RequestFunc(t, c)
			c.Set("db", db)
			c.Request = r

			tc.AuthFunc(t, db, c)

			a := API{}

			tc.HandlerFunc(&a, c)
			tc.VerifyFunc(t, r, w)
		})
	}
}

func TestProvider(t *testing.T) {
	cases := []struct {
		Name        string
		AuthFunc    func(t *testing.T, db *gorm.DB, c *gin.Context)
		RequestFunc func(t *testing.T, c *gin.Context) *http.Request
		HandlerFunc func(a *API, c *gin.Context)
		VerifyFunc  func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder)
	}{
		{
			Name: "CreateOK",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderCreate))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.ProviderRequest{
					Kind:         api.ProviderKindOkta,
					Domain:       "domain.okta.com",
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				}

				bts, err := json.Marshal(request)
				require.NoError(t, err)
				return httptest.NewRequest(http.MethodPost, "/v1/providers", bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.CreateProvider(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, w.Code)

				var provider api.Provider
				err := json.NewDecoder(w.Body).Decode(&provider)
				require.NoError(t, err)
				require.Equal(t, "domain.okta.com", provider.Domain)
				require.Equal(t, "client-id", provider.ClientID)
			},
		},
		{
			Name: "CreateOK/Okta",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderCreate))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.ProviderRequest{
					Kind:         api.ProviderKindOkta,
					Domain:       "domain.okta.com",
					ClientID:     "client-id",
					ClientSecret: "client-secret",
					Okta: &api.ProviderOkta{
						APIToken: "api-token",
					},
				}

				bts, err := json.Marshal(request)
				require.NoError(t, err)
				return httptest.NewRequest(http.MethodPost, "/v1/providers", bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.CreateProvider(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, w.Code)

				var provider api.Provider
				err := json.NewDecoder(w.Body).Decode(&provider)
				require.NoError(t, err)
				require.Equal(t, "domain.okta.com", provider.Domain)
				require.Equal(t, "client-id", provider.ClientID)
			},
		},
		{
			Name: "CreateNoKind",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderCreate))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.ProviderRequest{
					Domain:       "domain.okta.com",
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				}

				bts, err := json.Marshal(request)
				require.NoError(t, err)
				return httptest.NewRequest(http.MethodPost, "/v1/providers", bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.CreateProvider(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			Name: "CreateUnknownKind",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderCreate))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.ProviderRequest{
					Kind:         api.ProviderKind("unknown"),
					Domain:       "domain.okta.com",
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				}

				bts, err := json.Marshal(request)
				require.NoError(t, err)
				return httptest.NewRequest(http.MethodPost, "/v1/providers", bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.CreateProvider(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			Name: "CreateNoAuthorization",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", "")
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.ProviderRequest{
					Kind:         api.ProviderKindOkta,
					Domain:       "domain.okta.com",
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				}

				bts, err := json.Marshal(request)
				require.NoError(t, err)
				return httptest.NewRequest(http.MethodPost, "/v1/providers", bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.CreateProvider(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
			},
		},
		{
			Name: "CreateBadPermissions",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderUpdate))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.ProviderRequest{
					Kind:         api.ProviderKindOkta,
					Domain:       "domain.okta.com",
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				}

				bts, err := json.Marshal(request)
				require.NoError(t, err)
				return httptest.NewRequest(http.MethodPost, "/v1/providers", bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.CreateProvider(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
			},
		},
		{
			Name: "CreateDuplicate",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderCreate))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.ProviderRequest{
					Kind:         api.ProviderKindOkta,
					Domain:       "test.okta.com",
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				}

				bts, err := json.Marshal(request)
				require.NoError(t, err)
				return httptest.NewRequest(http.MethodPost, "/v1/providers", bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.CreateProvider(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, w.Code)
			},
		},
		{
			Name: "Update",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderUpdate))

				// test.okta.com is created by configure()
				providers, err := data.ListProviders(db, &models.Provider{})
				require.NoError(t, err)
				require.Len(t, providers, 1)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: providers[0].ID.String()})
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.ProviderRequest{
					Domain: "test2.okta.com",
					Kind:   api.ProviderKindOkta,
				}

				bts, err := json.Marshal(request)
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/providers/%s", c.Param("id")), bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.UpdateProvider(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var provider api.Provider
				err := json.NewDecoder(w.Body).Decode(&provider)
				require.NoError(t, err)
				require.Equal(t, "test2.okta.com", provider.Domain)
				require.Equal(t, "plaintext:0oapn0qwiQPiMIyR35d6", provider.ClientID)
			},
		},
		{
			Name: "UpdateNotFound",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderUpdate))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.ProviderRequest{
					Domain: "domain.okta.com",
					Kind:   api.ProviderKindOkta,
				}

				bts, err := json.Marshal(request)
				require.NoError(t, err)

				id, err := uuid.NewUUID()
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
				return httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/providers/%s", c.Param("id")), bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.UpdateProvider(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			Name: "Get",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				c.Params = append(c.Params, gin.Param{Key: "id", Value: providerOkta.ID.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/providers/%s", providerOkta.ID), nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.GetProvider(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var provider api.Provider
				err := json.NewDecoder(w.Body).Decode(&provider)
				require.NoError(t, err)
				require.Equal(t, "test.okta.com", provider.Domain)
				require.Equal(t, "plaintext:0oapn0qwiQPiMIyR35d6", provider.ClientID)
			},
		},
		{
			Name: "GetEmptyID",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.GetProvider(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			Name: "GetUnknownProvider",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/providers/%s", id), nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.GetProvider(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			Name: "List",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/providers", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListProviders(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var providers []api.Provider
				err := json.NewDecoder(w.Body).Decode(&providers)
				require.NoError(t, err)
				require.Len(t, providers, 1)
				require.Equal(t, "test.okta.com", providers[0].Domain)
			},
		},
		{
			Name: "ListByKind",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/providers?kind=okta", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListProviders(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var providers []api.Provider
				err := json.NewDecoder(w.Body).Decode(&providers)
				require.NoError(t, err)
				require.Len(t, providers, 1)
				require.Equal(t, "test.okta.com", providers[0].Domain)
			},
		},
		{
			Name: "ListByDomain",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/providers?domain=test.okta.com", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListProviders(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var providers []api.Provider
				err := json.NewDecoder(w.Body).Decode(&providers)
				require.NoError(t, err)
				require.Len(t, providers, 1)
				require.Equal(t, "test.okta.com", providers[0].Domain)
			},
		},
		{
			Name: "ListNotFound",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/providers?domain=nonexistent.okta.com", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListProviders(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var providers []api.Provider
				err := json.NewDecoder(w.Body).Decode(&providers)
				require.NoError(t, err)
				require.Len(t, providers, 0)
			},
		},
		{
			Name: "ListSensitiveInformation",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/providers?domain=test.okta.com", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListProviders(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
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
		{
			Name: "Delete",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderDelete))

				// test.okta.com is created by configure()
				providers, err := data.ListProviders(db, &models.Provider{})
				require.NoError(t, err)
				require.Len(t, providers, 1)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: providers[0].ID.String()})
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.ProviderRequest{
					Domain: "test2.okta.com",
					Kind:   api.ProviderKindOkta,
				}

				bts, err := json.Marshal(request)
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/providers/%s", c.Param("id")), bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.DeleteProvider(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNoContent, w.Code)
			},
		},
		{
			Name: "DeleteNotFound",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionProviderDelete))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.ProviderRequest{
					Domain: "domain.okta.com",
					Kind:   api.ProviderKindOkta,
				}

				bts, err := json.Marshal(request)
				require.NoError(t, err)

				id, err := uuid.NewUUID()
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
				return httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/providers/%s", c.Param("id")), bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.DeleteProvider(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			_, db := configure(t, nil)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set("db", db)

			r := tc.RequestFunc(t, c)
			c.Request = r

			tc.AuthFunc(t, db, c)

			tc.HandlerFunc(&API{}, c)

			tc.VerifyFunc(t, r, w)
		})
	}
}

func TestDestination(t *testing.T) {
	cases := []struct {
		Name        string
		AuthFunc    func(t *testing.T, db *gorm.DB, c *gin.Context)
		RequestFunc func(t *testing.T, c *gin.Context) *http.Request
		HandlerFunc func(a *API, c *gin.Context)
		VerifyFunc  func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder)
	}{

		{
			Name: "CreateOK",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationCreate))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.DestinationRequest{
					Kind:   api.DestinationKindKubernetes,
					NodeID: "test",
					Name:   "test",
					Kubernetes: &api.DestinationKubernetes{
						CA:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}

				bts, err := json.Marshal(request)

				require.NoError(t, err)
				return httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.CreateDestination(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, w.Code)
			},
		},
		{
			Name: "CreateNoKind",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationCreate))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.DestinationRequest{
					Kubernetes: &api.DestinationKubernetes{
						CA:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}

				bts, err := json.Marshal(request)

				require.NoError(t, err)
				return httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.CreateDestination(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			Name: "CreateUnknownKind",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationCreate))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.DestinationRequest{
					Kind: api.DestinationKind("unknown"),
					Kubernetes: &api.DestinationKubernetes{
						CA:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}

				bts, err := json.Marshal(request)

				require.NoError(t, err)
				return httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.CreateDestination(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			Name: "CreateNoAuthorization",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", "")
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.DestinationRequest{
					Kind: api.DestinationKindKubernetes,
					Kubernetes: &api.DestinationKubernetes{
						CA:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}

				bts, err := json.Marshal(request)

				require.NoError(t, err)
				return httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.CreateDestination(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
			},
		},
		{
			Name: "CreateBadPermissions",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", "infra.bad.permissions")
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.DestinationRequest{
					Kind: api.DestinationKindKubernetes,
					Kubernetes: &api.DestinationKubernetes{
						CA:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}

				bts, err := json.Marshal(request)

				require.NoError(t, err)
				return httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.CreateDestination(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
			},
		},
		{
			Name: "Update",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationUpdate))

				// AAA is created by configure()
				aaa, err := data.GetDestination(db, &models.Destination{NodeID: "AAA"})
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: aaa.ID.String()})
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.DestinationRequest{
					NodeID: "AAA",
					Name:   "aaa",
					Kind:   api.DestinationKindKubernetes,
					Labels: []string{},
				}

				bts, err := json.Marshal(request)
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/destinations/%s", c.Param("id")), bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.UpdateDestination(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var destination api.Destination
				err := json.NewDecoder(w.Body).Decode(&destination)
				require.NoError(t, err)
				require.Equal(t, "AAA", destination.NodeID)
				require.Equal(t, "aaa", destination.Name)
			},
		},
		{
			Name: "UpdateNotFound",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationUpdate))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				request := api.DestinationRequest{
					NodeID: "XYZ",
					Name:   "XYZ",
					Kind:   api.DestinationKindKubernetes,
					Labels: []string{},
				}

				bts, err := json.Marshal(request)
				require.NoError(t, err)

				id, err := uuid.NewUUID()
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
				return httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/destinations/%s", c.Param("id")), bytes.NewReader(bts))
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.UpdateDestination(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			Name: "Get",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				c.Params = append(c.Params, gin.Param{Key: "id", Value: destinationAAA.ID.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/destinations/%s", destinationAAA.ID), nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.GetDestination(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var destination api.Destination
				err := json.NewDecoder(w.Body).Decode(&destination)
				require.NoError(t, err)
				require.Equal(t, "AAA", destination.Name)
				require.Equal(t, "AAA", destination.NodeID)
				require.Equal(t, "develop.infrahq.com", destination.Kubernetes.Endpoint)
			},
		},
		{
			Name: "GetEmptyID",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/destinations/", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.GetDestination(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			Name: "GetUnknownDestination",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/destinations/%s", id), nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.GetDestination(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			Name: "List",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/destinations", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListDestinations(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
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
		{
			Name: "ListByKind",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/destinations?kind=kubernetes", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListDestinations(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var destinations []api.Destination
				err := json.NewDecoder(w.Body).Decode(&destinations)
				require.NoError(t, err)
				require.Len(t, destinations, 3)

				for _, d := range destinations {
					require.Equal(t, api.DestinationKindKubernetes, d.Kind)
				}
			},
		},
		{
			Name: "ListByName",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/destinations?name=AAA", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListDestinations(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
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
		{
			Name: "ListCombo",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/destinations?kind=kubernetes&name=AAA", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListDestinations(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
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
		{
			Name: "ListNotFound",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationRead))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/destinations?name=nonexistent", nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.ListDestinations(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var destinations []api.Destination
				err := json.NewDecoder(w.Body).Decode(&destinations)
				require.NoError(t, err)
				require.Len(t, destinations, 0)
			},
		},
		{
			Name: "Delete",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationDelete))

				// AAA is created by configure()
				aaa, err := data.GetDestination(db, &models.Destination{NodeID: "AAA"})
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: aaa.ID.String()})
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/destinations/%s", c.Param("id")), nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.DeleteDestination(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNoContent, w.Code)
			},
		},
		{
			Name: "DeleteNotFound",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationDelete))
			},
			RequestFunc: func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
				return httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/destinations/%s", c.Param("id")), nil)
			},
			HandlerFunc: func(a *API, c *gin.Context) {
				a.DeleteDestination(c)
			},
			VerifyFunc: func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
	}

	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			_, db := configure(t, nil)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set("db", db)

			r := test.RequestFunc(t, c)
			// c.Set("logger", zap.New(zapcore.NewCore()))
			c.Request = r

			test.AuthFunc(t, db, c)

			test.HandlerFunc(&API{}, c)

			test.VerifyFunc(t, r, w)
		})
	}
}

func TestCreateDestinationUpdatesField(t *testing.T) {
	_, db := configure(t, nil)

	destination := &models.Destination{
		Kind:     models.DestinationKindKubernetes,
		NodeID:   "node-id",
		Name:     "name",
		Endpoint: "endpoint",
		Kubernetes: models.DestinationKubernetes{
			CA: "ca",
		},
	}
	err := data.CreateDestination(db, destination)
	require.NoError(t, err)

	request := api.DestinationRequest{
		ID:     uuid.New(),
		Kind:   api.DestinationKindKubernetes,
		NodeID: destination.NodeID,
		Name:   "updated-name",
		Kubernetes: &api.DestinationKubernetes{
			CA:       "updated-ca",
			Endpoint: "updated-endpoint",
		},
	}

	bts, err := json.Marshal(request)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/v1/destinations", bytes.NewReader(bts))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("db", db)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: destination.ID.String()})
	c.Request = r

	c.Set("permissions", string(access.PermissionDestinationCreate)+" "+string(access.PermissionDestinationUpdate))

	a := API{}

	c.Request = httptest.NewRequest("PUT", "/v1/destinations/"+destination.ID.String(), bytes.NewReader(bts))
	a.UpdateDestination(c)

	require.Equal(t, http.StatusOK, w.Code)

	var body api.Destination
	err = json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	require.Equal(t, "node-id", body.NodeID)
	require.Equal(t, "updated-name", body.Name)
	require.Equal(t, "updated-ca", body.Kubernetes.CA)
	require.Equal(t, "updated-endpoint", body.Kubernetes.Endpoint)

	destinations, err := data.ListDestinations(db, &models.Destination{NodeID: "node-id"})
	require.NoError(t, err)
	require.Len(t, destinations, 1)
	require.Equal(t, body.ID, destinations[0].ID.String())
	require.Equal(t, body.NodeID, destinations[0].NodeID)
	require.Equal(t, body.Name, destinations[0].Name)
	require.Equal(t, body.Kubernetes.Endpoint, destinations[0].Endpoint)
	require.Equal(t, body.Kubernetes.CA, destinations[0].Kubernetes.CA)
}

func TestCreateAPIToken(t *testing.T) {
	_, db := configure(t, nil)

	request := api.InfraAPITokenCreateRequest{
		Name:        "tmp",
		Permissions: []string{string(access.PermissionAllInfra)},
	}

	bts, err := json.Marshal(request)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/v1/api-tokens", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("db", db)
	c.Set("permissions", string(access.PermissionAllInfra))
	c.Request = r

	a := API{}

	a.CreateAPIToken(c)

	require.Equal(t, http.StatusCreated, w.Code)

	var body api.InfraAPITokenCreateResponse
	err = json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)

	newr := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	neww := httptest.NewRecorder()
	newc, _ := gin.CreateTestContext(neww)
	newc.Set("db", db)
	newc.Set("permissions", string(access.PermissionUserRead))
	newc.Request = newr

	a.ListUsers(newc)

	require.Equal(t, http.StatusOK, neww.Code)
}

func TestDeleteAPIToken(t *testing.T) {
	_, db := configure(t, nil)

	permissions := strings.Join([]string{
		string(access.PermissionUserRead),
		string(access.PermissionAPITokenDelete),
	}, " ")

	apiToken := issueAPIToken(t, db, permissions)

	oldr := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	oldw := httptest.NewRecorder()
	oldc, _ := gin.CreateTestContext(oldw)
	oldc.Set("db", db)
	oldc.Set("permissions", permissions)
	oldc.Request = oldr

	a := API{}

	a.ListUsers(oldc)

	require.Equal(t, http.StatusOK, oldw.Code)

	r := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/api-tokens/%s", apiToken.ID.String()), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("db", db)
	c.Set("permissions", permissions)
	c.Request = r
	c.Params = append(c.Params, gin.Param{Key: "id", Value: apiToken.ID.String()})

	a.DeleteAPIToken(c)

	require.Equal(t, http.StatusNoContent, w.Code)
}

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

type testCase struct {
	Setup   func(*testing.T, *gorm.DB, *gin.Context)
	Request func(*testing.T, *gin.Context) *http.Request
	Handle  func(*testing.T, *gin.Context)
	Verify  func(*testing.T, *http.Request, *httptest.ResponseRecorder)
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

	apiToken, _, err := data.CreateAPIToken(db, apiToken, &models.Token{})
	require.NoError(t, err)

	return apiToken
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
			_, db := configure(t, nil)

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
	_, db := configure(t, nil)

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
				c.Set("permissions", string(access.PermissionUserRead))
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
				c.Set("permissions", string(access.PermissionUserRead))
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
				c.Set("permissions", string(access.PermissionUserRead))
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
				c.Set("permissions", string(access.PermissionUserRead))
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
				c.Set("permissions", string(access.PermissionUserRead))
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
				c.Set("permissions", string(access.PermissionUserRead))
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
				c.Set("permissions", string(access.PermissionGroupRead))
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
				c.Set("permissions", string(access.PermissionGroupRead))
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
				c.Set("permissions", string(access.PermissionGroupRead))
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
				c.Set("permissions", string(access.PermissionGroupRead))
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
				c.Set("permissions", string(access.PermissionGroupRead))
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
				c.Set("permissions", string(access.PermissionGroupRead))
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

		// /v1/grants
		"GetGrantEmptyID": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGrantRead))
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/grants/", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetGrant(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		"GetGrantUnknownGrant": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGrantRead))
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/grants/%s", id), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.GetGrant(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		"ListGrants": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGrantRead))
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/grants", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListGrants(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var grants []api.Grant
				err := json.NewDecoder(w.Body).Decode(&grants)
				require.NoError(t, err)
				require.Len(t, grants, 8)
			},
		},
		"ListGrantsByDestinationID": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGrantRead))
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/grants?destination=%s", destinationAAA.ID), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListGrants(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
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
					api.GRANTKUBERNETESKIND_CLUSTER_ROLE,
					api.GRANTKUBERNETESKIND_CLUSTER_ROLE,
				}, []api.GrantKubernetesKind{
					grants[0].Kubernetes.Kind,
					grants[1].Kubernetes.Kind,
				})
			},
		},
		"ListGrantsByKind": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGrantRead))
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/grants?kind=kubernetes", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListGrants(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var grants []api.Grant
				err := json.NewDecoder(w.Body).Decode(&grants)
				require.NoError(t, err)
				require.Len(t, grants, 8)

				for _, r := range grants {
					require.Equal(t, api.GRANTKIND_KUBERNETES, r.Kind)
				}
			},
		},
		"ListGrantsCombo": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGrantRead))
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/grants?destination=%s&kind=kubernetes", destinationCCC.ID), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListGrants(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var grants []api.Grant
				err := json.NewDecoder(w.Body).Decode(&grants)
				require.NoError(t, err)
				require.Len(t, grants, 4)

				for _, r := range grants {
					require.Equal(t, api.GRANTKIND_KUBERNETES, r.Kind)
				}
			},
		},
		"ListGrantsNotFound": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionGrantRead))
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				id, err := uuid.NewUUID()
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/grants?destination=%s&kind=grant&name=audit", id), nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListGrants(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var grants []api.Grant
				err := json.NewDecoder(w.Body).Decode(&grants)
				require.NoError(t, err)
				require.Len(t, grants, 0)
			},
		},

		// /v1/destinations
		"GetDestination": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationRead))
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
				c.Set("permissions", string(access.PermissionDestinationRead))
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
				c.Set("permissions", string(access.PermissionDestinationRead))
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
				c.Set("permissions", string(access.PermissionDestinationRead))
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
				c.Set("permissions", string(access.PermissionDestinationRead))
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
				c.Set("permissions", string(access.PermissionDestinationRead))
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
				c.Set("permissions", string(access.PermissionDestinationRead))
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
				c.Set("permissions", string(access.PermissionDestinationRead))
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

		// /v1/api-tokens
		"ListAPITokens": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				issueAPIToken(t, db, string(access.PermissionAPITokenRead))
				c.Set("permissions", string(access.PermissionAPITokenRead))
			},
			"requestFunc": func(t *testing.T, c *gin.Context) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v1/api-tokens", nil)
			},
			"func": func(a *API, c *gin.Context) {
				a.ListAPITokens(c)
			},
			"verifyFunc": func(t *testing.T, r *http.Request, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)

				var apiTokens []api.InfraAPIToken
				err := json.NewDecoder(w.Body).Decode(&apiTokens)
				require.NoError(t, err)
				require.Len(t, apiTokens, 1)
			},
		},

		// /v1/tokens
		"CreateToken": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				_, token, err := access.IssueUserToken(c, "jbond@infrahq.com", time.Hour*1)
				require.NoError(t, err)

				c.Set("authentication", token.SessionToken())
				c.Set("permissions", string(access.PermissionCredentialCreate))
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
			_, db := configure(t, nil)

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

func TestCreateAPIToken(t *testing.T) {
	db := configure(t, nil)

	request := api.InfraAPITokenCreateRequest{
		Name:        "tmp",
		Permissions: []string{string(access.PermissionAllAlternate)},
	}

	bts, err := request.MarshalJSON()
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/v1/api-tokens", bytes.NewReader(bts))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("db", db)
	c.Set("permissions", string(access.PermissionAllAlternate))
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
	db, _ := configure(t, nil)

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

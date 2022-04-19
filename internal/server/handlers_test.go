package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestAPI_ListIdentities(t *testing.T) {
	srv := setupServer(t, withDefaultAdminAccessKey)
	adminAccessKey := srv.options.AdminAccessKey
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	createID := func(t *testing.T, name string, kind string) {
		t.Helper()
		var buf bytes.Buffer
		body := api.CreateIdentityRequest{Name: name, Kind: kind}
		err := json.NewEncoder(&buf).Encode(body)
		assert.NilError(t, err)

		req, err := http.NewRequest(http.MethodPost, "/v1/identities", &buf)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())
	}
	createID(t, "me@example.com", "user")
	createID(t, "other@example.com", "user")
	createID(t, "HAL", "machine")
	createID(t, "other-HAL", "machine")

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		req, err := http.NewRequest(http.MethodGet, tc.urlPath, nil)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey)

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"no name match": {
			urlPath: "/v1/identities?name=doesnotmatch",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)
				assert.Equal(t, resp.Body.String(), `[]`)
			},
		},
		"name match": {
			urlPath: "/v1/identities?name=me@example.com",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				var actual []api.Identity
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				expected := []api.Identity{
					{Name: "me@example.com", Kind: "user"},
				}
				assert.DeepEqual(t, actual, expected, cmpAPIIdentityShallow)
			},
		},
		"no filter": {
			urlPath: "/v1/identities",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				var actual []api.Identity
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				expected := []api.Identity{
					{Name: "HAL", Kind: "machine"},
					{Name: "admin", Kind: "machine"},
					{Name: "connector", Kind: "machine"},
					{Name: "me@example.com", Kind: "user"},
					{Name: "other-HAL", Kind: "machine"},
					{Name: "other@example.com", Kind: "user"},
				}
				assert.DeepEqual(t, actual, expected, cmpAPIIdentityShallow)
			},
		},
		"no authorization": {
			urlPath: "/v1/identities",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		// TODO: assert full JSON response
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

var cmpAPIIdentityShallow = cmp.Comparer(func(x, y api.Identity) bool {
	return x.Name == y.Name && x.Kind == y.Kind
})

func TestListProviders(t *testing.T) {
	s := setupServer(t, withDefaultAdminAccessKey)
	routes := s.GenerateRoutes(prometheus.NewRegistry())

	testProvider := &models.Provider{Name: "mokta"}

	err := data.CreateProvider(s.db, testProvider)
	assert.NilError(t, err)

	dbProviders, err := data.ListProviders(s.db)
	assert.NilError(t, err)
	assert.Equal(t, len(dbProviders), 2)

	req, err := http.NewRequest(http.MethodGet, "/v1/providers", nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", "Bearer "+s.options.AdminAccessKey)

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

	apiProviders := make([]api.Provider, 0)
	err = json.Unmarshal(resp.Body.Bytes(), &apiProviders)
	assert.NilError(t, err)

	assert.Equal(t, len(apiProviders), 1)
	assert.Equal(t, apiProviders[0].Name, "mokta")
}

func TestDeleteProvider(t *testing.T) {
	s := setupServer(t, withDefaultAdminAccessKey)

	routes := s.GenerateRoutes(prometheus.NewRegistry())

	testProvider := &models.Provider{
		Name: "mokta",
	}

	err := data.CreateProvider(s.db, testProvider)
	assert.NilError(t, err)

	route := fmt.Sprintf("/v1/providers/%s", testProvider.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", "Bearer "+s.options.AdminAccessKey)

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
}

// withDefaultAdminAccessKey may be used with setupServer to setup the server
// with a default AdminAccessKey. The value for the key can be retrieved from
// server.options.AdminAccessKey.
func withDefaultAdminAccessKey(_ *testing.T, opts *Options) {
	opts.AdminAccessKey = "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb"
}

func TestDeleteProvider_NoDeleteInternalProvider(t *testing.T) {
	s := setupServer(t, withDefaultAdminAccessKey)

	routes := s.GenerateRoutes(prometheus.NewRegistry())

	route := fmt.Sprintf("/v1/providers/%s", s.InternalProvider.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", "Bearer "+s.options.AdminAccessKey)

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code, resp.Body.String())
}

func TestCreateIdentity(t *testing.T) {
	s := setupServer(t)

	admin := &models.Identity{Name: "admin@example.com", Kind: models.UserKind}
	err := data.CreateIdentity(s.db, admin)
	assert.NilError(t, err)

	adminGrant := &models.Grant{
		Subject:   admin.PolyID(),
		Privilege: models.InfraAdminRole,
		Resource:  "infra",
	}
	err = data.CreateGrant(s.db, adminGrant)
	assert.NilError(t, err)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("db", s.db)
	c.Set("identity", admin)

	handler := API{
		server: s,
	}

	t.Run("new unlinked user", func(t *testing.T) {
		req := &api.CreateIdentityRequest{
			Name: "test-create-identity@example.com",
			Kind: "user",
		}

		resp, err := handler.CreateIdentity(c, req)

		assert.NilError(t, err)
		assert.Equal(t, "test-create-identity@example.com", resp.Name)
		assert.Check(t, resp.OneTimePassword == "")
	})

	t.Run("new infra user gets one time password", func(t *testing.T) {
		req := &api.CreateIdentityRequest{
			Name:               "test-infra-identity@example.com",
			Kind:               "user",
			SetOneTimePassword: true,
		}

		resp, err := handler.CreateIdentity(c, req)
		assert.NilError(t, err)

		assert.NilError(t, err)
		assert.Equal(t, "test-infra-identity@example.com", resp.Name)
		assert.Check(t, resp.OneTimePassword != "")
	})

	t.Run("existing unlinked user gets password", func(t *testing.T) {
		req := &api.CreateIdentityRequest{
			Name: "test-link-identity@example.com",
			Kind: "user",
		}

		_, err := handler.CreateIdentity(c, req)
		assert.NilError(t, err)

		req = &api.CreateIdentityRequest{
			Name:               "test-link-identity@example.com",
			Kind:               "user",
			SetOneTimePassword: true,
		}

		resp, err := handler.CreateIdentity(c, req)
		assert.NilError(t, err)

		assert.NilError(t, err)
		assert.Equal(t, "test-link-identity@example.com", resp.Name)
		assert.Check(t, resp.OneTimePassword != "")
	})
}

func TestDeleteIdentity(t *testing.T) {
	s := setupServer(t, withDefaultAdminAccessKey)

	routes := s.GenerateRoutes(prometheus.NewRegistry())

	testUser := &models.Identity{
		Name: "test",
		Kind: models.UserKind,
	}

	err := data.CreateIdentity(s.db, testUser)
	assert.NilError(t, err)

	route := fmt.Sprintf("/v1/identities/%s", testUser.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", "Bearer "+s.options.AdminAccessKey)

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
}

func TestDeleteIdentity_NoDeleteInternalIdentities(t *testing.T) {
	s := setupServer(t, withDefaultAdminAccessKey)

	routes := s.GenerateRoutes(prometheus.NewRegistry())

	for name := range s.InternalIdentities {
		t.Run(name, func(t *testing.T) {
			route := fmt.Sprintf("/v1/identities/%s", s.InternalIdentities["admin"].ID)
			req, err := http.NewRequest(http.MethodDelete, route, nil)
			assert.NilError(t, err)

			req.Header.Add("Authorization", "Bearer "+s.options.AdminAccessKey)

			resp := httptest.NewRecorder()
			routes.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusForbidden, resp.Code, resp.Body.String())
		})
	}
}

func TestDeleteIdentity_NoDeleteSelf(t *testing.T) {
	s := setupServer(t)

	routes := s.GenerateRoutes(prometheus.NewRegistry())

	testUser := &models.Identity{
		Name: "test",
		Kind: models.UserKind,
	}

	err := data.CreateIdentity(s.db, testUser)
	assert.NilError(t, err)

	testAccessKey, err := data.CreateAccessKey(s.db, &models.AccessKey{
		Name:       "test",
		IssuedFor:  testUser.ID,
		ExpiresAt:  time.Now().Add(time.Hour),
		ProviderID: s.InternalProvider.ID,
	})
	assert.NilError(t, err)

	route := fmt.Sprintf("/v1/identities/%s", testUser.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", testAccessKey))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code, resp.Body.String())
}

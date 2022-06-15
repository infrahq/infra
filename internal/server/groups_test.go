package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func createIdentities(t *testing.T, db *gorm.DB, identities ...*models.Identity) {
	t.Helper()
	for i := range identities {
		err := data.CreateIdentity(db, identities[i])
		assert.NilError(t, err, identities[i].Name)
	}
}

func createGroups(t *testing.T, db *gorm.DB, groups ...*models.Group) {
	t.Helper()
	for i := range groups {
		err := data.CreateGroup(db, groups[i])
		assert.NilError(t, err, groups[i].Name)
	}
}

func TestAPI_ListGroups(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	var (
		humans = models.Group{Name: "humans"}
		second = models.Group{Name: "second"}
		others = models.Group{Name: "others"}
	)

	createGroups(t, srv.db, &humans, &second)

	var (
		idInGroup = models.Identity{
			Name:   "inagroup@example.com",
			Groups: []models.Group{humans, second},
		}
		idOther = models.Identity{
			Name:   "other@example.com",
			Groups: []models.Group{others},
		}
	)

	createIdentities(t, srv.db, &idInGroup, &idOther)

	token := &models.AccessKey{
		IssuedFor:  idInGroup.ID,
		ProviderID: data.InfraProvider(srv.db).ID,
		ExpiresAt:  time.Now().Add(10 * time.Second),
	}

	accessKey, err := data.CreateAccessKey(srv.db, token)
	assert.NilError(t, err)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		req, err := http.NewRequest(http.MethodGet, tc.urlPath, nil)
		assert.NilError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessKey)
		req.Header.Add("Infra-Version", "0.13.0")

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"not authenticated": {
			urlPath: "/api/groups",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		"not authorized, wrong identity": {
			urlPath: "/api/groups?userID=" + idOther.ID.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden, resp.Body.String())
			},
		},
		"not authorized, no identity in query": {
			urlPath: "/api/groups?name=humans",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden, resp.Body.String())
			},
		},
		"authorized by grant": {
			urlPath: "/api/groups",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				var actual api.ListResponse[api.Group]
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				assert.Equal(t, len(actual.Items), 3)
			},
		},
		"authorized by group membership": {
			urlPath: "/api/groups?userID=" + idInGroup.ID.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				var actual api.ListResponse[api.Group]
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				assert.Equal(t, len(actual.Items), 2)
			},
		},
		"version 0.12.2 - list groups": {
			urlPath: "/v1/groups?name=humans",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Infra-Version", "0.12.2")
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				expected := jsonUnmarshal(t, fmt.Sprintf(`
[
	{
		"id": "%[1]v",
		"name": "humans",
		"created": "%[2]v",
		"updated": "%[2]v"
	}
]`,
					humans.ID.String(),
					time.Now().UTC().Format(time.RFC3339),
				))
				actual := jsonUnmarshal(t, resp.Body.String())
				assert.DeepEqual(t, actual, expected, cmpAPIGrantJSON)
			},
		},
		"version 0.13.0 - list user groups": {
			urlPath: fmt.Sprintf("/api/users/%v/groups", idInGroup.ID),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Add("Infra-Version", "0.13.0")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				expected := jsonUnmarshal(t, fmt.Sprintf(`
{
	"count": 2,
	"items": [{
		"id": "%[1]v",
		"name": "humans",
		"created": "%[3]v",
		"updated": "%[3]v"
	},
	{
		"id": "%[2]v",
		"name": "second",
		"created": "%[3]v",
		"updated": "%[3]v"
	}]
}`,
					humans.ID,
					second.ID,
					time.Now().UTC().Format(time.RFC3339),
				))
				actual := jsonUnmarshal(t, resp.Body.String())
				assert.DeepEqual(t, actual, expected, cmpAPIGrantJSON)
			},
		},
		"full JSON response": {
			urlPath: "/api/groups?name=humans",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				expected := jsonUnmarshal(t, fmt.Sprintf(`
{
	"count": 1,
	"items": [{
		"id": "%[1]v",
		"name": "humans",
		"created": "%[2]v",
		"updated": "%[2]v"
	}]
}`,
					humans.ID.String(),
					time.Now().UTC().Format(time.RFC3339),
				))
				actual := jsonUnmarshal(t, resp.Body.String())
				assert.DeepEqual(t, actual, expected, cmpAPIGrantJSON)
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestAPI_DeleteGroup(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	var humans = models.Group{Name: "humans"}

	createGroups(t, srv.db, &humans)

	var (
		inGroup = models.Identity{
			Name:   "inagroup@example.com",
			Groups: []models.Group{humans},
		}
	)

	createIdentities(t, srv.db, &inGroup)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	token := &models.AccessKey{
		IssuedFor:  inGroup.ID,
		ProviderID: data.InfraProvider(srv.db).ID,
		ExpiresAt:  time.Now().Add(10 * time.Second),
	}

	accessKey, err := data.CreateAccessKey(srv.db, token)
	assert.NilError(t, err)

	run := func(t *testing.T, tc testCase) {
		req, err := http.NewRequest(http.MethodDelete, tc.urlPath, nil)
		assert.NilError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessKey)
		req.Header.Add("Infra-Version", "0.13.0")

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"not authenticated": {
			urlPath: "/api/groups/" + humans.ID.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		"authorized by grant": {
			urlPath: "/api/groups/" + humans.ID.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusNoContent, resp.Body.String())
				actual, err := data.ListGroups(srv.db, data.ByID(humans.ID))
				assert.NilError(t, err)
				assert.Equal(t, len(actual), 0)
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

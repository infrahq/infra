package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestAPI_ListGroups(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	createID := func(t *testing.T, name string) uid.ID {
		t.Helper()
		var buf bytes.Buffer
		body := api.CreateUserRequest{Name: name}
		err := json.NewEncoder(&buf).Encode(body)
		assert.NilError(t, err)

		req, err := http.NewRequest(http.MethodPost, "/v1/identities", &buf)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())
		respObj := &api.CreateUserResponse{}
		err = json.Unmarshal(resp.Body.Bytes(), respObj)
		assert.NilError(t, err)
		return respObj.ID
	}

	createGroup := func(t *testing.T, name string, users ...uid.ID) uid.ID {
		t.Helper()
		group := &models.Group{Name: name}
		for _, user := range users {
			iden := models.Identity{Model: models.Model{ID: user}}
			group.Identities = append(group.Identities, iden)
		}

		err := data.CreateGroup(srv.db, group)
		assert.NilError(t, err)
		return group.ID
	}

	idInGroup := createID(t, "inagroup@example.com")
	idOther := createID(t, "other@example.com")

	humansID := createGroup(t, "humans", idInGroup)
	secondID := createGroup(t, "second", idInGroup)
	createGroup(t, "others", idOther)

	token := &models.AccessKey{
		IssuedFor:  idInGroup,
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
			urlPath: "/api/groups?userID=" + idOther.String(),
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
			urlPath: "/api/groups?userID=" + idInGroup.String(),
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
					humansID.String(),
					time.Now().UTC().Format(time.RFC3339),
				))
				actual := jsonUnmarshal(t, resp.Body.String())
				assert.DeepEqual(t, actual, expected, cmpAPIGrantJSON)
			},
		},
		"version 0.13.0 - list user groups": {
			urlPath: fmt.Sprintf("/api/users/%v/groups", idInGroup),
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
					humansID,
					secondID,
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
					humansID.String(),
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

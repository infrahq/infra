package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestAPI_ListGroups(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	var (
		humans = models.Group{Name: "humans"}
		second = models.Group{Name: "second"}
		others = models.Group{Name: "others"}
	)

	createGroups(t, srv.DB(), &humans, &second, &others)

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

	createIdentities(t, srv.DB(), &idInGroup, &idOther)

	token := &models.AccessKey{
		IssuedFor:  idInGroup.ID,
		ProviderID: data.InfraProvider(srv.DB()).ID,
		ExpiresAt:  time.Now().Add(10 * time.Second),
	}

	accessKey, err := data.CreateAccessKey(srv.DB(), token)
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
		"page 2": {
			urlPath: "/api/groups?page=2&limit=2",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				var actual api.ListResponse[api.Group]
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				assert.Equal(t, len(actual.Items), 1)
				assert.Equal(t, api.PaginationResponse{Page: 2, Limit: 2, TotalCount: 3, TotalPages: 2}, actual.PaginationResponse)
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
	"limit": 100,
	"page": 1,
	"totalPages": 1,
	"totalCount": 1,
	"items": [{
		"id": "%[1]v",
		"name": "humans",
		"created": "%[2]v",
		"updated": "%[2]v",
		"totalUsers": 1
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

func TestAPI_CreateGroup(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	type testCase struct {
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
		body     api.CreateGroupRequest
	}

	meUser := models.Identity{Name: "me@example.com"}
	createIdentities(t, srv.DB(), &meUser)

	token := &models.AccessKey{
		IssuedFor:  meUser.ID,
		ProviderID: data.InfraProvider(srv.DB()).ID,
		ExpiresAt:  time.Now().Add(10 * time.Second),
	}

	accessKey, err := data.CreateAccessKey(srv.DB(), token)
	assert.NilError(t, err)

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.body)
		// nolint:noctx
		req, err := http.NewRequest(http.MethodPost, "/api/groups", body)
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
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized, resp.Body.String())
			},
		},
		"authorized by grant": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())
			},
			body: api.CreateGroupRequest{
				Name: "AwesomeGroup",
			},
		},
		"missing required fields": {
			body: api.CreateGroupRequest{},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "name", Errors: []string{"is required"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
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
	routes := srv.GenerateRoutes()

	humans := models.Group{Name: "humans"}
	createGroups(t, srv.DB(), &humans)

	inGroup := models.Identity{
		Name:   "inagroup@example.com",
		Groups: []models.Group{humans},
	}

	createIdentities(t, srv.DB(), &inGroup)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	token := &models.AccessKey{
		IssuedFor:  inGroup.ID,
		ProviderID: data.InfraProvider(srv.DB()).ID,
		ExpiresAt:  time.Now().Add(10 * time.Second),
	}

	accessKey, err := data.CreateAccessKey(srv.DB(), token)
	assert.NilError(t, err)

	run := func(t *testing.T, tc testCase) {
		// nolint:noctx
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
				_, err := data.GetGroup(srv.DB(), data.GetGroupOptions{ByID: humans.ID})
				assert.ErrorIs(t, err, internal.ErrNotFound)
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestAPI_UpdateUsersInGroup(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	humans := models.Group{Name: "humans"}
	createGroups(t, srv.DB(), &humans)

	var (
		first  = models.Identity{Name: "first@example.com"}
		second = models.Identity{Name: "second@example.com"}
	)

	createIdentities(t, srv.DB(), &first, &second)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
		body     api.UpdateUsersInGroupRequest
	}

	token := &models.AccessKey{
		IssuedFor:  first.ID,
		ProviderID: data.InfraProvider(srv.DB()).ID,
		ExpiresAt:  time.Now().Add(10 * time.Second),
	}

	accessKey, err := data.CreateAccessKey(srv.DB(), token)
	assert.NilError(t, err)

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.body)
		// nolint:noctx
		req, err := http.NewRequest(http.MethodPatch, tc.urlPath, body)
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
			urlPath: fmt.Sprintf("/api/groups/%s/users", humans.ID.String()),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		"add users": {
			urlPath: fmt.Sprintf("/api/groups/%s/users", humans.ID.String()),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				idents, err := data.ListIdentities(srv.DB(), data.ListIdentityOptions{ByGroupID: humans.ID})
				assert.NilError(t, err)
				assert.DeepEqual(t, idents, []models.Identity{first, second}, cmpModelsIdentityShallow)
			},
			body: api.UpdateUsersInGroupRequest{
				UserIDsToAdd: []uid.ID{first.ID, second.ID},
			},
		},
		"remove users": {
			urlPath: fmt.Sprintf("/api/groups/%s/users", humans.ID.String()),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
				err := data.AddUsersToGroup(srv.DB(), humans.ID, []uid.ID{first.ID, second.ID})
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				idents, err := data.ListIdentities(srv.DB(), data.ListIdentityOptions{ByGroupID: humans.ID})
				assert.NilError(t, err)
				assert.Assert(t, len(idents) == 0)
			},
			body: api.UpdateUsersInGroupRequest{
				UserIDsToRemove: []uid.ID{first.ID, second.ID},
			},
		},
		"add unknown user": {
			urlPath: fmt.Sprintf("/api/groups/%s/users", humans.ID.String()),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())
			},
			body: api.UpdateUsersInGroupRequest{
				UserIDsToAdd: []uid.ID{first.ID, 1337, second.ID},
			},
		},
		"remove unknown user": {
			urlPath: fmt.Sprintf("/api/groups/%s/users", humans.ID.String()),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())
			},
			body: api.UpdateUsersInGroupRequest{
				UserIDsToRemove: []uid.ID{1337},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

var cmpModelsIdentityShallow = cmp.Comparer(func(x, y models.Identity) bool {
	return x.Name == y.Name
})

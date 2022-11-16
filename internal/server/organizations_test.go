package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/uid"
)

func createOrgs(t *testing.T, db data.WriteTxn, orgs ...*models.Organization) {
	t.Helper()
	for i := range orgs {
		err := data.CreateOrganization(db, orgs[i])
		assert.NilError(t, err, orgs[i].Name)
	}
}

func TestAPI_GetOrganization(t *testing.T) {
	srv := setupServer(t, withAdminUser, withSupportAdminGrant, withMultiOrgEnabled)
	routes := srv.GenerateRoutes()

	first := models.Organization{Name: "first", Domain: "first.com"}

	createOrgs(t, srv.DB(), &first)
	createID := func(t *testing.T, name string) uid.ID {
		t.Helper()
		var buf bytes.Buffer
		body := api.CreateUserRequest{Name: name}
		err := json.NewEncoder(&buf).Encode(body)
		assert.NilError(t, err)

		// nolint:noctx
		req := httptest.NewRequest(http.MethodPost, "/api/users", &buf)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())
		respObj := &api.CreateUserResponse{}
		err = json.Unmarshal(resp.Body.Bytes(), respObj)
		assert.NilError(t, err)
		return respObj.ID
	}
	idMe := createID(t, "me@example.com")

	token := &models.AccessKey{
		IssuedFor:  idMe,
		ProviderID: data.InfraProvider(srv.DB()).ID,
		ExpiresAt:  time.Now().Add(10 * time.Second),
	}

	accessKeyMe, err := data.CreateAccessKey(srv.DB(), token)
	assert.NilError(t, err)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		req := httptest.NewRequest(http.MethodGet, tc.urlPath, nil)
		req.Header.Add("Infra-Version", "0.15.2")

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"not authenticated": {
			urlPath: "/api/organizations/" + first.ID.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		"not authorized": {
			urlPath: "/api/organizations/" + first.ID.String(),
			setup: func(t *testing.T, req *http.Request) {
				key, _ := createAccessKey(t, srv.DB(), "someonenew@example.com")
				req.Header.Set("Authorization", "Bearer "+key)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden)
			},
		},
		"organization by ID for default org": {
			urlPath: "/api/organizations/" + srv.db.DefaultOrg.ID.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+accessKeyMe)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)
			},
		},
		"organization by ID for a different org": {
			urlPath: "/api/organizations/" + first.ID.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+accessKeyMe)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden)
			},
		},
		"organization by ID for a different org by support admin": {
			urlPath: "/api/organizations/" + first.ID.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)
			},
		},
		"organization by self": {
			urlPath: "/api/organizations/self",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+accessKeyMe)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)
			},
		},
		"JSON response": {
			urlPath: "/api/organizations/" + srv.db.DefaultOrg.ID.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+accessKeyMe)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				expected := jsonUnmarshal(t, fmt.Sprintf(`
					{
						"id": "%[1]v",
						"name": "%[2]v",
						"created": "%[3]v",
						"updated": "%[3]v",
						"domain": "%[4]v",
						"allowedDomains": "%[5]v"
					}`,
					srv.db.DefaultOrg.ID.String(),
					srv.db.DefaultOrg.Name,
					time.Now().UTC().Format(time.RFC3339),
					srv.db.DefaultOrg.Domain,
					srv.db.DefaultOrg.AllowedDomains,
				))
				actual := jsonUnmarshal(t, resp.Body.String())

				cmpAPIOrganizationJSON := gocmp.Options{
					gocmp.FilterPath(pathMapKey(`created`, `updated`), cmpApproximateTime),
					gocmp.FilterPath(pathMapKey(`allowedDomains`), cmpEquateEmptySlice),
				}
				assert.DeepEqual(t, actual, expected, cmpAPIOrganizationJSON)
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestAPI_ListOrganizations(t *testing.T) {
	srv := setupServer(t, withAdminUser, withSupportAdminGrant)
	routes := srv.GenerateRoutes()

	var (
		first  = models.Organization{Name: "first", Domain: "first.example.com"}
		second = models.Organization{Name: "second", Domain: "second.example.com"}
		third  = models.Organization{Name: "third", Domain: "third.example.com"}
	)

	createOrgs(t, srv.DB(), &first, &second, &third)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		req := httptest.NewRequest(http.MethodGet, tc.urlPath, nil)
		req.Header.Add("Infra-Version", "0.14.1")

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"not authenticated": {
			urlPath: "/api/organizations",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		"authorized by grant": {
			urlPath: "/api/organizations",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				var actual api.ListResponse[api.Organization]
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				assert.Equal(t, len(actual.Items), 4)
			},
		},
		"page 2": {
			urlPath: "/api/organizations?page=2&limit=2",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				var actual api.ListResponse[api.Organization]
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				assert.Equal(t, len(actual.Items), 2)
				assert.Equal(t, api.PaginationResponse{Page: 2, Limit: 2, TotalCount: 4, TotalPages: 2}, actual.PaginationResponse)
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestAPI_CreateOrganization(t *testing.T) {
	srv := setupServer(t, withAdminUser, withSupportAdminGrant)
	routes := srv.GenerateRoutes()

	type testCase struct {
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
		body     api.CreateOrganizationRequest
	}

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.body)
		// nolint:noctx
		req := httptest.NewRequest(http.MethodPost, "/api/organizations", body)
		req.Header.Add("Infra-Version", "0.14.1")
		ctx := providers.WithOIDCClient(req.Context(), &fakeOIDCImplementation{})
		*req = *req.WithContext(ctx)

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
			body: api.CreateOrganizationRequest{
				Name:   "AwesomeOrg",
				Domain: "awesome.example.com",
			},
		},
		"missing required fields": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			body: api.CreateOrganizationRequest{},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "domain", Errors: []string{"is required"}},
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

func TestAPI_DeleteOrganization(t *testing.T) {
	srv := setupServer(t, withAdminUser, withSupportAdminGrant)
	routes := srv.GenerateRoutes()

	first := models.Organization{Name: "first", Domain: "first.example.com"}
	createOrgs(t, srv.DB(), &first)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		// nolint:noctx
		req := httptest.NewRequest(http.MethodDelete, tc.urlPath, nil)
		req.Header.Add("Infra-Version", "0.14.1")

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"not authenticated": {
			urlPath: "/api/organizations/" + first.ID.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		"authorized by grant": {
			urlPath: "/api/organizations/" + first.ID.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusNoContent, resp.Body.String())
				_, err := data.GetOrganization(srv.DB(), data.GetOrganizationOptions{ByID: first.ID})
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

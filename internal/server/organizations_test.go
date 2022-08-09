package server

import (
	"context"
	"encoding/json"
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
	"github.com/infrahq/infra/uid"
)

func createOrgs(t *testing.T, db *gorm.DB, orgs ...*models.Organization) {
	t.Helper()
	for i := range orgs {
		o, err := data.GetOrganization(db, data.ByName(orgs[i].Name))
		if err == nil {
			*orgs[i] = *o
			continue
		}
		orgs[i].SetDefaultDomain()
		err = data.CreateOrganization(db, orgs[i])
		assert.NilError(t, err, orgs[i].Name)
	}
}

func TestAPI_ListOrganizations(t *testing.T) {
	srv := setupServer(t, withAdminUser, withSupportAdminGrant)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	var (
		first  = models.Organization{Name: "first"}
		second = models.Organization{Name: "second"}
		third  = models.Organization{Name: "third"}
	)

	createOrgs(t, srv.db, &first, &second, &third)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		// nolint:noctx
		req, err := http.NewRequest(http.MethodGet, tc.urlPath, nil)
		assert.NilError(t, err)
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
				assert.Assert(t, len(actual.Items) >= 3, len(actual.Items))
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
				assert.Assert(t, len(actual.Items) >= 1, len(actual.Items))
				assert.Equal(t, 2, actual.PaginationResponse.Page)
				assert.Equal(t, 2, actual.PaginationResponse.Limit)
				assert.Assert(t, actual.PaginationResponse.TotalCount >= 3, actual.PaginationResponse.TotalCount)
				assert.Assert(t, actual.PaginationResponse.TotalPages >= 2, actual.PaginationResponse.TotalPages)
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
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	type testCase struct {
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
		body     api.CreateOrganizationRequest
	}

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.body)
		// nolint:noctx
		req, err := http.NewRequest(http.MethodPost, "/api/organizations", body)
		assert.NilError(t, err)
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
				Name: "AwesomeOrg",
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

func setCurrentOrg(db *gorm.DB, org *models.Organization) {
	db.Statement.Context = context.WithValue(db.Statement.Context, data.OrgCtxKey{}, org)
}

func TestAPI_DeleteOrganization(t *testing.T) {
	srv := setupServer(t, withAdminUser, withSupportAdminGrant)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	first := &models.Organization{Name: "first"}
	createOrgs(t, srv.db, first)
	setCurrentOrg(srv.db, first)
	p := data.InfraProvider(srv.db)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	user := &models.Identity{
		Model: models.Model{OrganizationID: first.ID},
		Name:  "joe@example.com",
	}
	err := data.CreateIdentity(srv.db, user)
	assert.NilError(t, err)

	key := &models.AccessKey{
		Model:      models.Model{OrganizationID: first.ID},
		Name:       "foo",
		ExpiresAt:  time.Now().Add(10 * time.Minute).UTC(),
		IssuedFor:  user.ID,
		ProviderID: p.ID,
	}
	_, err = data.CreateAccessKey(srv.db, key)
	assert.NilError(t, err)

	err = data.CreateGrant(srv.db, &models.Grant{
		Model:     models.Model{OrganizationID: first.ID},
		Subject:   uid.NewIdentityPolymorphicID(user.ID),
		Resource:  "infra",
		Privilege: "support-admin",
	})
	assert.NilError(t, err)

	run := func(t *testing.T, tc testCase) {
		// nolint:noctx
		req, err := http.NewRequest(http.MethodDelete, tc.urlPath, nil)
		assert.NilError(t, err)
		req.Header.Add("Infra-Version", "0.14.1")
		req.Header.Add("host", first.Domain)

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
				req.Header.Set("Authorization", "Bearer "+key.KeyID+"."+key.Secret)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusNoContent, resp.Body.String())
				actual, err := data.ListOrganizations(srv.db, &models.Pagination{}, data.ByID(first.ID))
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

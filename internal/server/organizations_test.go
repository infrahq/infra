package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestAPI_UpdateOrganization(t *testing.T) {
	srv := setupServer(t, withAdminUser, withSupportAdminGrant, withMultiOrgEnabled)
	routes := srv.GenerateRoutes()

	org := models.Organization{Name: "update-org", Domain: "update.example.com"}

	createOrgs(t, srv.DB(), &org)
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
	idDiffOrg := createID(t, "me@example.com")

	token := &models.AccessKey{
		IssuedFor:  idDiffOrg,
		ProviderID: data.InfraProvider(srv.DB()).ID,
		ExpiresAt:  time.Now().Add(10 * time.Second),
	}

	accessDifferentOrg, err := data.CreateAccessKey(srv.DB(), token)
	assert.NilError(t, err)

	// create a user in the testing org
	user := &models.Identity{
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
		Name:               "joe@example.com",
	}
	assert.NilError(t, data.CreateIdentity(srv.db, user))

	userAccess := &models.AccessKey{
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
		Name:               "org key",
		IssuedFor:          user.ID,
		IssuedForName:      user.Name,
		ProviderID:         data.InfraProvider(srv.db).ID,
		ExpiresAt:          time.Now().Add(10 * time.Minute).UTC().Truncate(time.Second),
	}
	userKey, err := data.CreateAccessKey(srv.db, userAccess)
	assert.NilError(t, err)

	// create an admin user in the testing org
	admin := &models.Identity{
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
		Name:               "alice@example.com",
	}
	assert.NilError(t, data.CreateIdentity(srv.db, admin))

	adminGrant := &models.Grant{
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
		Subject:            uid.NewIdentityPolymorphicID(admin.ID),
		Privilege:          "admin",
		Resource:           "infra",
	}
	assert.NilError(t, data.CreateGrant(srv.db, adminGrant))

	adminAccessKey := &models.AccessKey{
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
		Name:               "org admin key",
		IssuedFor:          admin.ID,
		IssuedForName:      admin.Name,
		ProviderID:         data.InfraProvider(srv.db).ID,
		ExpiresAt:          time.Now().Add(10 * time.Minute).UTC().Truncate(time.Second),
	}
	adminKey, err := data.CreateAccessKey(srv.db, adminAccessKey)
	assert.NilError(t, err)

	type testCase struct {
		urlPath  string
		body     api.UpdateOrganizationRequest
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.body)
		req := httptest.NewRequest(http.MethodPut, tc.urlPath, body)
		req.Header.Add("Infra-Version", apiVersionLatest)

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"not authenticated": {
			urlPath: "/api/organizations/" + org.ID.String(),
			body: api.UpdateOrganizationRequest{
				AllowedDomains: []string{"fail.example.com"},
			},
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		"not authorized": {
			urlPath: "/api/organizations/" + org.ID.String(),
			body: api.UpdateOrganizationRequest{
				AllowedDomains: []string{"fail.example.com"},
			},
			setup: func(t *testing.T, req *http.Request) {
				key, _ := createAccessKey(t, srv.DB(), "someonenew@example.com")
				req.Header.Set("Authorization", "Bearer "+key)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden)
			},
		},
		"fails to update organization for a different org": {
			urlPath: "/api/organizations/" + org.ID.String(),
			body: api.UpdateOrganizationRequest{
				AllowedDomains: []string{"fail.example.com"},
			},
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+accessDifferentOrg)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden)
			},
		},
		"fails to update allowed domains with no organization admin grant": {
			urlPath: "/api/organizations/" + org.ID.String(),
			body: api.UpdateOrganizationRequest{
				AllowedDomains: []string{"hello.example.com", "hi.example.com"},
			},
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+userKey)
				req.Host = "update.example.com"
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden)
			},
		},
		"fails to update allowed domains when a domain is invalid": {
			urlPath: "/api/organizations/" + org.ID.String(),
			body: api.UpdateOrganizationRequest{
				AllowedDomains: []string{"hello.example.com", "@example.com"},
			},
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminKey)
				req.Host = "update.example.com"
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest)
				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				assert.Assert(t, strings.Contains(respBody.Message, "first character '@' is not allowed"), respBody.Message)
			},
		},
		"can update allowed domains when organization admin": {
			urlPath: "/api/organizations/" + org.ID.String(),
			body: api.UpdateOrganizationRequest{
				AllowedDomains: []string{"hello.example.com", "hi.example.com"},
			},
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminKey)
				req.Host = "update.example.com"
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				actual := &api.Organization{}
				err := json.Unmarshal(resp.Body.Bytes(), actual)
				assert.NilError(t, err)
				expected := &api.Organization{
					ID:             actual.ID, // does not matter
					Name:           org.Name,
					Created:        actual.Created, // does not matter
					Updated:        actual.Updated, // does not matter
					Domain:         org.Domain,
					AllowedDomains: []string{"hello.example.com", "hi.example.com"},
				}
				assert.DeepEqual(t, actual, expected)
			},
		},
		"duplicate allowed domains are ignored": {
			urlPath: "/api/organizations/" + org.ID.String(),
			body: api.UpdateOrganizationRequest{
				AllowedDomains: []string{"hello.example.com", "hello.example.com"},
			},
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminKey)
				req.Host = "update.example.com"
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				actual := &api.Organization{}
				err := json.Unmarshal(resp.Body.Bytes(), actual)
				assert.NilError(t, err)
				expected := &api.Organization{
					ID:             actual.ID, // does not matter
					Name:           org.Name,
					Created:        actual.Created, // does not matter
					Updated:        actual.Updated, // does not matter
					Domain:         org.Domain,
					AllowedDomains: []string{"hello.example.com"},
				}
				assert.DeepEqual(t, actual, expected)
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

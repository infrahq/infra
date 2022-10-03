package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestAPI_ListProviderUsers(t *testing.T) {
	type testCase struct {
		name   string
		setup  func(t *testing.T) (bearer, params string, routes Routes, expected api.ListProviderUsersResponse)
		verify func(t *testing.T, expected api.ListProviderUsersResponse, resp *httptest.ResponseRecorder)
	}

	testCases := []testCase{
		{
			name: "valid, no parameters",
			setup: func(t *testing.T) (bearer, params string, routes Routes, expected api.ListProviderUsersResponse) {
				s := setupServer(t, withAdminUser)
				bearer, users, routes := createTestSCIMProvider(t, s)
				expectedUsers := []api.SCIMUser{}
				for _, u := range users {
					expectedUsers = append(expectedUsers, *u.ToAPI())
				}

				expected = api.ListProviderUsersResponse{
					Schemas:      []string{api.ListResponseSchema},
					Resources:    expectedUsers,
					TotalResults: 1,
					StartIndex:   0,
					ItemsPerPage: 100,
				}

				return bearer, "", routes, expected
			},
			verify: func(t *testing.T, expected api.ListProviderUsersResponse, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

				var response api.ListProviderUsersResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NilError(t, err)
				assert.DeepEqual(t, expected, response)
			},
		},
		{
			name: "scim role is required",
			setup: func(t *testing.T) (bearer, params string, routes Routes, expected api.ListProviderUsersResponse) {
				s := setupServer(t, withAdminUser)
				// setup the users and permissions as needed
				bearer, users, routes := createTestSCIMProvider(t, s)

				// then remove the grant for the SCIM provider
				opts := data.DeleteGrantsOptions{
					BySubject: uid.NewIdentityPolymorphicID(users[0].IdentityID),
				}
				err := data.DeleteGrants(s.DB(), opts)
				assert.NilError(t, err)

				return bearer, "", routes, api.ListProviderUsersResponse{}
			},
			verify: func(t *testing.T, expected api.ListProviderUsersResponse, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusForbidden, resp.Code)
			},
		},
		{
			name: "access key issued for non-existent provider results in an empty response",
			setup: func(t *testing.T) (bearer, params string, routes Routes, expected api.ListProviderUsersResponse) {
				s := setupServer(t, withAdminUser)
				// setup the users and permissions as needed
				_, users, routes := createTestSCIMProvider(t, s, "another")
				key := &models.AccessKey{
					OrganizationMember: s.db.DefaultOrgSettings.OrganizationMember,
					IssuedFor:          users[0].IdentityID,
					IssuedForName:      "another",
					Name:               fmt.Sprintf("%s-123", "another"),
					ProviderID:         data.InfraProvider(s.DB()).ID,
				}
				bearer, err := data.CreateAccessKey(s.DB(), key)
				assert.NilError(t, err)

				err = data.CreateGrant(s.DB(),
					&models.Grant{
						Subject:   uid.NewIdentityPolymorphicID(users[0].IdentityID),
						Privilege: models.InfraSCIMRole,
						Resource:  "infra",
					},
				)
				assert.NilError(t, err)

				return bearer, "", routes, api.ListProviderUsersResponse{Schemas: []string{api.ListResponseSchema}, ItemsPerPage: 100}
			},
			verify: func(t *testing.T, expected api.ListProviderUsersResponse, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				var response api.ListProviderUsersResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NilError(t, err)
				assert.DeepEqual(t, expected, response)
			},
		},
		{
			name: "2 users, 1 count",
			setup: func(t *testing.T) (bearer, params string, routes Routes, expected api.ListProviderUsersResponse) {
				s := setupServer(t, withAdminUser)
				bearer, users, routes := createTestSCIMProvider(t, s, "a")

				expected = api.ListProviderUsersResponse{
					Schemas:      []string{api.ListResponseSchema},
					Resources:    []api.SCIMUser{*users[0].ToAPI()}, // this user has the "a" email, so they return first alphabetically
					TotalResults: 2,
					StartIndex:   0,
					ItemsPerPage: 1,
				}

				return bearer, "?count=1", routes, expected
			},
			verify: func(t *testing.T, expected api.ListProviderUsersResponse, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

				var response api.ListProviderUsersResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NilError(t, err)
				assert.DeepEqual(t, expected, response)
			},
		},
		{
			name: "2 users, 3 count",
			setup: func(t *testing.T) (bearer, params string, routes Routes, expected api.ListProviderUsersResponse) {
				s := setupServer(t, withAdminUser)
				bearer, users, routes := createTestSCIMProvider(t, s, "a")

				expectedUsers := []api.SCIMUser{}
				for _, u := range users {
					expectedUsers = append(expectedUsers, *u.ToAPI())
				}

				expected = api.ListProviderUsersResponse{
					Schemas:      []string{api.ListResponseSchema},
					Resources:    expectedUsers,
					TotalResults: 2,
					StartIndex:   0,
					ItemsPerPage: 3,
				}

				return bearer, "?count=3", routes, expected
			},
			verify: func(t *testing.T, expected api.ListProviderUsersResponse, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

				var response api.ListProviderUsersResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NilError(t, err)
				assert.DeepEqual(t, expected, response)
			},
		},
	}

	for _, tc := range testCases {
		bearer, params, routes, exp := tc.setup(t)
		// nolint:noctx
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/scim/v2/Users%s", params), nil)
		assert.NilError(t, err)

		req.Header.Add("Authorization", "Bearer "+bearer)
		req.Header.Add("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.verify(t, exp, resp)
	}
}

func createTestSCIMProvider(t *testing.T, s *Server, extraUserNames ...string) (bearer string, users []models.ProviderUser, routes Routes) {
	testProvider := &models.Provider{
		Model: models.Model{
			ID: 1234,
		},
		Name:    "mockta",
		Kind:    models.ProviderKindOkta,
		AuthURL: "https://example.com/v1/auth",
		Scopes:  []string{"openid", "email"},
	}

	err := data.CreateProvider(s.DB(), testProvider)
	assert.NilError(t, err)

	testProviderUser := createTestSCIMUserIdentity(t, s.DB(), testProvider, 1234, "test@example.com") // intentionally the same ID as the test provider as a workaround until provider access keys are supported
	users = append(users, *testProviderUser)
	for i, name := range extraUserNames {
		testProviderUser := createTestSCIMUserIdentity(t, s.DB(), testProvider, uid.ID(i), name) // intentionally the same ID as the test provider as a workaround until provider access keys are supported
		users = append(users, *testProviderUser)
	}

	// sort users by email, this is to match expected result
	sort.Slice(users, func(i, j int) bool {
		return users[i].Email < users[j].Email
	})

	key := &models.AccessKey{
		OrganizationMember: s.db.DefaultOrgSettings.OrganizationMember,
		IssuedFor:          testProvider.ID,
		IssuedForName:      testProvider.Name,
		Name:               fmt.Sprintf("%s-123", testProvider.Name),
		ProviderID:         data.InfraProvider(s.DB()).ID,
	}
	bearer, err = data.CreateAccessKey(s.DB(), key)
	assert.NilError(t, err)

	err = data.CreateGrant(s.DB(),
		&models.Grant{
			Subject:   uid.NewIdentityPolymorphicID(testProviderUser.IdentityID),
			Privilege: models.InfraSCIMRole,
			Resource:  "infra",
		},
	)
	assert.NilError(t, err)

	return bearer, users, s.GenerateRoutes()
}

func createTestSCIMUserIdentity(t *testing.T, db data.GormTxn, provider *models.Provider, id uid.ID, name string) *models.ProviderUser {
	testIdentity := &models.Identity{
		Model: models.Model{
			ID: id,
		},
		Name: name,
	}
	err := data.CreateIdentity(db, testIdentity)
	assert.NilError(t, err)

	testProviderUser, err := data.CreateProviderUser(db, provider, testIdentity)
	assert.NilError(t, err)

	return testProviderUser
}

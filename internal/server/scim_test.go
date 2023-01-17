package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestAPI_GetProviderUser(t *testing.T) {
	type testCase struct {
		name   string
		setup  func(t *testing.T) (bearer string, routes Routes, user api.SCIMUser)
		verify func(t *testing.T, expected api.SCIMUser, resp *httptest.ResponseRecorder)
	}

	testCases := []testCase{
		{
			name: "valid ID for expected provider",
			setup: func(t *testing.T) (bearer string, routes Routes, user api.SCIMUser) {
				s := setupServer(t, withAdminUser)
				bearer, users, routes := createTestSCIMProvider(t, s)
				user = *(users[0]).ToAPI()

				return bearer, routes, user
			},
			verify: func(t *testing.T, expected api.SCIMUser, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

				var response api.SCIMUser
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NilError(t, err)
				assert.DeepEqual(t, expected, response)
			},
		},
		{
			name: "invalid ID for user fails",
			setup: func(t *testing.T) (bearer string, routes Routes, user api.SCIMUser) {
				s := setupServer(t, withAdminUser)
				bearer, users, routes := createTestSCIMProvider(t, s)
				user = *(users[0]).ToAPI()
				// set a bad ID on the user
				user.ID = uid.New().String()

				return bearer, routes, user
			},
			verify: func(t *testing.T, expected api.SCIMUser, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
			},
		},
		{
			name: "valid ID in different provider fails",
			setup: func(t *testing.T) (bearer string, routes Routes, user api.SCIMUser) {
				s := setupServer(t, withAdminUser)
				bearer, _, routes = createTestSCIMProvider(t, s)
				userInDifferentProvider := createTestSCIMUserIdentity(t, s.DB(), data.InfraProvider(s.DB()), 123, "infra-user")

				return bearer, routes, *userInDifferentProvider.ToAPI()
			},
			verify: func(t *testing.T, expected api.SCIMUser, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
			},
		},
		{
			name: "access key not issued for provider fails",
			setup: func(t *testing.T) (bearer string, routes Routes, user api.SCIMUser) {
				s := setupServer(t, withAdminUser)
				_, users, routes := createTestSCIMProvider(t, s, "some-user-name")
				user = *(users[1]).ToAPI()

				key := &models.AccessKey{
					OrganizationMember: s.db.DefaultOrgSettings.OrganizationMember,
					IssuedForUser:      users[1].IdentityID,
					IssuedForUserName:  user.UserName,
					Name:               fmt.Sprintf("%s-123", user.UserName),
					ProviderID:         1234,
				}
				bearer, err := data.CreateAccessKey(s.DB(), key)
				assert.NilError(t, err)

				return bearer, routes, user
			},
			verify: func(t *testing.T, expected api.SCIMUser, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
			},
		},
	}

	for _, tc := range testCases {
		bearer, routes, user := tc.setup(t)
		// nolint:noctx
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/scim/v2/Users/%s", user.ID), nil)
		req.Header.Add("Authorization", "Bearer "+bearer)
		req.Header.Add("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.verify(t, user, resp)
	}
}

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
					ItemsPerPage: 1,
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
			name: "access key issued for non-existent provider results in an empty response",
			setup: func(t *testing.T) (bearer, params string, routes Routes, expected api.ListProviderUsersResponse) {
				s := setupServer(t, withAdminUser)
				// setup the users and permissions as needed
				_, users, routes := createTestSCIMProvider(t, s, "another")
				key := &models.AccessKey{
					OrganizationMember: s.db.DefaultOrgSettings.OrganizationMember,
					IssuedForUser:      users[0].IdentityID,
					IssuedForUserName:  "another",
					Name:               fmt.Sprintf("%s-123", "another"),
					ProviderID:         data.InfraProvider(s.DB()).ID,
				}
				bearer, err := data.CreateAccessKey(s.DB(), key)
				assert.NilError(t, err)

				return bearer, "", routes, api.ListProviderUsersResponse{}
			},
			verify: func(t *testing.T, expected api.ListProviderUsersResponse, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, resp.Code)
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
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/scim/v2/Users%s", params), nil)
		req.Header.Add("Authorization", "Bearer "+bearer)
		req.Header.Add("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.verify(t, exp, resp)
	}
}

var cmpSCIMUserResponse = cmp.Options{
	cmp.FilterPath(opt.PathField(api.SCIMUser{}, "ID"), cmpAnyString),
}

func TestAPI_CreateProviderUser(t *testing.T) {
	type testCase struct {
		name   string
		setup  func(t *testing.T) (bearer string, routes Routes, reqBody api.SCIMUserCreateRequest)
		verify func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	testCases := []testCase{
		{
			name: "valid new user",
			setup: func(t *testing.T) (bearer string, routes Routes, reqBody api.SCIMUserCreateRequest) {
				s := setupServer(t, withAdminUser)
				bearer, _, routes = createTestSCIMProvider(t, s)

				reqBody = api.SCIMUserCreateRequest{
					Schemas:  []string{api.UserSchema},
					UserName: "david@example.com",
					Name: api.SCIMUserName{
						GivenName:  "David",
						FamilyName: "Martinez",
					},
					Emails: []api.SCIMUserEmail{
						{
							Primary: true,
							Value:   "david@example.com",
						},
					},
					Active: true,
				}
				return bearer, routes, reqBody
			},
			verify: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

				var response api.SCIMUser
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NilError(t, err)

				expected := api.SCIMUser{
					Schemas:  []string{api.UserSchema},
					ID:       "<any-string>",
					UserName: "david@example.com",
					Name: api.SCIMUserName{
						GivenName:  "David",
						FamilyName: "Martinez",
					},
					Emails: []api.SCIMUserEmail{
						{
							Primary: true,
							Value:   "david@example.com",
						},
					},
					Active: true,
					Meta: api.SCIMMetadata{
						ResourceType: "User",
					},
				}
				assert.DeepEqual(t, expected, response, cmpSCIMUserResponse)
			},
		},
		{
			name: "valid user that exists in another identity provider already",
			setup: func(t *testing.T) (bearer string, routes Routes, reqBody api.SCIMUserCreateRequest) {
				s := setupServer(t, withAdminUser)
				bearer, _, routes = createTestSCIMProvider(t, s)
				createTestSCIMUserIdentity(t, s.DB(), data.InfraProvider(s.DB()), 123, "david@example.com")

				reqBody = api.SCIMUserCreateRequest{
					Schemas:  []string{api.UserSchema},
					UserName: "david@example.com",
					Name: api.SCIMUserName{
						GivenName:  "David",
						FamilyName: "Martinez",
					},
					Emails: []api.SCIMUserEmail{
						{
							Primary: true,
							Value:   "david@example.com",
						},
					},
					Active: true,
				}
				return bearer, routes, reqBody
			},
			verify: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

				var response api.SCIMUser
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NilError(t, err)

				expected := api.SCIMUser{
					Schemas:  []string{api.UserSchema},
					ID:       "<any-string>",
					UserName: "david@example.com",
					Name: api.SCIMUserName{
						GivenName:  "David",
						FamilyName: "Martinez",
					},
					Emails: []api.SCIMUserEmail{
						{
							Primary: true,
							Value:   "david@example.com",
						},
					},
					Active: true,
					Meta: api.SCIMMetadata{
						ResourceType: "User",
					},
				}
				assert.DeepEqual(t, expected, response, cmpSCIMUserResponse)
			},
		},
		{
			name: "user already provisioned",
			setup: func(t *testing.T) (bearer string, routes Routes, reqBody api.SCIMUserCreateRequest) {
				s := setupServer(t, withAdminUser)
				bearer, _, routes = createTestSCIMProvider(t, s, "david@example.com")

				reqBody = api.SCIMUserCreateRequest{
					Schemas:  []string{api.UserSchema},
					UserName: "david@example.com",
					Name: api.SCIMUserName{
						GivenName:  "David",
						FamilyName: "Martinez",
					},
					Emails: []api.SCIMUserEmail{
						{
							Primary: true,
							Value:   "david@example.com",
						},
					},
					Active: true,
				}
				return bearer, routes, reqBody
			},
			verify: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
			},
		},
		{
			name: "invalid user, schema required",
			setup: func(t *testing.T) (bearer string, routes Routes, reqBody api.SCIMUserCreateRequest) {
				s := setupServer(t, withAdminUser)
				bearer, _, routes = createTestSCIMProvider(t, s)

				reqBody = api.SCIMUserCreateRequest{
					UserName: "david@example.com",
					Name: api.SCIMUserName{
						GivenName:  "David",
						FamilyName: "Martinez",
					},
					Emails: []api.SCIMUserEmail{
						{
							Primary: true,
							Value:   "david@example.com",
						},
					},
					Active: true,
				}
				return bearer, routes, reqBody
			},
			verify: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, resp.Code)
				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "schemas", Errors: []string{"is required"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "access key not issued for provider fails",
			setup: func(t *testing.T) (bearer string, routes Routes, reqBody api.SCIMUserCreateRequest) {
				s := setupServer(t, withAdminUser)
				_, users, routes := createTestSCIMProvider(t, s, "some-user-name")
				user := *(users[1]).ToAPI()
				key := &models.AccessKey{
					OrganizationMember: s.db.DefaultOrgSettings.OrganizationMember,
					IssuedForUser:      users[1].IdentityID,
					IssuedForUserName:  user.UserName,
					Name:               fmt.Sprintf("%s-123", user.UserName),
					ProviderID:         1234,
				}
				bearer, err := data.CreateAccessKey(s.DB(), key)
				assert.NilError(t, err)

				reqBody = api.SCIMUserCreateRequest{
					Schemas:  []string{api.UserSchema},
					UserName: "david@example.com",
					Name: api.SCIMUserName{
						GivenName:  "David",
						FamilyName: "Martinez",
					},
					Emails: []api.SCIMUserEmail{
						{
							Primary: true,
							Value:   "david@example.com",
						},
					},
					Active: true,
				}
				return bearer, routes, reqBody
			},
			verify: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
			},
		},
	}

	for _, tc := range testCases {
		bearer, routes, reqBody := tc.setup(t)
		// nolint:noctx
		req := httptest.NewRequest(http.MethodPost, "/api/scim/v2/Users", jsonBody(t, reqBody))
		req.Header.Add("Authorization", "Bearer "+bearer)
		req.Header.Add("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.verify(t, resp)
	}
}

func TestAPI_UpdateProviderUser(t *testing.T) {
	type testCase struct {
		name   string
		setup  func(t *testing.T) (bearer string, routes Routes, id uid.ID, reqBody api.SCIMUserUpdateRequest)
		verify func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	testCases := []testCase{
		{
			name: "valid user update, same email",
			setup: func(t *testing.T) (bearer string, routes Routes, id uid.ID, reqBody api.SCIMUserUpdateRequest) {
				s := setupServer(t, withAdminUser)
				bearer, users, routes := createTestSCIMProvider(t, s, "david@example.com")
				user := users[0]

				reqBody = api.SCIMUserUpdateRequest{
					Schemas:  []string{api.UserSchema},
					UserName: "david@example.com",
					Name: api.SCIMUserName{
						GivenName:  "Dave",
						FamilyName: "Martinez",
					},
					Emails: []api.SCIMUserEmail{
						{
							Primary: true,
							Value:   "david@example.com",
						},
					},
					Active: true,
				}
				return bearer, routes, user.IdentityID, reqBody
			},
			verify: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

				var response api.SCIMUser
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NilError(t, err)

				expected := api.SCIMUser{
					Schemas:  []string{api.UserSchema},
					ID:       "<any-string>",
					UserName: "david@example.com",
					Name: api.SCIMUserName{
						GivenName:  "Dave",
						FamilyName: "Martinez",
					},
					Emails: []api.SCIMUserEmail{
						{
							Primary: true,
							Value:   "david@example.com",
						},
					},
					Active: true,
					Meta: api.SCIMMetadata{
						ResourceType: "User",
					},
				}
				assert.DeepEqual(t, expected, response, cmpSCIMUserResponse)
			},
		},
		{
			name: "valid user update, new values",
			setup: func(t *testing.T) (bearer string, routes Routes, id uid.ID, reqBody api.SCIMUserUpdateRequest) {
				s := setupServer(t, withAdminUser)
				bearer, users, routes := createTestSCIMProvider(t, s, "david@example.com")
				user := users[0]

				reqBody = api.SCIMUserUpdateRequest{
					Schemas:  []string{api.UserSchema},
					UserName: "davidm@example.com",
					Name: api.SCIMUserName{
						GivenName:  "Davie",
						FamilyName: "M",
					},
					Emails: []api.SCIMUserEmail{
						{
							Primary: true,
							Value:   "davidm@example.com",
						},
					},
					Active: false,
				}
				return bearer, routes, user.IdentityID, reqBody
			},
			verify: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

				var response api.SCIMUser
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NilError(t, err)

				expected := api.SCIMUser{
					Schemas:  []string{api.UserSchema},
					ID:       "<any-string>",
					UserName: "davidm@example.com",
					Name: api.SCIMUserName{
						GivenName:  "Davie",
						FamilyName: "M",
					},
					Emails: []api.SCIMUserEmail{
						{
							Primary: true,
							Value:   "davidm@example.com",
						},
					},
					Active: false,
					Meta: api.SCIMMetadata{
						ResourceType: "User",
					},
				}
				assert.DeepEqual(t, expected, response, cmpSCIMUserResponse)
			},
		},
		{
			name: "invalid user update, no schema",
			setup: func(t *testing.T) (bearer string, routes Routes, id uid.ID, reqBody api.SCIMUserUpdateRequest) {
				s := setupServer(t, withAdminUser)
				bearer, users, routes := createTestSCIMProvider(t, s, "david@example.com")
				user := users[0]

				reqBody = api.SCIMUserUpdateRequest{
					UserName: "davidm@example.com",
					Name: api.SCIMUserName{
						GivenName:  "Davie",
						FamilyName: "M",
					},
					Emails: []api.SCIMUserEmail{
						{
							Primary: true,
							Value:   "davidm@example.com",
						},
					},
					Active: false,
				}
				return bearer, routes, user.IdentityID, reqBody
			},
			verify: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, resp.Code)
				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "schemas", Errors: []string{"is required"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "invalid request, access key not issued for provider",
			setup: func(t *testing.T) (bearer string, routes Routes, id uid.ID, reqBody api.SCIMUserUpdateRequest) {
				s := setupServer(t, withAdminUser)
				_, users, routes := createTestSCIMProvider(t, s, "david@example.com")
				user := users[0]

				key := &models.AccessKey{
					OrganizationMember: s.db.DefaultOrgSettings.OrganizationMember,
					IssuedForUser:      user.IdentityID,
					IssuedForUserName:  user.Email,
					Name:               fmt.Sprintf("%s-123", user.Email),
					ProviderID:         1234,
				}
				bearer, err := data.CreateAccessKey(s.DB(), key)
				assert.NilError(t, err)

				reqBody = api.SCIMUserUpdateRequest{
					Schemas:  []string{api.UserSchema},
					UserName: "davidm@example.com",
					Name: api.SCIMUserName{
						GivenName:  "Davie",
						FamilyName: "M",
					},
					Emails: []api.SCIMUserEmail{
						{
							Primary: true,
							Value:   "davidm@example.com",
						},
					},
					Active: false,
				}
				return bearer, routes, user.IdentityID, reqBody
			},
			verify: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
			},
		},
	}

	for _, tc := range testCases {
		bearer, routes, id, reqBody := tc.setup(t)
		// nolint:noctx
		req := httptest.NewRequest(http.MethodPut, "/api/scim/v2/Users/"+id.String(), jsonBody(t, reqBody))
		req.Header.Add("Authorization", "Bearer "+bearer)
		req.Header.Add("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.verify(t, resp)
	}
}

func TestAPI_PatchProviderUser(t *testing.T) {
	type testCase struct {
		name   string
		setup  func(t *testing.T) (bearer string, routes Routes, id uid.ID, reqBody api.SCIMUserPatchRequest)
		verify func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	testCases := []testCase{
		{
			name: "valid user patch",
			setup: func(t *testing.T) (bearer string, routes Routes, id uid.ID, reqBody api.SCIMUserPatchRequest) {
				s := setupServer(t, withAdminUser)
				bearer, users, routes := createTestSCIMProvider(t, s, "david@example.com")
				user := users[0]

				reqBody = api.SCIMUserPatchRequest{
					Schemas: []string{api.PatchOperationSchema},
					Operations: []api.SCIMPatchOperation{
						{
							Op: "replace",
							Value: api.SCIMPatchStatus{
								Active: false,
							},
						},
					},
				}
				return bearer, routes, user.IdentityID, reqBody
			},
			verify: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

				var response api.SCIMUser
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NilError(t, err)

				expected := api.SCIMUser{
					Schemas:  []string{api.UserSchema},
					ID:       "<any-string>",
					UserName: "david@example.com",
					Name: api.SCIMUserName{
						GivenName:  "",
						FamilyName: "",
					},
					Emails: []api.SCIMUserEmail{
						{
							Primary: true,
							Value:   "david@example.com",
						},
					},
					Active: false,
					Meta: api.SCIMMetadata{
						ResourceType: "User",
					},
				}
				assert.DeepEqual(t, expected, response, cmpSCIMUserResponse)
			},
		},
		{
			name: "invalid request, key not issued for provider",
			setup: func(t *testing.T) (bearer string, routes Routes, id uid.ID, reqBody api.SCIMUserPatchRequest) {
				s := setupServer(t, withAdminUser)
				_, users, routes := createTestSCIMProvider(t, s, "david@example.com")
				user := users[0]
				key := &models.AccessKey{
					OrganizationMember: s.db.DefaultOrgSettings.OrganizationMember,
					IssuedForUser:      user.IdentityID,
					IssuedForUserName:  user.Email,
					Name:               fmt.Sprintf("%s-123", user.Email),
					ProviderID:         1234,
				}
				bearer, err := data.CreateAccessKey(s.DB(), key)
				assert.NilError(t, err)

				reqBody = api.SCIMUserPatchRequest{
					Schemas: []string{api.PatchOperationSchema},
					Operations: []api.SCIMPatchOperation{
						{
							Op: "replace",
							Value: api.SCIMPatchStatus{
								Active: false,
							},
						},
					},
				}
				return bearer, routes, user.IdentityID, reqBody
			},
			verify: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
			},
		},
	}

	for _, tc := range testCases {
		bearer, routes, id, reqBody := tc.setup(t)
		// nolint:noctx
		req := httptest.NewRequest(http.MethodPatch, "/api/scim/v2/Users/"+id.String(), jsonBody(t, reqBody))
		req.Header.Add("Authorization", "Bearer "+bearer)
		req.Header.Add("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.verify(t, resp)
	}
}

func TestAPI_DeleteProviderUser(t *testing.T) {
	type testCase struct {
		name   string
		setup  func(t *testing.T) (bearer string, routes Routes, id uid.ID)
		verify func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	testCases := []testCase{
		{
			name: "valid user delete",
			setup: func(t *testing.T) (bearer string, routes Routes, id uid.ID) {
				s := setupServer(t, withAdminUser)
				bearer, users, routes := createTestSCIMProvider(t, s, "david@example.com")
				user := users[0]

				return bearer, routes, user.IdentityID
			},
			verify: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, resp.Code)
			},
		},
		{
			name: "invalid request, key not issued for provider",
			setup: func(t *testing.T) (bearer string, routes Routes, id uid.ID) {
				s := setupServer(t, withAdminUser)
				_, users, routes := createTestSCIMProvider(t, s, "david@example.com")
				user := users[0]
				key := &models.AccessKey{
					OrganizationMember: s.db.DefaultOrgSettings.OrganizationMember,
					IssuedForUser:      user.IdentityID,
					IssuedForUserName:  user.Email,
					Name:               fmt.Sprintf("%s-123", user.Email),
					ProviderID:         1234,
				}
				bearer, err := data.CreateAccessKey(s.DB(), key)
				assert.NilError(t, err)

				return bearer, routes, user.IdentityID
			},
			verify: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
			},
		},
	}

	for _, tc := range testCases {
		bearer, routes, id := tc.setup(t)
		// nolint:noctx
		req := httptest.NewRequest(http.MethodDelete, "/api/scim/v2/Users/"+id.String(), nil)
		req.Header.Add("Authorization", "Bearer "+bearer)
		req.Header.Add("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.verify(t, resp)
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

	testProviderUser := createTestSCIMUserIdentity(t, s.DB(), testProvider, 1234, testProvider.Name+"-scim") // intentionally the same ID as the test provider as a workaround until provider access keys are supported
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
		IssuedForUser:      testProvider.ID,
		IssuedForUserName:  testProvider.Name,
		Name:               fmt.Sprintf("%s-123", testProvider.Name),
		ProviderID:         testProvider.ID,
	}
	bearer, err = data.CreateAccessKey(s.DB(), key)
	assert.NilError(t, err)

	return bearer, users, s.GenerateRoutes()
}

func createTestSCIMUserIdentity(t *testing.T, db data.WriteTxn, provider *models.Provider, id uid.ID, name string) *models.ProviderUser {
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

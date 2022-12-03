package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestAPI_Signup(t *testing.T) {
	type testCase struct {
		name     string
		setup    func(t *testing.T) api.SignupRequest
		expected func(t *testing.T, response *httptest.ResponseRecorder)
	}

	srv := setupServer(t, withAdminUser)
	srv.options.EnableSignup = true
	srv.options.BaseDomain = "exampledomain.com"
	routes := srv.GenerateRoutes()

	run := func(t *testing.T, tc testCase) {
		body := tc.setup(t)

		// nolint:noctx
		req := httptest.NewRequest(http.MethodPost, "/api/signup", jsonBody(t, body))
		req.Header.Set("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := []testCase{
		{
			name: "invalid subdomain",
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{
					Name:     "admin@example.com",
					Password: "thispasswordisgreat",
					Org: api.SignupOrg{
						Name:      "My org is awesome",
						Subdomain: "h@",
					},
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "org.subDomain", Errors: []string{
						"must be at least 4 characters",
						"character '@' at position 1 is not allowed",
					}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "invalid password",
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{
					Name:     "admin@example.com",
					Password: "short",
					Org: api.SignupOrg{
						Name:      "My org is awesome",
						Subdomain: "helloo",
					},
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "password", Errors: []string{
						"must be at least 8 characters",
					}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)

				// the org should have been rolled back
				_, err = data.GetOrganization(srv.DB(), data.GetOrganizationOptions{
					ByDomain: "hello.exampledomain.com",
				})
				assert.ErrorIs(t, err, internal.ErrNotFound)
			},
		},
		{
			name: "missing name, password, and org",
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "name", Errors: []string{"is required"}},
					{FieldName: "org.name", Errors: []string{"is required"}},
					{FieldName: "org.subDomain", Errors: []string{"is required"}},
					{FieldName: "password", Errors: []string{"is required"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "signup disabled",
			setup: func(t *testing.T) api.SignupRequest {
				srv.options.EnableSignup = false
				t.Cleanup(func() {
					srv.options.EnableSignup = true
				})

				return api.SignupRequest{
					Name:     "admin@example.com",
					Password: "password",
					Org:      api.SignupOrg{Name: "acme", Subdomain: "acme-co"},
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				assert.Equal(t, respBody.Message, "bad request: signup is disabled")
			},
		},
		{
			name: "duplicate organization name",
			setup: func(t *testing.T) api.SignupRequest {
				err := data.CreateOrganization(srv.DB(), &models.Organization{
					Name:   "Something",
					Domain: "taken.exampledomain.com",
				})
				assert.NilError(t, err)

				return api.SignupRequest{
					Name:     "admin@example.com",
					Password: "password",
					Org:      api.SignupOrg{Name: "Example", Subdomain: "taken"},
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusConflict, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{
						FieldName: "org.subDomain",
						Errors:    []string{"an organization with that domain already exists"},
					},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "successful signup",
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{
					Name:     "admin@example.com",
					Password: "password",
					Org:      api.SignupOrg{Name: "acme", Subdomain: "acme-co"},
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				// the response is success
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

				respBody := &api.SignupResponse{}
				err := json.NewDecoder(resp.Body).Decode(respBody)
				assert.NilError(t, err)

				assert.Equal(t, respBody.User.Name, "admin@example.com")
				assert.Equal(t, respBody.Organization.Name, "acme")
				assert.DeepEqual(t, respBody.Organization.AllowedDomains, []string{"example.com"})
				userID := respBody.User.ID
				orgID := respBody.Organization.ID

				// the organization exists
				org, err := data.GetOrganization(srv.DB(), data.GetOrganizationOptions{
					ByDomain: "acme-co.exampledomain.com",
				})
				assert.NilError(t, err)
				assert.Equal(t, org.ID, respBody.Organization.ID)

				// the admin user has a valid access key
				httpResp := resp.Result()
				cookies := httpResp.Cookies()
				assert.Equal(t, len(cookies), 1)

				key := cookies[0].Value
				// nolint:noctx
				req := httptest.NewRequest(http.MethodGet, "/api/users/self", nil)
				req.Header.Set("Infra-Version", apiVersionLatest)
				req.Header.Set("Authorization", "Bearer "+key)

				resp = httptest.NewRecorder()
				routes.ServeHTTP(resp, req)

				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				userResp := &api.User{}
				err = json.NewDecoder(resp.Body).Decode(userResp)
				assert.NilError(t, err)
				assert.Equal(t, userResp.ID, respBody.User.ID)

				// check the user is an admin
				tx := txnForTestCase(t, srv.db, orgID)
				_, err = data.GetGrant(tx, data.GetGrantOptions{
					BySubject:   uid.NewIdentityPolymorphicID(userID),
					ByResource:  access.ResourceInfraAPI,
					ByPrivilege: api.InfraAdminRole,
				})
				assert.NilError(t, err)

				// check their access token has the expected scope
				k, err := data.GetAccessKeyByKeyID(srv.DB(), strings.Split(key, ".")[0])
				assert.NilError(t, err)

				assert.DeepEqual(t, k.Scopes, models.CommaSeparatedStrings{models.ScopeAllowCreateAccessKey, models.ScopeAllowApproveDeviceFlowRequest})
			},
		},
		{
			name: "successful signup with gmail",
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{
					Name:     "example@gmail.com",
					Password: "password",
					Org:      api.SignupOrg{Name: "acme-goog", Subdomain: "acme-goog-co"},
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				// the response is success
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

				respBody := &api.SignupResponse{}
				err := json.NewDecoder(resp.Body).Decode(respBody)
				assert.NilError(t, err)

				assert.Equal(t, respBody.User.Name, "example@gmail.com")
				assert.Equal(t, respBody.Organization.Name, "acme-goog")
				assert.DeepEqual(t, respBody.Organization.AllowedDomains, []string{""}) // this is empty by default for gmail
			},
		},
		{
			name: "successful signup with googlemail",
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{
					Name:     "example@googlemail.com",
					Password: "password",
					Org:      api.SignupOrg{Name: "acme-googmail", Subdomain: "acme-goog-mail-co"},
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				// the response is success
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

				respBody := &api.SignupResponse{}
				err := json.NewDecoder(resp.Body).Decode(respBody)
				assert.NilError(t, err)

				assert.Equal(t, respBody.User.Name, "example@googlemail.com")
				assert.Equal(t, respBody.Organization.Name, "acme-googmail")
				assert.DeepEqual(t, respBody.Organization.AllowedDomains, []string{""}) // this is empty by default for gmail
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

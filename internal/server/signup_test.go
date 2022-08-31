package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
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
		req, err := http.NewRequest(http.MethodPost, "/api/signup", jsonBody(t, body))
		assert.NilError(t, err)
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
						"length of string is 2, must be at least 3",
						"character '@' at position 1 is not allowed",
					}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
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
					Org:      api.SignupOrg{Name: "acme", Subdomain: "acme"},
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
			name: "successful signup",
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{
					Name:     "admin@example.com",
					Password: "password",
					Org:      api.SignupOrg{Name: "acme", Subdomain: "acme"},
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

				// the organization exists
				org, err := data.GetOrganization(srv.DB(), data.ByDomain("acme.exampledomain.com"))
				assert.NilError(t, err)
				assert.Equal(t, org.ID, respBody.Organization.ID)

				// the admin user has a valid access key
				httpResp := resp.Result()
				cookies := httpResp.Cookies()
				assert.Equal(t, len(cookies), 1)

				key := cookies[0].Value
				// nolint:noctx
				req, err := http.NewRequest(http.MethodGet, "/api/users/self", nil)
				assert.NilError(t, err)
				req.Header.Set("Infra-Version", apiVersionLatest)
				req.Header.Set("Authorization", "Bearer "+key)

				resp = httptest.NewRecorder()
				routes.ServeHTTP(resp, req)

				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				userResp := &api.User{}
				err = json.NewDecoder(resp.Body).Decode(userResp)
				assert.NilError(t, err)
				assert.Equal(t, userResp.ID, respBody.User.ID)

				// TODO: check the user is an admin
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

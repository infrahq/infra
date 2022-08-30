package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
)

func TestAPI_Signup(t *testing.T) {
	type testCase struct {
		name     string
		setup    func(t *testing.T) api.SignupRequest
		expected func(t *testing.T, response *httptest.ResponseRecorder)
	}

	srv := setupServer(t, withAdminUser)
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

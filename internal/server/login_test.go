package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestAPI_Login(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	// setup user to login as
	user := &models.Identity{Name: "steve"}
	err := data.CreateIdentity(srv.DB(), user)
	assert.NilError(t, err)

	p := data.InfraProvider(srv.DB())

	_, err = data.CreateProviderUser(srv.DB(), p, user)
	assert.NilError(t, err)

	hash, err := bcrypt.GenerateFromPassword([]byte("hunter2"), bcrypt.MinCost)
	assert.NilError(t, err)

	userCredential := &models.Credential{
		IdentityID:   user.ID,
		PasswordHash: hash,
	}

	err = data.CreateCredential(srv.DB(), userCredential)
	assert.NilError(t, err)

	type testCase struct {
		name     string
		setup    func(t *testing.T) api.LoginRequest
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.setup(t))
		req := httptest.NewRequest(http.MethodPost, "/api/login", body)
		req.Header.Add("Infra-Version", "0.13.3")

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		tc.expected(t, resp)
	}

	testCases := []testCase{
		{
			name: "login with username and password",
			setup: func(t *testing.T) api.LoginRequest {
				return api.LoginRequest{
					PasswordCredentials: &api.LoginRequestPasswordCredentials{
						Name:     "steve",
						Password: "hunter2",
					},
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

				loginResp := &api.LoginResponse{}
				err = json.Unmarshal(resp.Body.Bytes(), loginResp)
				assert.NilError(t, err)

				assert.Assert(t, loginResp.AccessKey != "")
				assert.Equal(t, len(resp.Result().Cookies()), 1)

				cookies := make(map[string]string)
				for _, c := range resp.Result().Cookies() {
					cookies[c.Name] = c.Value
				}

				assert.Equal(t, cookies["auth"], loginResp.AccessKey) // make sure the cookie matches the response
				assert.Equal(t, loginResp.UserID, user.ID)
				assert.Equal(t, loginResp.Name, "steve")
				assert.Equal(t, loginResp.PasswordUpdateRequired, false)
			},
		},
		{
			name: "missing login method",
			setup: func(t *testing.T) api.LoginRequest {
				return api.LoginRequest{}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{Errors: []string{"one of (accessKey, passwordCredentials, oidc) is required"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "too many login methods",
			setup: func(t *testing.T) api.LoginRequest {
				return api.LoginRequest{
					OIDC: &api.LoginRequestOIDC{
						Code:        "code",
						RedirectURL: "https://",
						ProviderID:  uid.ID(12345),
					},
					PasswordCredentials: &api.LoginRequestPasswordCredentials{
						Name:     "name",
						Password: "password",
					},
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{Errors: []string{"only one of (passwordCredentials, oidc) can have a value"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "missing password and name",
			setup: func(t *testing.T) api.LoginRequest {
				return api.LoginRequest{
					PasswordCredentials: &api.LoginRequestPasswordCredentials{},
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{
						FieldName: "passwordCredentials.name",
						Errors:    []string{"is required"},
					},
					{
						FieldName: "passwordCredentials.password",
						Errors:    []string{"is required"},
					},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "missing oidc fields",
			setup: func(t *testing.T) api.LoginRequest {
				return api.LoginRequest{OIDC: &api.LoginRequestOIDC{}}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{
						FieldName: "oidc.code",
						Errors:    []string{"is required"},
					},
					{
						FieldName: "oidc.providerID",
						Errors:    []string{"is required"},
					},
					{
						FieldName: "oidc.redirectURL",
						Errors:    []string{"is required"},
					},
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

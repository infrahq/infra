package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/generate"
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
				hash, err := bcrypt.GenerateFromPassword([]byte("hunter2"), bcrypt.MinCost)
				assert.NilError(t, err)

				userCredential := &models.Credential{
					IdentityID:   user.ID,
					PasswordHash: hash,
				}

				err = data.CreateCredential(srv.DB(), userCredential)
				assert.NilError(t, err)

				t.Cleanup(func() {
					err := data.DeleteCredential(srv.DB(), userCredential.ID)
					assert.NilError(t, err)
				})

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

				expectedResp := &api.LoginResponse{
					UserID:                 user.ID,
					Name:                   "steve",
					AccessKey:              "<any-string>",
					OrganizationName:       "Default",
					PasswordUpdateRequired: false,
					Expires:                api.Time(time.Now().UTC().Add(srv.options.SessionDuration)),
				}
				assert.DeepEqual(t, loginResp, expectedResp, cmpLoginResponse)

				expectedCookies := []*http.Cookie{
					{
						Name:     "auth",
						Value:    loginResp.AccessKey, // make sure the cookie matches the response
						Path:     "/",
						Domain:   "example.com",
						MaxAge:   600,
						HttpOnly: true,
						SameSite: http.SameSiteStrictMode,
					},
				}
				actual := resp.Result().Cookies()
				assert.DeepEqual(t, actual, expectedCookies, cmpSetCookies)
			},
		},
		{
			name: "login with temporary password",
			setup: func(t *testing.T) api.LoginRequest {
				tmpPassword, err := generate.CryptoRandom(12, generate.CharsetPassword)
				assert.NilError(t, err)

				pwHash, err := bcrypt.GenerateFromPassword([]byte(tmpPassword), bcrypt.DefaultCost)
				assert.NilError(t, err)

				userCredential := &models.Credential{
					IdentityID:      user.ID,
					PasswordHash:    pwHash,
					OneTimePassword: true,
				}

				err = data.CreateCredential(srv.db, userCredential)
				assert.NilError(t, err)

				t.Cleanup(func() {
					err := data.DeleteCredential(srv.DB(), userCredential.ID)
					assert.NilError(t, err)
				})

				return api.LoginRequest{
					PasswordCredentials: &api.LoginRequestPasswordCredentials{
						Name:     "steve",
						Password: tmpPassword,
					},
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

				loginResp := &api.LoginResponse{}
				err = json.Unmarshal(resp.Body.Bytes(), loginResp)
				assert.NilError(t, err)

				expectedResp := &api.LoginResponse{
					UserID:                 user.ID,
					Name:                   "steve",
					AccessKey:              "<any-string>",
					OrganizationName:       "Default",
					PasswordUpdateRequired: true,
					Expires:                api.Time(time.Now().UTC().Add(srv.options.SessionDuration)),
				}
				assert.DeepEqual(t, loginResp, expectedResp, cmpLoginResponse)

				expectedCookies := []*http.Cookie{
					{
						Name:     "auth",
						Value:    loginResp.AccessKey,
						Path:     "/",
						Domain:   "example.com",
						MaxAge:   600,
						HttpOnly: true,
						SameSite: http.SameSiteStrictMode,
					},
				}
				actual := resp.Result().Cookies()
				assert.DeepEqual(t, actual, expectedCookies, cmpSetCookies)
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

var cmpSetCookies = cmp.Options{
	cmp.FilterPath(opt.PathField(http.Cookie{}, "MaxAge"), cmpApproximateInt),
	cmp.FilterPath(opt.PathField(http.Cookie{}, "Raw"), cmp.Ignore()),
}

// cmpApproximateInt returns true if the two ints are different by less than 5.
// Most likely only useful when an int is used to measure duration in
// seconds.
var cmpApproximateInt = cmp.Comparer(func(x, y int) bool {
	if y > x {
		x, y = y, x
	}
	return y-x < 5
})

var cmpLoginResponse = cmp.Options{
	cmp.FilterPath(opt.PathField(api.LoginResponse{}, "AccessKey"), cmpAnyString),
	cmp.FilterPath(opt.PathField(api.LoginResponse{}, "Expires"),
		cmpApiTimeWithThreshold(10*time.Second)),
}

func cmpApiTimeWithThreshold(threshold time.Duration) cmp.Option {
	return cmp.Comparer(func(xa, ya api.Time) bool {
		x, y := time.Time(xa), time.Time(ya)
		if x.IsZero() || y.IsZero() {
			return false
		}
		delta := x.Sub(y)
		return delta <= threshold && delta >= -threshold
	})
}

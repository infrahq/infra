package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
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
			name: "no social or org sign-up",
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "", Errors: []string{"one of (social, user) is required"}},
					{FieldName: "orgName", Errors: []string{"is required"}},
					{FieldName: "subDomain", Errors: []string{"is required"}},
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
					User: &api.SignupUser{
						UserName: "admin@example.com",
						Password: "password",
					},
					OrgName:   "acme",
					Subdomain: "acme-co",
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestAPI_SignupSocial(t *testing.T) {
	type testCase struct {
		name     string
		client   *fakeOIDCImplementation
		setup    func(t *testing.T) api.SignupRequest
		expected func(t *testing.T, response *httptest.ResponseRecorder)
	}

	srv := setupServer(t, withAdminUser)
	srv.options.EnableSignup = true
	srv.options.BaseDomain = "exampledomain.com"
	srv.Google = &models.Provider{
		Model: models.Model{
			ID: models.InternalGoogleProviderID,
		},
		Name:         "Moogle",
		URL:          "example.com",
		ClientID:     "aaa",
		ClientSecret: models.EncryptedAtRest("bbb"),
		CreatedBy:    models.CreatedBySystem,
		Kind:         models.ProviderKindGoogle,
		AuthURL:      "https://example.com/o/oauth2/v2/auth",
		Scopes:       []string{"openid", "email"}, // TODO: update once our social client has groups
	}
	routes := srv.GenerateRoutes()

	existingOrg := models.Organization{
		Name:   "bmacd",
		Domain: "bmacd.exampledomain.com",
	}
	err := data.CreateOrganization(srv.DB(), &existingOrg)
	assert.NilError(t, err)

	run := func(t *testing.T, tc testCase) {
		body := tc.setup(t)

		// nolint:noctx
		req := httptest.NewRequest(http.MethodPost, "/api/signup", jsonBody(t, body))
		req.Header.Set("Infra-Version", apiVersionLatest)
		ctx := providers.WithOIDCClient(req.Context(), tc.client)
		*req = *req.WithContext(ctx)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := []testCase{
		{
			name:   "no social sign-up details",
			client: &fakeOIDCImplementation{},
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{Social: &api.SocialSignup{}}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "orgName", Errors: []string{"is required"}},
					{FieldName: "social.code", Errors: []string{"is required"}},
					{FieldName: "social.redirectURL", Errors: []string{"is required"}},
					{FieldName: "subDomain", Errors: []string{"is required"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "invalid social sign-up code",
			client: &fakeOIDCImplementation{
				FailExchange: true,
			},
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{
					Social: &api.SocialSignup{
						Code:        "1234",
						RedirectURL: "example.com/redirect",
					},
					OrgName:   "invalid-social-code",
					Subdomain: "invalid-social-code",
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				assert.Equal(t, respBody.Message, "unauthorized")
			},
		},
		{
			name: "successful social sign-up, unique org name",
			client: &fakeOIDCImplementation{
				UserEmail: "hello@bruce-macdonald.com",
			},
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{
					Social: &api.SocialSignup{
						Code:        "1234",
						RedirectURL: "example.com/redirect",
					},
					OrgName:   "success-social",
					Subdomain: "success-social",
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

				validateTestSignup := validateTestSignup{
					Routes: routes,
					Expected: &api.SignupResponse{
						User: &api.User{
							Name: "hello@bruce-macdonald.com",
						},
						Organization: &api.Organization{
							Name:           "success-social",
							Domain:         "success-social.exampledomain.com",
							AllowedDomains: []string{"bruce-macdonald.com"},
						},
					},
					Response: resp,
				}
				tx, err := srv.db.Begin(context.Background(), nil)
				assert.NilError(t, err)
				validateSuccessfulSignup(t, tx, validateTestSignup)
				assert.NilError(t, tx.Commit())
			},
		},
		{
			name: "successful social sign-up, gmail admin email",
			client: &fakeOIDCImplementation{
				UserEmail: "example@gmail.com",
			},
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{
					Social: &api.SocialSignup{
						Code:        "1234",
						RedirectURL: "example.com/redirect",
					},
					OrgName:   "success-gmail-social",
					Subdomain: "success-gmail-social",
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

				validateTestSignup := validateTestSignup{
					Routes: routes,
					Expected: &api.SignupResponse{
						User: &api.User{
							Name: "example@gmail.com",
						},
						Organization: &api.Organization{
							Name:           "success-gmail-social",
							Domain:         "success-gmail-social.exampledomain.com",
							AllowedDomains: []string{""},
						},
					},
					Response: resp,
				}
				tx, err := srv.db.Begin(context.Background(), nil)
				assert.NilError(t, err)
				validateSuccessfulSignup(t, tx, validateTestSignup)
				assert.NilError(t, tx.Commit())
			},
		},
		{
			name: "successful social sign-up, googlemail admin email",
			client: &fakeOIDCImplementation{
				UserEmail: "example@googlemail.com",
			},
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{
					Social: &api.SocialSignup{
						Code:        "1234",
						RedirectURL: "example.com/redirect",
					},
					OrgName:   "success-googlemail-social",
					Subdomain: "success-googlemail-social",
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

				validateTestSignup := validateTestSignup{
					Routes: routes,
					Expected: &api.SignupResponse{
						User: &api.User{
							Name: "example@googlemail.com",
						},
						Organization: &api.Organization{
							Name:           "success-googlemail-social",
							Domain:         "success-googlemail-social.exampledomain.com",
							AllowedDomains: []string{""},
						},
					},
					Response: resp,
				}
				tx, err := srv.db.Begin(context.Background(), nil)
				assert.NilError(t, err)
				validateSuccessfulSignup(t, tx, validateTestSignup)
				assert.NilError(t, tx.Commit())
			},
		},
		{
			name: "social sign-up, duplicate org name",
			client: &fakeOIDCImplementation{
				UserEmail: "hello@example.com",
			},
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{
					Social: &api.SocialSignup{
						Code:        "1234",
						RedirectURL: "example.com/redirect",
					},
					OrgName:   "bmacd",
					Subdomain: "bmacd",
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusConflict, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)
				assert.Equal(t, respBody.Message, "an organization with domain bmacd.exampledomain.com already exists")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestAPI_SignupOrg(t *testing.T) {
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
					User: &api.SignupUser{
						UserName: "admin@example.com",
						Password: "thispasswordisgreat",
					},
					OrgName:   "My org is awesome",
					Subdomain: "h@",
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "subDomain", Errors: []string{
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
					User: &api.SignupUser{
						UserName: "admin@example.com",
						Password: "short",
					},
					OrgName:   "My org is awesome",
					Subdomain: "helloo",
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "user.password", Errors: []string{
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
				return api.SignupRequest{User: &api.SignupUser{}}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "orgName", Errors: []string{"is required"}},
					{FieldName: "subDomain", Errors: []string{"is required"}},
					{FieldName: "user.password", Errors: []string{"is required"}},
					{FieldName: "user.username", Errors: []string{"is required"}},
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
					User: &api.SignupUser{
						UserName: "admin@example.com",
						Password: "password",
					},
					OrgName:   "acme",
					Subdomain: "acme-co",
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
					User: &api.SignupUser{
						UserName: "admin@example.com",
						Password: "password",
					},
					OrgName:   "Example",
					Subdomain: "taken",
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
					User: &api.SignupUser{
						UserName: "admin@example.com",
						Password: "password",
					},
					OrgName:   "acme",
					Subdomain: "acme-co",
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				// the response is success
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

				validateTestSignup := validateTestSignup{
					Routes: routes,
					Expected: &api.SignupResponse{
						User: &api.User{
							Name: "admin@example.com",
						},
						Organization: &api.Organization{
							Name:           "acme",
							Domain:         "acme-co.exampledomain.com",
							AllowedDomains: []string{"example.com"},
						},
					},
					Response: resp,
				}
				tx, err := srv.db.Begin(context.Background(), nil)
				assert.NilError(t, err)
				validateSuccessfulSignup(t, tx, validateTestSignup)
				assert.NilError(t, tx.Commit())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

type validateTestSignup struct {
	Routes   Routes
	Expected *api.SignupResponse
	Response *httptest.ResponseRecorder
}

func validateSuccessfulSignup(t *testing.T, db *data.Transaction, testSignup validateTestSignup) {
	respBody := &api.SignupResponse{}
	err := json.NewDecoder(testSignup.Response.Body).Decode(respBody)
	assert.NilError(t, err)

	assert.Equal(t, respBody.User.Name, testSignup.Expected.User.Name)
	assert.Equal(t, respBody.Organization.Name, testSignup.Expected.Organization.Name)
	orgDomainMatch, err := regexp.MatchString(testSignup.Expected.Organization.Domain, respBody.Organization.Domain)
	assert.NilError(t, err)
	assert.Assert(t, orgDomainMatch)
	userID := respBody.User.ID

	// the organization exists
	_, err = data.GetOrganization(db, data.GetOrganizationOptions{
		ByDomain: respBody.Organization.Domain,
	})
	assert.NilError(t, err)

	// the admin user has a valid access key
	httpResp := testSignup.Response.Result()
	cookies := httpResp.Cookies()
	assert.Equal(t, len(cookies), 1)

	key := cookies[0].Value
	// nolint:noctx
	req := httptest.NewRequest(http.MethodGet, "/api/users/self", nil)
	req.Header.Set("Infra-Version", apiVersionLatest)
	req.Header.Set("Authorization", "Bearer "+key)

	resp := httptest.NewRecorder()
	testSignup.Routes.ServeHTTP(resp, req)

	assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
	userResp := &api.User{}
	err = json.NewDecoder(resp.Body).Decode(userResp)
	assert.NilError(t, err)
	assert.Equal(t, userResp.ID, respBody.User.ID)

	// check the user is an admin
	db = db.WithOrgID(respBody.Organization.ID)
	_, err = data.GetGrant(db, data.GetGrantOptions{
		BySubject:   uid.NewIdentityPolymorphicID(userID),
		ByResource:  access.ResourceInfraAPI,
		ByPrivilege: api.InfraAdminRole,
	})
	assert.NilError(t, err)

	// check their access token has the expected scope
	k, err := data.GetAccessKeyByKeyID(db, strings.Split(key, ".")[0])
	assert.NilError(t, err)

	assert.DeepEqual(t, k.Scopes, models.CommaSeparatedStrings{models.ScopeAllowCreateAccessKey})
}

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
					{FieldName: "", Errors: []string{"one of (social, org) is required"}},
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
					Org: &api.SignupOrg{
						UserName:  "admin@example.com",
						Password:  "password",
						OrgName:   "acme",
						Subdomain: "acme-co",
					},
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
	routes := srv.GenerateRoutes()

	existingOrg := models.Organization{
		Name:   "bmacd",
		Domain: "bmacd.exampledomain.com",
	}
	err := data.CreateOrganization(srv.DB(), &existingOrg)
	assert.NilError(t, err)

	socialProvider := models.Provider{Name: "moogle", Kind: models.ProviderKindGoogle, SocialLogin: true}
	err = data.CreateSocialLoginProvider(srv.DB(), &socialProvider)
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
					{FieldName: "social.code", Errors: []string{"is required"}},
					{FieldName: "social.kind", Errors: []string{"is required"}},
					{FieldName: "social.redirectURL", Errors: []string{"is required"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name:   "invalid social sign-up kind",
			client: &fakeOIDCImplementation{},
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{Social: &api.SocialSignup{
					Kind:        "okta",
					Code:        "1234",
					RedirectURL: "example.com/redirect",
				}}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusNotFound, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				assert.Equal(t, respBody.Message, "invalid social identity provider: record not found")
			},
		},
		{
			name: "invalid social sign-up code",
			client: &fakeOIDCImplementation{
				FailExchange: true,
			},
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{Social: &api.SocialSignup{
					Kind:        models.ProviderKindGoogle.String(),
					Code:        "1234",
					RedirectURL: "example.com/redirect",
				}}
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
			name: "invalid social sign-up, empty email domain",
			client: &fakeOIDCImplementation{
				UserEmail: "hello@",
			},
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{Social: &api.SocialSignup{
					Kind:        models.ProviderKindGoogle.String(),
					Code:        "1234",
					RedirectURL: "example.com/redirect",
				}}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())
				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				assert.Equal(t, respBody.Message, "bad request: get email domain: invalid email domain")
			},
		},
		{
			name: "successful social sign-up, org domain, unique org name",
			client: &fakeOIDCImplementation{
				UserEmail: "hello@bruce-macdonald.com",
			},
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{Social: &api.SocialSignup{
					Kind:        models.ProviderKindGoogle.String(),
					Code:        "1234",
					RedirectURL: "example.com/redirect",
				}}
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
							Name:   "bruce-macdonald",
							Domain: "bruce-macdonald.exampledomain.com",
						},
					},
					Response: resp,
				}
				tx, err := srv.db.Begin(context.Background(), nil)
				defer tx.Commit()
				assert.NilError(t, err)
				validateSuccessfulSignup(t, tx, validateTestSignup)
			},
		},
		{
			name: "successful social sign-up, org domain, duplicate org name",
			client: &fakeOIDCImplementation{
				UserEmail: "hello@bmacd.xyz",
			},
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{Social: &api.SocialSignup{
					Kind:        models.ProviderKindGoogle.String(),
					Code:        "1234",
					RedirectURL: "example.com/redirect",
				}}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

				validateTestSignup := validateTestSignup{
					Routes: routes,
					Expected: &api.SignupResponse{
						User: &api.User{
							Name: "hello@bmacd.xyz",
						},
						Organization: &api.Organization{
							Name:   "bmacd",
							Domain: `bmacd-\d\d\d.exampledomain.com`,
						},
					},
					Response: resp,
				}
				tx, err := srv.db.Begin(context.Background(), nil)
				defer tx.Commit()
				assert.NilError(t, err)
				validateSuccessfulSignup(t, tx, validateTestSignup)
			},
		},
		{
			name: "successful social sign-up, gmail domain",
			client: &fakeOIDCImplementation{
				UserEmail: "bruce@gmail.com",
			},
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{Social: &api.SocialSignup{
					Kind:        models.ProviderKindGoogle.String(),
					Code:        "1234",
					RedirectURL: "example.com/redirect",
				}}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

				validateTestSignup := validateTestSignup{
					Routes: routes,
					Expected: &api.SignupResponse{
						User: &api.User{
							Name: "bruce@gmail.com",
						},
						Organization: &api.Organization{
							Name:   "bruce",
							Domain: `bruce.exampledomain.com`,
						},
					},
					Response: resp,
				}
				tx, err := srv.db.Begin(context.Background(), nil)
				defer tx.Commit()
				assert.NilError(t, err)
				validateSuccessfulSignup(t, tx, validateTestSignup)
			},
		},
		{
			name: "successful social sign-up, short domain",
			client: &fakeOIDCImplementation{
				UserEmail: "hello@g.com",
			},
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{Social: &api.SocialSignup{
					Kind:        models.ProviderKindGoogle.String(),
					Code:        "1234",
					RedirectURL: "example.com/redirect",
				}}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

				validateTestSignup := validateTestSignup{
					Routes: routes,
					Expected: &api.SignupResponse{
						User: &api.User{
							Name: "hello@g.com",
						},
						Organization: &api.Organization{
							Name:   "g",
							Domain: `g-\d\d\d.exampledomain.com`,
						},
					},
					Response: resp,
				}
				tx, err := srv.db.Begin(context.Background(), nil)
				defer tx.Commit()
				assert.NilError(t, err)
				validateSuccessfulSignup(t, tx, validateTestSignup)
			},
		},
		{
			name: "successful social sign-up, reserved domain",
			client: &fakeOIDCImplementation{
				UserEmail: "hello@infrahq.com",
			},
			setup: func(t *testing.T) api.SignupRequest {
				return api.SignupRequest{Social: &api.SocialSignup{
					Kind:        models.ProviderKindGoogle.String(),
					Code:        "1234",
					RedirectURL: "example.com/redirect",
				}}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

				validateTestSignup := validateTestSignup{
					Routes: routes,
					Expected: &api.SignupResponse{
						User: &api.User{
							Name: "hello@infrahq.com",
						},
						Organization: &api.Organization{
							Name:   "infrahq",
							Domain: `infrahq-\d\d\d.exampledomain.com`,
						},
					},
					Response: resp,
				}
				tx, err := srv.db.Begin(context.Background(), nil)
				defer tx.Commit()
				assert.NilError(t, err)
				validateSuccessfulSignup(t, tx, validateTestSignup)
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
					Org: &api.SignupOrg{
						UserName:  "admin@example.com",
						Password:  "thispasswordisgreat",
						OrgName:   "My org is awesome",
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
					Org: &api.SignupOrg{
						UserName:  "admin@example.com",
						Password:  "short",
						OrgName:   "My org is awesome",
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
					{FieldName: "org.password", Errors: []string{
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
				return api.SignupRequest{Org: &api.SignupOrg{}}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "org.orgName", Errors: []string{"is required"}},
					{FieldName: "org.password", Errors: []string{"is required"}},
					{FieldName: "org.subDomain", Errors: []string{"is required"}},
					{FieldName: "org.userName", Errors: []string{"is required"}},
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
					Org: &api.SignupOrg{
						UserName:  "admin@example.com",
						Password:  "password",
						OrgName:   "acme",
						Subdomain: "acme-co",
					},
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
					Org: &api.SignupOrg{
						UserName:  "admin@example.com",
						Password:  "password",
						OrgName:   "Example",
						Subdomain: "taken",
					},
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
					Org: &api.SignupOrg{
						UserName:  "admin@example.com",
						Password:  "password",
						OrgName:   "acme",
						Subdomain: "acme-co",
					},
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
							Name:   "acme",
							Domain: "acme-co.exampledomain.com",
						},
					},
					Response: resp,
				}
				tx, err := srv.db.Begin(context.Background(), nil)
				defer tx.Commit()
				assert.NilError(t, err)
				validateSuccessfulSignup(t, tx, validateTestSignup)
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

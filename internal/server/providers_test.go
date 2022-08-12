package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"
	"k8s.io/utils/strings/slices"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
)

func TestAPI_ListProviders(t *testing.T) {
	s := setupServer(t, withAdminUser)
	routes := s.GenerateRoutes(prometheus.NewRegistry())

	testProvider := &models.Provider{
		Name:    "mokta",
		Kind:    models.ProviderKindOkta,
		AuthURL: "https://example.com/v1/auth",
		Scopes:  []string{"openid", "email"},
	}

	err := data.CreateProvider(s.db, testProvider)
	assert.NilError(t, err)

	dbProviders, err := data.ListProviders(s.db, nil)
	assert.NilError(t, err)
	assert.Equal(t, len(dbProviders), 2)

	// nolint:noctx
	req, err := http.NewRequest(http.MethodGet, "/api/providers", nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", "Bearer "+adminAccessKey(s))
	req.Header.Add("Infra-Version", "0.12.3")

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

	var apiProviders api.ListResponse[Provider]
	err = json.Unmarshal(resp.Body.Bytes(), &apiProviders)
	assert.NilError(t, err)

	assert.Equal(t, len(apiProviders.Items), 1)
	assert.Equal(t, apiProviders.Items[0].Name, "mokta")
	assert.Equal(t, apiProviders.Items[0].AuthURL, "https://example.com/v1/auth")
	assert.Assert(t, slices.Equal(apiProviders.Items[0].Scopes, []string{"openid", "email"}))
}

func TestAPI_DeleteProvider(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	createProvider := func(t *testing.T) *models.Provider {
		t.Helper()
		p := &models.Provider{Name: "mokta", Kind: models.ProviderKindOkta}
		err := data.CreateProvider(srv.db, p)
		assert.NilError(t, err)
		return p
	}

	provider1 := createProvider(t)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		req, err := http.NewRequest(http.MethodDelete, tc.urlPath, nil)
		assert.NilError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"not authenticated": {
			urlPath: "/api/providers/1234",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
			},
		},
		"not authorized": {
			urlPath: "/api/providers/2341",
			setup: func(t *testing.T, req *http.Request) {
				key, _ := createAccessKey(t, srv.db, "someonenew@example.com")
				req.Header.Set("Authorization", "Bearer "+key)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusForbidden, resp.Code, resp.Body.String())
			},
		},
		"successful delete": {
			urlPath: "/api/providers/" + provider1.ID.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
			},
		},
		"infra provider can not be deleted": {
			urlPath: "/api/providers/" + data.InfraProvider(srv.db).ID.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestAPI_CreateProvider(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	type testCase struct {
		name     string
		body     api.CreateProviderRequest
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, response *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.body)

		req, err := http.NewRequest(http.MethodPost, "/api/providers", body)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := []testCase{
		{
			name: "not authenticated",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized, resp.Body.String())
			},
		},
		{
			name: "not authorized",
			body: api.CreateProviderRequest{
				Name:         "olive",
				URL:          "https://example.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
			setup: func(t *testing.T, req *http.Request) {
				accessKey, _ := createAccessKey(t, srv.db, "usera@example.com")
				req.Header.Set("Authorization", "Bearer "+accessKey)

				ctx := providers.WithOIDCClient(req.Context(), &fakeOIDCImplementation{})
				*req = *req.WithContext(ctx)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden, resp.Body.String())
			},
		},
		{
			name: "missing required fields",
			body: api.CreateProviderRequest{},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "clientID", Errors: []string{"is required"}},
					{FieldName: "clientSecret", Errors: []string{"is required"}},
					{FieldName: "name", Errors: []string{"is required"}},
					{FieldName: "url", Errors: []string{"is required"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "api credential invalid emails",
			body: api.CreateProviderRequest{
				Name:         "google",
				URL:          "accounts.google.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				Kind:         string(models.ProviderKindGoogle),
				API: &api.ProviderAPICredentials{
					ClientEmail:      "notanemail",
					DomainAdminEmail: "domainadmin",
				},
			},
			setup: func(t *testing.T, req *http.Request) {
				ctx := providers.WithOIDCClient(req.Context(), &fakeOIDCImplementation{})
				*req = *req.WithContext(ctx)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "api.clientEmail", Errors: []string{"invalid email address"}},
					{FieldName: "api.domainAdminEmail", Errors: []string{"invalid email address"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "invalid kind",
			body: api.CreateProviderRequest{
				Name:         "olive",
				URL:          "https://example.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				Kind:         "vegetable",
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "kind", Errors: []string{"must be one of (oidc, okta, azure, google)"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "valid provider (no external checks)",
			body: api.CreateProviderRequest{
				Name:         "google",
				URL:          "accounts.google.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				Kind:         string(models.ProviderKindGoogle),
				API: &api.ProviderAPICredentials{
					PrivateKey:       "-----BEGIN PRIVATE KEY-----\naaa=\n-----END PRIVATE KEY-----\n",
					ClientEmail:      "example@tenant.iam.gserviceaccount.com",
					DomainAdminEmail: "admin@example.com",
				},
			},
			setup: func(t *testing.T, req *http.Request) {
				ctx := providers.WithOIDCClient(req.Context(), &fakeOIDCImplementation{})
				*req = *req.WithContext(ctx)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

				respBody := &api.Provider{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := &api.Provider{
					ID:       respBody.ID, // does not matter
					Name:     "google",
					Created:  respBody.Created, // does not matter
					Updated:  respBody.Updated, // does not matter
					URL:      "accounts.google.com",
					ClientID: "client-id",
					Kind:     string(models.ProviderKindGoogle),
					AuthURL:  "example.com/v1/auth",
					Scopes:   []string{"openid", "email"},
				}
				assert.DeepEqual(t, respBody, expected)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestAPI_UpdateProvider(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	provider := &models.Provider{
		Name:    "private",
		Kind:    models.ProviderKindAzure,
		AuthURL: "https://example.com/v1/auth",
		Scopes:  []string{"openid", "email"},
	}

	err := data.CreateProvider(srv.db, provider)
	assert.NilError(t, err)

	type testCase struct {
		name     string
		body     api.UpdateProviderRequest
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, response *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.body)

		id := provider.ID.String()
		req, err := http.NewRequest(http.MethodPut, "/api/providers/"+id, body)
		assert.NilError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := []testCase{
		{
			name: "not authenticated",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized, resp.Body.String())
			},
		},
		{
			name: "not authorized",
			body: api.UpdateProviderRequest{
				Name:         "olive",
				URL:          "https://example.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
			setup: func(t *testing.T, req *http.Request) {
				accessKey, _ := createAccessKey(t, srv.db, "usera@example.com")
				req.Header.Set("Authorization", "Bearer "+accessKey)

				ctx := providers.WithOIDCClient(req.Context(), &fakeOIDCImplementation{})
				*req = *req.WithContext(ctx)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden, resp.Body.String())
			},
		},
		{
			name: "missing required fields",
			body: api.UpdateProviderRequest{},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "clientID", Errors: []string{"is required"}},
					{FieldName: "clientSecret", Errors: []string{"is required"}},
					{FieldName: "name", Errors: []string{"is required"}},
					{FieldName: "url", Errors: []string{"is required"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "invalid kind",
			body: api.UpdateProviderRequest{
				Name:         "olive",
				URL:          "https://example.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				Kind:         "vegetable",
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "kind", Errors: []string{"must be one of (oidc, okta, azure, google)"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "valid provider (no external checks)",
			body: api.UpdateProviderRequest{
				Name:         "google",
				URL:          "accounts.google.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				Kind:         string(models.ProviderKindGoogle),
				API: &api.ProviderAPICredentials{
					PrivateKey:       "-----BEGIN PRIVATE KEY-----\naaa=\n-----END PRIVATE KEY-----\n",
					ClientEmail:      "example@tenant.iam.gserviceaccount.com",
					DomainAdminEmail: "admin@example.com",
				},
			},
			setup: func(t *testing.T, req *http.Request) {
				ctx := providers.WithOIDCClient(req.Context(), &fakeOIDCImplementation{})
				*req = *req.WithContext(ctx)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				respBody := &api.Provider{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := &api.Provider{
					ID:       respBody.ID, // does not matter
					Name:     "google",
					Created:  respBody.Created, // does not matter
					Updated:  respBody.Updated, // does not matter
					URL:      "accounts.google.com",
					ClientID: "client-id",
					Kind:     string(models.ProviderKindGoogle),
					AuthURL:  "example.com/v1/auth",
					Scopes:   []string{"openid", "email"},
				}
				assert.DeepEqual(t, respBody, expected)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// mockOIDC is a fake oidc identity provider
type fakeOIDCImplementation struct {
	UserInfoRevoked bool // when true returns an error fromt the user info endpoint
}

func (m *fakeOIDCImplementation) Validate(_ context.Context) error {
	return nil
}

func (m *fakeOIDCImplementation) AuthServerInfo(_ context.Context) (*providers.AuthServerInfo, error) {
	return &providers.AuthServerInfo{AuthURL: "example.com/v1/auth", ScopesSupported: []string{"openid", "email"}}, nil
}

func (m *fakeOIDCImplementation) ExchangeAuthCodeForProviderTokens(_ context.Context, _ string) (acc, ref string, exp time.Time, email string, err error) {
	return "acc", "ref", exp, "", nil
}

func (m *fakeOIDCImplementation) RefreshAccessToken(_ context.Context, providerUser *models.ProviderUser) (accessToken string, expiry *time.Time, err error) {
	// never update
	return string(providerUser.AccessToken), &providerUser.ExpiresAt, nil
}

func (m *fakeOIDCImplementation) GetUserInfo(_ context.Context, _ *models.ProviderUser) (*providers.UserInfoClaims, error) {
	if m.UserInfoRevoked {
		return nil, fmt.Errorf("user revoked")
	}
	return &providers.UserInfoClaims{}, nil
}

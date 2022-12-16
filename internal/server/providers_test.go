package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"k8s.io/utils/strings/slices"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
)

func TestAPI_ListProviders(t *testing.T) {
	s := setupServer(t, withAdminUser)
	routes := s.GenerateRoutes()

	testProvider := &models.Provider{
		Name:    "mokta",
		Kind:    models.ProviderKindOkta,
		AuthURL: "https://example.com/v1/auth",
		Scopes:  []string{"openid", "email"},
	}

	err := data.CreateProvider(s.DB(), testProvider)
	assert.NilError(t, err)

	dbProviders, err := data.ListProviders(s.DB(), data.ListProvidersOptions{})
	assert.NilError(t, err)
	assert.Equal(t, len(dbProviders), 2) // infra provider and mokta

	t.Run("list providers returns providers for org", func(t *testing.T) {
		// nolint:noctx
		req, err := http.NewRequest(http.MethodGet, "/api/providers", nil)
		assert.NilError(t, err)

		user := &models.Identity{Name: "bruce@example.com"}
		err = data.CreateIdentity(s.DB(), user)
		assert.NilError(t, err)

		key := &models.AccessKey{
			IssuedFor:  user.ID,
			ProviderID: testProvider.ID,
			ExpiresAt:  time.Now().Add(-1 * time.Minute),
		}
		bearer, err := data.CreateAccessKey(s.DB(), key)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+bearer)
		req.Header.Add("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

		var apiProviders api.ListResponse[api.Provider]
		err = json.Unmarshal(resp.Body.Bytes(), &apiProviders)
		assert.NilError(t, err)

		assert.Equal(t, len(apiProviders.Items), 1) // infra provider is not returned
		assert.Equal(t, apiProviders.Items[0].Name, testProvider.Name)
		assert.Equal(t, apiProviders.Items[0].AuthURL, testProvider.AuthURL)
		assert.Assert(t, slices.Equal(apiProviders.Items[0].Scopes, testProvider.Scopes))
	})
}

func TestAPI_GetProvider(t *testing.T) {
	s := setupServer(t, withAdminUser)
	routes := s.GenerateRoutes()

	testProvider := &models.Provider{
		Name:    "mokta",
		Kind:    models.ProviderKindOkta,
		AuthURL: "https://example.com/v1/auth",
		Scopes:  []string{"openid", "email"},
	}

	err := data.CreateProvider(s.DB(), testProvider)
	assert.NilError(t, err)

	dbProviders, err := data.ListProviders(s.DB(), data.ListProvidersOptions{})
	assert.NilError(t, err)
	assert.Equal(t, len(dbProviders), 2) // infra provider and mokta

	t.Run("get provider with access key for org returns provider with sensitive fields", func(t *testing.T) {
		// nolint:noctx
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/providers/%s", testProvider.ID), nil)
		assert.NilError(t, err)

		req.Header.Add("Authorization", "Bearer "+adminAccessKey(s))
		req.Header.Add("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

		var provider api.Provider
		err = json.Unmarshal(resp.Body.Bytes(), &provider)
		assert.NilError(t, err)

		assert.Equal(t, provider.Name, testProvider.Name)
		assert.Equal(t, provider.AuthURL, testProvider.AuthURL)
		assert.Assert(t, slices.Equal(provider.Scopes, testProvider.Scopes))
	})
	t.Run("get provider with no access key for org returns provider without fields", func(t *testing.T) {
		// nolint:noctx
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/providers/%s", testProvider.ID), nil)
		assert.NilError(t, err)

		req.Header.Add("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

		var provider api.Provider
		err = json.Unmarshal(resp.Body.Bytes(), &provider)
		assert.NilError(t, err)

		assert.Equal(t, provider.Name, testProvider.Name)
		assert.Equal(t, provider.AuthURL, testProvider.AuthURL)
		assert.Assert(t, slices.Equal(provider.Scopes, testProvider.Scopes))
	})
	t.Run("get provider with expired access key for org returns provider without fields", func(t *testing.T) {
		// nolint:noctx
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/providers/%s", testProvider.ID), nil)
		assert.NilError(t, err)

		user := &models.Identity{Name: "bruce@example.com"}
		err = data.CreateIdentity(s.DB(), user)
		assert.NilError(t, err)

		key := &models.AccessKey{
			IssuedFor:  user.ID,
			ProviderID: testProvider.ID,
			ExpiresAt:  time.Now().Add(-1 * time.Minute),
		}
		bearer, err := data.CreateAccessKey(s.DB(), key)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+bearer)

		req.Header.Add("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

		var provider api.Provider
		err = json.Unmarshal(resp.Body.Bytes(), &provider)
		assert.NilError(t, err)

		assert.Equal(t, provider.Name, testProvider.Name)
		assert.Equal(t, provider.AuthURL, testProvider.AuthURL)
		assert.Assert(t, slices.Equal(provider.Scopes, testProvider.Scopes))
	})
}

func TestAPI_DeleteProvider(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	createProvider := func(t *testing.T) *models.Provider {
		t.Helper()
		p := &models.Provider{Name: "mokta", Kind: models.ProviderKindOkta}
		err := data.CreateProvider(srv.DB(), p)
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
		req := httptest.NewRequest(http.MethodDelete, tc.urlPath, nil)
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
				key, _ := createAccessKey(t, srv.DB(), "someonenew@example.com")
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
			urlPath: "/api/providers/" + data.InfraProvider(srv.DB()).ID.String(),
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
	routes := srv.GenerateRoutes()

	type testCase struct {
		name     string
		body     api.CreateProviderRequest
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, response *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.body)

		req := httptest.NewRequest(http.MethodPost, "/api/providers", body)
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
				accessKey, _ := createAccessKey(t, srv.DB(), "usera@example.com")
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
			name: "valid provider (name is generated to default, providerkind)",
			body: api.CreateProviderRequest{
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
		{
			name: "valid provider (name is generated but default is already taken)",
			body: api.CreateProviderRequest{
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
					ID:       respBody.ID,      // does not matter
					Name:     respBody.Name,    // does not matter
					Created:  respBody.Created, // does not matter
					Updated:  respBody.Updated, // does not matter
					URL:      "accounts.google.com",
					ClientID: "client-id",
					Kind:     string(models.ProviderKindGoogle),
					AuthURL:  "example.com/v1/auth",
					Scopes:   []string{"openid", "email"},
				}
				assert.DeepEqual(t, respBody, expected)
				assert.Assert(t, respBody.Name != string(models.ProviderKindGoogle))
			},
		},
		{
			name: "valid provider (name is provided)",
			body: api.CreateProviderRequest{
				Name:         "google-123",
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
					Name:     "google-123",
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
	routes := srv.GenerateRoutes()

	provider := &models.Provider{
		Name:    "private",
		Kind:    models.ProviderKindAzure,
		AuthURL: "https://example.com/v1/auth",
		Scopes:  []string{"openid", "email"},
	}

	err := data.CreateProvider(srv.DB(), provider)
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
		req := httptest.NewRequest(http.MethodPut, "/api/providers/"+id, body)
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
				accessKey, _ := createAccessKey(t, srv.DB(), "usera@example.com")
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

func TestAPI_PatchProvider(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	provider := &models.Provider{
		Name:         "name",
		Kind:         models.ProviderKindOkta,
		ClientSecret: "secret",
	}

	err := data.CreateProvider(srv.DB(), provider)
	assert.NilError(t, err)

	type testCase struct {
		name     string
		body     api.PatchProviderRequest
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, response *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.body)

		id := provider.ID.String()
		req := httptest.NewRequest(http.MethodPatch, "/api/providers/"+id, body)
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
			body: api.PatchProviderRequest{
				Name:         "olive",
				ClientSecret: "client-secret",
			},
			setup: func(t *testing.T, req *http.Request) {
				accessKey, _ := createAccessKey(t, srv.DB(), "usera@example.com")
				req.Header.Set("Authorization", "Bearer "+accessKey)

				ctx := providers.WithOIDCClient(req.Context(), &fakeOIDCImplementation{})
				*req = *req.WithContext(ctx)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden, resp.Body.String())
			},
		},
		{
			name: "valid provider (no external checks)",
			body: api.PatchProviderRequest{
				Name:         "new-name-google",
				ClientSecret: "new-client-secret",
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
					ID:      respBody.ID, // does not matter
					Name:    "new-name-google",
					Created: respBody.Created, // does not matter
					Updated: respBody.Updated, // does not matter
					Kind:    respBody.Kind,
				}
				assert.DeepEqual(t, respBody, expected, cmpopts.EquateEmpty())

				// Test secret, which is not returned from the api
				savedProvider, err := data.GetProvider(srv.DB(), data.GetProviderOptions{ByID: provider.ID})
				assert.NilError(t, err)
				assert.Equal(t, savedProvider.ClientSecret, models.EncryptedAtRest("new-client-secret"))
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
	UserInfoRevoked bool   // when true returns an error fromt the user info endpoint
	FailExchange    bool   // when true auth code exchange fails
	UserEmail       string // the email returned from the fake identity provider
}

func (m *fakeOIDCImplementation) Validate(_ context.Context) error {
	return nil
}

func (m *fakeOIDCImplementation) AuthServerInfo(_ context.Context) (*providers.AuthServerInfo, error) {
	return &providers.AuthServerInfo{AuthURL: "example.com/v1/auth", ScopesSupported: []string{"openid", "email"}}, nil
}

func (m *fakeOIDCImplementation) ExchangeAuthCodeForProviderTokens(_ context.Context, _ string) (*providers.IdentityProviderAuth, error) {
	if m.FailExchange {
		return nil, fmt.Errorf("invalid auth code")
	}
	return &providers.IdentityProviderAuth{
		AccessToken:       "acc",
		RefreshToken:      "ref",
		AccessTokenExpiry: time.Now().Add(1 * time.Minute),
		Email:             m.UserEmail,
	}, nil
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

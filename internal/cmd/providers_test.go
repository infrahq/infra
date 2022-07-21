package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"

	"github.com/infrahq/infra/api"
)

func TestProvidersAddCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	setup := func(t *testing.T) chan api.CreateProviderRequest {
		requestCh := make(chan api.CreateProviderRequest, 1)

		handler := func(resp http.ResponseWriter, req *http.Request) {
			if !strings.Contains(req.URL.Path, "/api/providers") {
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}

			switch req.Method {
			case http.MethodPost:
				var createRequest api.CreateProviderRequest
				err := json.NewDecoder(req.Body).Decode(&createRequest)
				assert.Check(t, err)

				requestCh <- createRequest

				_, _ = resp.Write([]byte(`{}`))
				return
			case http.MethodGet:
				var apiProviders []api.Provider
				apiProviders = append(apiProviders, api.Provider{
					Name:     "okta",
					URL:      "https://okta.com/path",
					ClientID: "okta-client-id",
				})
				b, err := json.Marshal(api.ListResponse[api.Provider]{
					Items: apiProviders,
					Count: len(apiProviders),
				})
				assert.NilError(t, err)
				_, _ = resp.Write(b)
				return
			}
		}
		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.User{})
		err := writeConfig(&cfg)
		assert.NilError(t, err)
		return requestCh
	}

	t.Run("okta provider with flags", func(t *testing.T) {
		ch := setup(t)

		err := Run(context.Background(),
			"providers", "add", "okta",
			"--url", "https://okta.com/path",
			"--client-id", "okta-client-id",
			"--client-secret", "okta-client-secret",
			"--kind", "oidc",
		)
		assert.NilError(t, err)

		createProviderRequest := <-ch

		expected := api.CreateProviderRequest{
			Name:         "okta",
			URL:          "https://okta.com/path",
			ClientID:     "okta-client-id",
			ClientSecret: "okta-client-secret",
			Kind:         "oidc",
			API:          &api.ProviderAPICredentials{},
		}
		assert.DeepEqual(t, createProviderRequest, expected)
	})

	t.Run("okta provider with env vars", func(t *testing.T) {
		ch := setup(t)

		t.Setenv("INFRA_PROVIDER_URL", "https://okta.com/path")
		t.Setenv("INFRA_PROVIDER_CLIENT_ID", "okta-client-id")
		t.Setenv("INFRA_PROVIDER_CLIENT_SECRET", "okta-client-secret")

		err := Run(context.Background(), "providers", "add", "okta")
		assert.NilError(t, err)

		createProviderRequest := <-ch

		expected := api.CreateProviderRequest{
			Name:         "okta",
			URL:          "https://okta.com/path",
			ClientID:     "okta-client-id",
			ClientSecret: "okta-client-secret",
			Kind:         "oidc",
			API:          &api.ProviderAPICredentials{},
		}
		assert.DeepEqual(t, createProviderRequest, expected)
	})

	t.Run("google provider with no api flags", func(t *testing.T) {
		ch := setup(t)

		err := Run(context.Background(),
			"providers", "add", "google",
			"--url", "accounts.google.com",
			"--client-id", "aaa.apps.googleusercontent.com",
			"--client-secret", "GOCSPX-bbb",
			"--kind", "google",
		)
		assert.NilError(t, err)

		createProviderRequest := <-ch

		expected := api.CreateProviderRequest{
			Name:         "google",
			URL:          "accounts.google.com",
			ClientID:     "aaa.apps.googleusercontent.com",
			ClientSecret: "GOCSPX-bbb",
			Kind:         "google",
			API:          &api.ProviderAPICredentials{},
		}
		assert.DeepEqual(t, createProviderRequest, expected)
	})

	t.Run("google provider with api flags", func(t *testing.T) {
		ch := setup(t)

		err := Run(context.Background(),
			"providers", "add", "google",
			"--url", "accounts.google.com",
			"--client-id", "aaa.apps.googleusercontent.com",
			"--client-secret", "GOCSPX-bbb",
			"--kind", "google",
			"--service-account--key", "-----BEGIN PRIVATE KEY-----\naaa=\n-----END PRIVATE KEY-----\n",
			"--service-account--email", "example@tenant.iam.gserviceaccount.com",
			"--domain-admin", "admin@example.com",
		)
		assert.NilError(t, err)

		createProviderRequest := <-ch

		expected := api.CreateProviderRequest{
			Name:         "google",
			URL:          "accounts.google.com",
			ClientID:     "aaa.apps.googleusercontent.com",
			ClientSecret: "GOCSPX-bbb",
			Kind:         "google",
			API: &api.ProviderAPICredentials{
				PrivateKey:       api.PEM("-----BEGIN PRIVATE KEY-----\naaa=\n-----END PRIVATE KEY-----\n"),
				ClientEmail:      "example@tenant.iam.gserviceaccount.com",
				DomainAdminEmail: "admin@example.com",
			},
		}
		assert.DeepEqual(t, createProviderRequest, expected)
	})

	t.Run("api flags cannot be specified for non-google kind", func(t *testing.T) {
		err := Run(context.Background(),
			"providers", "add", "okta",
			"--url", "example.okta.com",
			"--client-id", "aaa",
			"--client-secret", "bbb",
			"--kind", "okta",
			"--service-account--key", "-----BEGIN PRIVATE KEY-----\naaa=\n-----END PRIVATE KEY-----\n",
			"--service-account--email", "example@tenant.iam.gserviceaccount.com",
			"--domain-admin", "admin@example.com",
		)

		assert.ErrorContains(t, err, "field(s) [\"clientEmail\" \"domainAdminEmail\" \"privateKey\"] are only applicable to Google identity providers")
	})

	t.Run("missing required flags", func(t *testing.T) {
		err := Run(context.Background(), "providers", "add", "okta")
		assert.ErrorContains(t, err, "missing value for required flags: url, client-id, client-secret")
	})

	t.Run("list with json", func(t *testing.T) {
		setup(t)

		t.Setenv("INFRA_PROVIDER_URL", "https://okta.com/path")
		t.Setenv("INFRA_PROVIDER_CLIENT_ID", "okta-client-id")
		t.Setenv("INFRA_PROVIDER_CLIENT_SECRET", "okta-client-secret")

		err := Run(context.Background(), "providers", "add", "okta")
		assert.NilError(t, err)

		ctx, bufs := PatchCLI(context.Background())
		err = Run(ctx, "providers", "list", "--format=json")
		assert.NilError(t, err)

		golden.Assert(t, bufs.Stdout.String(), t.Name())
		assert.Assert(t, !strings.Contains(bufs.Stdout.String(), `count`))
		assert.Assert(t, !strings.Contains(bufs.Stdout.String(), `items`))
	})

	t.Run("list with yaml", func(t *testing.T) {
		setup(t)

		t.Setenv("INFRA_PROVIDER_URL", "https://okta.com/path")
		t.Setenv("INFRA_PROVIDER_CLIENT_ID", "okta-client-id")
		t.Setenv("INFRA_PROVIDER_CLIENT_SECRET", "okta-client-secret")

		err := Run(context.Background(), "providers", "add", "okta")
		assert.NilError(t, err)

		ctx, bufs := PatchCLI(context.Background())
		err = Run(ctx, "providers", "list", "--format=yaml")
		assert.NilError(t, err)

		golden.Assert(t, bufs.Stdout.String(), t.Name())
	})
}

func TestProvidersEditCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	var apiProviders []api.Provider
	apiProviders = append(apiProviders, api.Provider{
		Name:     "okta",
		URL:      "https://okta.com/path",
		ClientID: "okta-client-id",
		Kind:     "oidc",
	})
	apiProviders = append(apiProviders, api.Provider{
		Name:     "google",
		URL:      "https://example.com/google",
		ClientID: "google-client-id",
		Kind:     "google",
	})

	setup := func(t *testing.T) chan api.UpdateProviderRequest {
		requestCh := make(chan api.UpdateProviderRequest, 1)

		handler := func(resp http.ResponseWriter, req *http.Request) {
			if !strings.Contains(req.URL.Path, "/api/providers") {
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}

			switch req.Method {
			case http.MethodPut:
				var updateRequest api.UpdateProviderRequest
				err := json.NewDecoder(req.Body).Decode(&updateRequest)
				assert.Check(t, err)

				requestCh <- updateRequest

				_, _ = resp.Write([]byte(`{}`))
				return
			case http.MethodGet:
				name := req.URL.Query().Get("name")
				items := []api.Provider{}
				for _, p := range apiProviders {
					if p.Name == name {
						items = append(items, p)
					}
				}

				b, err := json.Marshal(api.ListResponse[api.Provider]{
					Items: items,
					Count: len(items),
				})
				assert.NilError(t, err)
				_, _ = resp.Write(b)
				return
			}
		}
		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.User{})
		err := writeConfig(&cfg)
		assert.NilError(t, err)
		return requestCh
	}

	t.Run("edit secret", func(t *testing.T) {
		ch := setup(t)

		err := Run(context.Background(),
			"providers", "edit", "okta",
			"--client-secret", "okta-client-secret",
		)
		assert.NilError(t, err)

		updateProviderRequest := <-ch

		expected := api.UpdateProviderRequest{
			Name:         "okta",
			URL:          "https://okta.com/path",
			ClientID:     "okta-client-id",
			ClientSecret: "okta-client-secret",
			Kind:         "oidc",
			API:          &api.ProviderAPICredentials{},
		}
		assert.DeepEqual(t, updateProviderRequest, expected)
	})

	t.Run("edit google api parameters", func(t *testing.T) {
		ch := setup(t)

		err := Run(context.Background(),
			"providers", "edit", "google",
			"--client-secret", "google-client-secret-2",
			"--service-account--key", "-----BEGIN PRIVATE KEY-----\naaa=\n-----END PRIVATE KEY-----\n",
			"--service-account--email", "example@tenant.iam.gserviceaccount.com",
			"--domain-admin", "admin@example.com",
		)
		assert.NilError(t, err)

		updateProviderRequest := <-ch

		expected := api.UpdateProviderRequest{
			Name:         "google",
			URL:          "https://example.com/google",
			ClientID:     "google-client-id",
			ClientSecret: "google-client-secret-2",
			Kind:         "google",
			API: &api.ProviderAPICredentials{
				PrivateKey:       "-----BEGIN PRIVATE KEY-----\naaa=\n-----END PRIVATE KEY-----\n",
				ClientEmail:      "example@tenant.iam.gserviceaccount.com",
				DomainAdminEmail: "admin@example.com",
			},
		}
		assert.DeepEqual(t, updateProviderRequest, expected)
	})

	t.Run("edit google without api parameters", func(t *testing.T) {
		ch := setup(t)

		err := Run(context.Background(),
			"providers", "edit", "google",
			"--client-secret", "google-client-secret-3",
		)
		assert.NilError(t, err)

		updateProviderRequest := <-ch

		expected := api.UpdateProviderRequest{
			Name:         "google",
			URL:          "https://example.com/google",
			ClientID:     "google-client-id",
			ClientSecret: "google-client-secret-3",
			Kind:         "google",
			API:          &api.ProviderAPICredentials{},
		}
		assert.DeepEqual(t, updateProviderRequest, expected)
	})

	t.Run("edit secret non-existing", func(t *testing.T) {
		_ = setup(t)

		t.Setenv("INFRA_PROVIDER_URL", "https://okta.com/path")
		t.Setenv("INFRA_PROVIDER_CLIENT_ID", "okta-client-id")
		t.Setenv("INFRA_PROVIDER_CLIENT_SECRET", "okta-client-secret")

		err := Run(context.Background(), "providers", "edit", "okta2", "--client-secret", "okta-client-secret")
		assert.ErrorContains(t, err, fmt.Sprintf("Provider %s does not exist", "okta2"))
	})

	t.Run("edit non-supported flag", func(t *testing.T) {
		_ = setup(t)
		err := Run(context.Background(), "providers", "edit", "okta", "--client-id", "okta-client-id")
		assert.ErrorContains(t, err, "unknown flag")
	})
}

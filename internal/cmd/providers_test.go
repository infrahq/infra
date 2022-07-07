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
		}
		assert.DeepEqual(t, createProviderRequest, expected)
	})

	t.Run("missing require flags", func(t *testing.T) {
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
				for _, p := range apiProviders {
					if p.Name == name {
						b, err := json.Marshal(api.ListResponse[api.Provider]{
							Items: apiProviders,
							Count: len(apiProviders),
						})
						assert.NilError(t, err)
						_, _ = resp.Write(b)
						return
					}
				}

				b, err := json.Marshal(api.ListResponse[api.Provider]{
					Count: 0,
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

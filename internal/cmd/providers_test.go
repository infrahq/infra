package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

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
		}
		assert.DeepEqual(t, createProviderRequest, expected)
	})

	t.Run("missing require flags", func(t *testing.T) {
		err := Run(context.Background(), "providers", "add", "okta")
		assert.ErrorContains(t, err, "missing value for required flags: url, client-id, client-secret")
	})

	t.Run("list with json", func(t *testing.T) {
		setup(t)
		ctx, bufs := PatchCLI(context.Background())

		t.Setenv("INFRA_PROVIDER_URL", "https://okta.com/path")
		t.Setenv("INFRA_PROVIDER_CLIENT_ID", "okta-client-id")
		t.Setenv("INFRA_PROVIDER_CLIENT_SECRET", "okta-client-secret")

		err := Run(ctx, "providers", "add", "okta")
		assert.NilError(t, err)

		err = Run(ctx, "providers", "list", "--format=json")
		assert.NilError(t, err)

		strings.Contains(bufs.Stdout.String(), `{"items":[{"id":"","name":"okta","created":null,"updated":null,"url":"https://okta.com/path","clientID":"okta-client-id"}],"count":1}`)
		assert.NilError(t, err)
	})
}

package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
			defer close(requestCh)
			if !requestMatches(req, http.MethodPost, "/v1/providers") {
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}

			var createRequest api.CreateProviderRequest
			err := json.NewDecoder(req.Body).Decode(&createRequest)
			assert.Check(t, err)

			requestCh <- createRequest

			_, _ = resp.Write([]byte(`{}`))
		}
		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.Identity{})
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
		)
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
}

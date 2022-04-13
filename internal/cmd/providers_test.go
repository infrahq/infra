package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

func TestProviders(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	id := uid.New()

	setup := func(t *testing.T, handler func(http.ResponseWriter, *http.Request)) {
		svc := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(svc.Close)

		cfg := ClientConfig{
			Version: "0.3",
			Hosts: []ClientHostConfig{
				{
					PolymorphicID: uid.NewIdentityPolymorphicID(id),
					Name:          "test",
					Host:          svc.Listener.Addr().String(),
					SkipTLSVerify: true,
					AccessKey:     "access-key",
					Expires:       api.Time(time.Now().Add(time.Hour)),
					Current:       true,
				},
			},
		}

		err := writeConfig(&cfg)
		assert.NilError(t, err)
	}

	t.Run("AddOktaProvider", func(t *testing.T) {
		setup(t, func(resp http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodPost && req.URL.Path == "/v1/providers" {
				var createProviderRequest api.CreateProviderRequest

				err := json.NewDecoder(req.Body).Decode(&createProviderRequest)
				assert.NilError(t, err)

				assert.Check(t, "okta" == createProviderRequest.Name)
				assert.Check(t, "okta-url" == createProviderRequest.URL)
				assert.Check(t, "okta-client-id" == createProviderRequest.ClientID)
				assert.Check(t, "okta-client-secret" == createProviderRequest.ClientSecret)

				provider := api.Provider{
					ID:       uid.New(),
					Name:     createProviderRequest.Name,
					Created:  api.Time(time.Now()),
					Updated:  api.Time(time.Now()),
					URL:      createProviderRequest.URL,
					ClientID: createProviderRequest.ClientID,
				}

				bytes, err := json.Marshal(&provider)
				assert.NilError(t, err)

				_, err = resp.Write(bytes)
				assert.NilError(t, err)
			}
		})

		err := Run(context.Background(), "providers", "add", "okta", "--url", "okta-url", "--client-id", "okta-client-id", "--client-secret", "okta-client-secret")
		assert.NilError(t, err)
	})
}

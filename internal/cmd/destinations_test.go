package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"

	"github.com/infrahq/infra/api"
)

func TestDestinationsListCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	setup := func(t *testing.T) chan api.ListDestinationsRequest {
		requestCh := make(chan api.ListDestinationsRequest, 1)

		handler := func(resp http.ResponseWriter, req *http.Request) {
			if !strings.Contains(req.URL.Path, "/api/destinations") || req.Method != http.MethodGet {
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}

			var apiDestinations []api.Destination
			apiDestinations = append(apiDestinations, api.Destination{
				Name: "destinationName",
				ID:   123,
			})

			b, err := json.Marshal(api.ListResponse[api.Destination]{
				Items: apiDestinations,
				Count: len(apiDestinations),
			})
			assert.NilError(t, err)
			_, _ = resp.Write(b)
		}

		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.User{})
		err := writeConfig(&cfg)
		assert.NilError(t, err)
		return requestCh
	}

	t.Run("list with json", func(t *testing.T) {
		setup(t)
		ctx, bufs := PatchCLI(context.Background())

		err := Run(ctx, "destinations", "list", "--format=json")
		assert.NilError(t, err)
		golden.Assert(t, bufs.Stdout.String(), t.Name())
	})

	t.Run("list with yaml", func(t *testing.T) {
		setup(t)
		ctx, bufs := PatchCLI(context.Background())

		err := Run(ctx, "destinations", "list", "--format=yaml")
		assert.NilError(t, err)
		golden.Assert(t, bufs.Stdout.String(), t.Name())
	})
}

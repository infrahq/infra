package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
)

func TestVersionCmd(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	t.Run("version no server", func(t *testing.T) {
		ctx, bufs := PatchCLI(context.Background())

		err := Run(ctx, "version")
		assert.NilError(t, err)

		intVersion := strings.TrimPrefix(internal.FullVersion(), "v")
		assert.Equal(t, bufs.Stdout.String(), fmt.Sprintf(expectedDisconnectedOutput, intVersion))
	})

	t.Run("version saas server", func(t *testing.T) {
		ctx, bufs := PatchCLI(context.Background())

		cfg := ClientConfig{
			ClientConfigVersion: clientConfigVersion,
			Hosts: []ClientHostConfig{
				{
					Name:      "user1",
					Host:      "awesome.infrahq.com",
					AccessKey: "something",
					UserID:    1,
					Current:   true,
					Expires:   api.Time(time.Now().Add(time.Hour * 2).UTC().Truncate(time.Second)),
				},
			},
		}

		err := writeConfig(&cfg)
		assert.NilError(t, err)

		err = Run(ctx, "version")
		assert.NilError(t, err)

		intVersion := strings.TrimPrefix(internal.FullVersion(), "v")
		assert.Equal(t, bufs.Stdout.String(), fmt.Sprintf(expectedSaasServerOutput, intVersion))
	})

	t.Run("version local server", func(t *testing.T) {
		ctx, bufs := PatchCLI(context.Background())

		intVersion := strings.TrimPrefix(internal.FullVersion(), "v")

		handler := func(resp http.ResponseWriter, req *http.Request) {
			if req.URL.Path != "/api/version" {
				resp.WriteHeader(http.StatusBadRequest)
				return
			}
			resp.WriteHeader(http.StatusOK)
			_, _ = resp.Write([]byte(fmt.Sprintf(`{"version": "%s"}`, intVersion)))
		}

		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := ClientConfig{
			ClientConfigVersion: clientConfigVersion,
			Hosts: []ClientHostConfig{
				{
					Name:          "user1",
					Host:          srv.Listener.Addr().String(),
					AccessKey:     "something",
					UserID:        1,
					Current:       true,
					SkipTLSVerify: true,
					Expires:       api.Time(time.Now().Add(time.Hour * 2).UTC().Truncate(time.Second)),
				},
			},
		}

		err := writeConfig(&cfg)
		assert.NilError(t, err)

		err = Run(ctx, "version")
		assert.NilError(t, err)

		assert.Equal(t, bufs.Stdout.String(), fmt.Sprintf(expectedLocalServerOutput, intVersion, intVersion))
	})
}

var expectedDisconnectedOutput = `
 Client: %s
 Server: disconnected

`

var expectedSaasServerOutput = `
 Client: %s

`

var expectedLocalServerOutput = `
 Client: %s
 Server: %s

`

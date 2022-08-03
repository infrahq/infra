package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/infrahq/infra/api"
	"gotest.tools/v3/assert"
)

func TestGroupAddCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	setup := func(t *testing.T) chan api.CreateGroupRequest {
		requestCh := make(chan api.CreateGroupRequest, 1)
		handler := func(resp http.ResponseWriter, req *http.Request) {
			if !requestMatches(req, http.MethodPost, "/api/groups") {
				resp.WriteHeader(http.StatusBadRequest)
				return
			}

			defer close(requestCh)
			var createRequest api.CreateGroupRequest
			err := json.NewDecoder(req.Body).Decode(&createRequest)
			assert.NilError(t, err)

			resp.WriteHeader(http.StatusOK)
			err = json.NewEncoder(resp).Encode(&api.Group{})
			assert.NilError(t, err)

			requestCh <- createRequest
		}
		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.User{})
		err := writeConfig(&cfg)
		assert.NilError(t, err)

		return requestCh

	}

	t.Run("create group", func(t *testing.T) {
		ch := setup(t)
		ctx, bufs := PatchCLI(context.Background())

		err := Run(ctx, "groups", "add", "Test")
		assert.NilError(t, err)
		req := <-ch
		expected := api.CreateGroupRequest{Name: "Test"}

		assert.DeepEqual(t, expected, req)
		assert.Equal(t, bufs.Stdout.String(), expectedGroupsAddOutput)
	})

	t.Run("without argument", func(t *testing.T) {
		err := Run(context.Background(), "groups", "add")
		assert.ErrorContains(t, err, `"infra groups add" requires exactly 1 argument.`)
		assert.ErrorContains(t, err, `Usage:  infra groups add GROUP [flags]`)
	})
}

var expectedGroupsAddOutput = `Added group "Test"
`

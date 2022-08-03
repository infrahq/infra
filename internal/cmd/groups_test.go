package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"

	"github.com/infrahq/infra/api"
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestGroupsListCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	setup := func(t *testing.T) {
		handler := func(resp http.ResponseWriter, req *http.Request) {
			if requestMatches(req, http.MethodGet, "/api/users") {
				if req.URL.Query().Get("group") == "2J" {
					limit, err := strconv.Atoi(string(req.URL.Query().Get("limit")))
					assert.NilError(t, err)

					if limit == 0 {
						limit = 100
					}
					resp.WriteHeader(http.StatusOK)
					users := []api.User{
						{Name: "a@example.com"},
						{Name: "b@example.com"},
						{Name: "c@example.com"},
						{Name: "d@example.com"},
						{Name: "e@example.com"},
						{Name: "f@example.com"},
						{Name: "g@example.com"},
						{Name: "h@example.com"},
						{Name: "i@example.com"}}

					err = json.NewEncoder(resp).Encode(api.ListResponse[api.User]{Items: users[:min(limit, len(users))]})
					assert.NilError(t, err)
					return
				}
			}
			if !requestMatches(req, http.MethodGet, "/api/groups") {
				resp.WriteHeader(http.StatusTeapot)
				return
			}

			resp.WriteHeader(http.StatusOK)
			err := json.NewEncoder(resp).Encode(&api.ListResponse[api.Group]{Items: []api.Group{
				{Name: "Test", ID: 100, TotalUsers: 9},
			}})
			assert.NilError(t, err)
		}

		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.User{})
		err := writeConfig(&cfg)
		assert.NilError(t, err)
	}

	t.Run("list groups default", func(t *testing.T) {
		setup(t)
		ctx, bufs := PatchCLI(context.Background())
		err := Run(ctx, "groups", "list")
		assert.NilError(t, err)

		golden.Assert(t, bufs.Stdout.String(), t.Name())
	})

	t.Run("no truncate", func(t *testing.T) {
		setup(t)
		ctx, bufs := PatchCLI(context.Background())
		err := Run(ctx, "groups", "list", "--no-truncate")
		assert.NilError(t, err)

		golden.Assert(t, bufs.Stdout.String(), t.Name())
	})

	t.Run("num users", func(t *testing.T) {
		setup(t)
		ctx, bufs := PatchCLI(context.Background())
		err := Run(ctx, "groups", "list", "--num-users", "2")
		assert.NilError(t, err)

		golden.Assert(t, bufs.Stdout.String(), t.Name())
	})
}

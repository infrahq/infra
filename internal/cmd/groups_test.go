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

func TestGroupsAddCmd(t *testing.T) {
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

func TestGroupsRemoveCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	setup := func(t *testing.T) {
		handler := func(resp http.ResponseWriter, req *http.Request) {
			listResp := api.ListResponse[api.Group]{}

			if requestMatches(req, http.MethodGet, "/api/groups") {
				if req.URL.Query().Get("name") == "Test" {
					listResp = api.ListResponse[api.Group]{Count: 1, Items: []api.Group{{ID: 100, Name: "Test"}}}
				}
				resp.WriteHeader(http.StatusOK)
				err := json.NewEncoder(resp).Encode(listResp)
				assert.NilError(t, err)
				return

			}

			if !requestMatchesPrefix(req, http.MethodDelete, "/api/groups") {
				resp.WriteHeader(http.StatusBadRequest)
				return
			}

			resp.WriteHeader(http.StatusOK)
			err := json.NewEncoder(resp).Encode(map[string]string{})
			assert.NilError(t, err)
		}
		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.User{})
		err := writeConfig(&cfg)
		assert.NilError(t, err)

	}

	t.Run("remove group", func(t *testing.T) {
		setup(t)
		ctx, bufs := PatchCLI(context.Background())
		err := Run(ctx, "groups", "remove", "Test")
		assert.NilError(t, err)
		assert.Equal(t, bufs.Stdout.String(), `Removed group "Test"`+"\n")
	})

	t.Run("remove group unknown", func(t *testing.T) {
		setup(t)
		ctx, _ := PatchCLI(context.Background())
		err := Run(ctx, "groups", "remove", "Nonexistent")
		assert.ErrorContains(t, err, `unknown group "Nonexistent"`)
	})

	t.Run("remove group force", func(t *testing.T) {
		setup(t)
		ctx, bufs := PatchCLI(context.Background())
		err := Run(ctx, "groups", "remove", "Nonexistent", "--force")
		assert.NilError(t, err)
		assert.Equal(t, bufs.Stdout.String(), "")
	})

	t.Run("without argument", func(t *testing.T) {
		err := Run(context.Background(), "groups", "remove")
		assert.ErrorContains(t, err, `"infra groups remove" requires exactly 1 argument.`)
		assert.ErrorContains(t, err, `Usage:  infra groups remove GROUP [flags]`)
	})
}

func TestGroupsAddAndRemoveUserCmds(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	setup := func(t *testing.T) {
		handler := func(resp http.ResponseWriter, req *http.Request) {

			if requestMatches(req, http.MethodGet, "/api/groups") {
				resp.WriteHeader(http.StatusOK)
				if req.URL.Query().Get("name") == "Test" {
					err := json.NewEncoder(resp).Encode(api.ListResponse[api.Group]{Count: 1, Items: []api.Group{{ID: 100, Name: "Test"}}})
					assert.NilError(t, err)
				} else {
					err := json.NewEncoder(resp).Encode(api.ListResponse[api.Group]{Count: 0, Items: []api.Group{}})
					assert.NilError(t, err)
				}
				return
			}

			if requestMatches(req, http.MethodGet, "/api/users") {
				resp.WriteHeader(http.StatusOK)
				if req.URL.Query().Get("name") == "user@example.com" {
					err := json.NewEncoder(resp).Encode(api.ListResponse[api.User]{Count: 1, Items: []api.User{{ID: 1, Name: "user@example.com"}}})
					assert.NilError(t, err)
				} else {
					err := json.NewEncoder(resp).Encode(api.ListResponse[api.User]{Count: 0, Items: []api.User{}})
					assert.NilError(t, err)
				}
				return
			}
			if !requestMatches(req, http.MethodPatch, "/api/groups/2J/users") {
				resp.WriteHeader(http.StatusBadRequest)
				return
			}

			var updateRequest api.UpdateUsersInGroupRequest
			err := json.NewDecoder(req.Body).Decode(&updateRequest)
			assert.NilError(t, err)

			if (len(updateRequest.UserIDsToAdd) > 0 && updateRequest.UserIDsToAdd[0] == 1) || (len(updateRequest.UserIDsToRemove) > 0 && updateRequest.UserIDsToRemove[0] == 1) {
				resp.WriteHeader(http.StatusOK)
				err = json.NewEncoder(resp).Encode(map[string]string{})
				assert.NilError(t, err)
				return
			}

		}
		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.User{})
		err := writeConfig(&cfg)
		assert.NilError(t, err)

	}

	t.Run("add user", func(t *testing.T) {
		setup(t)
		ctx, bufs := PatchCLI(context.Background())
		err := Run(ctx, "groups", "adduser", "user@example.com", "Test")
		assert.NilError(t, err)
		assert.Equal(t, bufs.Stdout.String(), `Added user "user@example.com" to group "Test"`+"\n")
	})

	t.Run("remove user", func(t *testing.T) {
		setup(t)
		ctx, bufs := PatchCLI(context.Background())
		err := Run(ctx, "groups", "removeuser", "user@example.com", "Test")
		assert.NilError(t, err)
		assert.Equal(t, bufs.Stdout.String(), `Removed user "user@example.com" from group "Test"`+"\n")
	})

	t.Run("remove user unknown", func(t *testing.T) {
		setup(t)
		ctx, _ := PatchCLI(context.Background())
		err := Run(ctx, "groups", "removeuser", "unknown@example.com", "Test")
		assert.ErrorContains(t, err, `unknown user "unknown@example.com"`)
	})

	t.Run("add user unknown", func(t *testing.T) {
		setup(t)
		ctx, _ := PatchCLI(context.Background())
		err := Run(ctx, "groups", "adduser", "unknown@example.com", "Test")
		assert.ErrorContains(t, err, `unknown user "unknown@example.com"`)
	})

	t.Run("group unknown add", func(t *testing.T) {
		setup(t)
		ctx, _ := PatchCLI(context.Background())
		err := Run(ctx, "groups", "adduser", "user@example.com", "Nonexistent")
		assert.ErrorContains(t, err, `unknown group "Nonexistent"`)
	})

	t.Run("group unknown remove", func(t *testing.T) {
		setup(t)
		ctx, _ := PatchCLI(context.Background())
		err := Run(ctx, "groups", "removeuser", "user@example.com", "Nonexistent")
		assert.ErrorContains(t, err, `unknown group "Nonexistent"`)
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

package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
)

func TestGrantsAddCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // for windows

	setup := func(t *testing.T) chan api.GrantRequest {
		requestCh := make(chan api.GrantRequest, 1)

		handler := func(resp http.ResponseWriter, req *http.Request) {
			query := req.URL.Query()

			if requestMatches(req, http.MethodGet, "/api/users") {
				resp.WriteHeader(http.StatusOK)
				switch query.Get("name") {
				case "existing@example.com":
					writeResponse(t, resp, api.ListResponse[api.User]{Count: 1, Items: []api.User{{ID: 3000}}})
				case "existingMachine":
					writeResponse(t, resp, api.ListResponse[api.User]{Count: 1, Items: []api.User{{ID: 3001}}})
				default:
					writeResponse(t, resp, &api.ListResponse[api.User]{})
				}
				return
			} else if requestMatches(req, http.MethodPost, "/api/users") {
				resp.WriteHeader(http.StatusOK)
				writeResponse(t, resp, &api.CreateUserResponse{ID: 3002})
				return
			}

			if requestMatches(req, http.MethodGet, "/api/groups") {
				resp.WriteHeader(http.StatusOK)
				if query.Get("name") == "existingGroup" {
					writeResponse(t, resp, api.ListResponse[api.Group]{Count: 1, Items: []api.Group{{ID: 4000}}})
					return
				}
				writeResponse(t, resp, &api.ListResponse[api.Group]{})
				return
			} else if requestMatches(req, http.MethodPost, "/api/groups") {
				resp.WriteHeader(http.StatusOK)
				writeResponse(t, resp, &api.Group{ID: 4001})
				return
			}

			if requestMatches(req, http.MethodGet, "/api/destinations") {
				resp.WriteHeader(http.StatusOK)
				if query.Get("name") == "the-destination" {
					writeResponse(t, resp, api.ListResponse[api.Destination]{Count: 1, Items: []api.Destination{{ID: 5000, Roles: []string{"role"}, Resources: []string{"default"}}}})
					return
				}
				writeResponse(t, resp, &api.ListResponse[api.Destination]{})
				return
			}

			if !requestMatches(req, http.MethodPost, "/api/grants") {
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}

			defer close(requestCh)
			var createReq api.GrantRequest
			err := json.NewDecoder(req.Body).Decode(&createReq)
			assert.Check(t, err)

			requestCh <- createReq
			writeResponse(t, resp, api.Grant{ID: 7000})
		}
		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.User{})
		err := writeConfig(&cfg)
		assert.NilError(t, err)
		return requestCh
	}

	t.Run("add default role to existing identity", func(t *testing.T) {
		ch := setup(t)
		ctx := context.Background()
		err := Run(ctx, "grants", "add", "existing@example.com", "the-destination")
		assert.NilError(t, err)

		createReq := <-ch
		expected := api.GrantRequest{
			User:      3000,
			Privilege: "connect",
			Resource:  "the-destination",
		}
		assert.DeepEqual(t, createReq, expected)
	})
	t.Run("add default role to existing identity for namespace", func(t *testing.T) {
		ch := setup(t)
		ctx := context.Background()
		err := Run(ctx, "grants", "add", "existing@example.com", "the-destination.default")
		assert.NilError(t, err)

		createReq := <-ch
		expected := api.GrantRequest{
			User:      3000,
			Privilege: "connect",
			Resource:  "the-destination.default",
		}
		assert.DeepEqual(t, createReq, expected)
	})
	t.Run("add role to existing identity", func(t *testing.T) {
		ch := setup(t)
		ctx := context.Background()
		err := Run(ctx, "grants", "add", "existing@example.com", "the-destination", "--role", "role")
		assert.NilError(t, err)

		createReq := <-ch
		expected := api.GrantRequest{
			User:      3000,
			Privilege: "role",
			Resource:  "the-destination",
		}
		assert.DeepEqual(t, createReq, expected)
	})
	t.Run("add role to existing group", func(t *testing.T) {
		ch := setup(t)
		ctx := context.Background()
		err := Run(ctx,
			"grants", "add", "existingGroup", "the-destination",
			"--group", "--role", "role")
		assert.NilError(t, err)

		createReq := <-ch
		expected := api.GrantRequest{
			Group:     4000,
			Privilege: "role",
			Resource:  "the-destination",
		}
		assert.DeepEqual(t, createReq, expected)
	})

	t.Run("add grant for nonexistent user", func(t *testing.T) {
		_ = setup(t)
		err := Run(context.Background(), "grants", "add", "nonexistent", "destination")
		assert.ErrorContains(t, err, "unknown user")
	})

	t.Run("add grant for nonexistent group", func(t *testing.T) {
		_ = setup(t)
		err := Run(context.Background(), "grants", "add", "nonexistent", "destination", "--group")
		assert.ErrorContains(t, err, "unknown group")
	})

	t.Run("force add grant for nonexistent user", func(t *testing.T) {
		ch := setup(t)
		err := Run(context.Background(), "grants", "add", "nonexistent", "destination", "--force")
		assert.NilError(t, err)

		actual := <-ch
		expected := api.GrantRequest{
			User:      3002,
			Privilege: "connect",
			Resource:  "destination",
		}

		assert.DeepEqual(t, actual, expected)
	})

	t.Run("force add grant for nonexistent group", func(t *testing.T) {
		ch := setup(t)
		err := Run(context.Background(), "grants", "add", "nonexistent", "destination", "--group", "--force")
		assert.NilError(t, err)

		actual := <-ch
		expected := api.GrantRequest{
			Group:     4001,
			Privilege: "connect",
			Resource:  "destination",
		}

		assert.DeepEqual(t, actual, expected)
	})

	t.Run("add role to non-existent destination", func(t *testing.T) {
		_ = setup(t)
		ctx := context.Background()
		err := Run(ctx, "grants", "add", "existing@example.com", "nonexistent")
		assert.ErrorContains(t, err, "not connected")
	})

	t.Run("add role to non-existent namespace", func(t *testing.T) {
		_ = setup(t)
		ctx := context.Background()
		err := Run(ctx, "grants", "add", "existing@example.com", "the-destination.nonexistent")
		assert.ErrorContains(t, err, "not detected in destination")
	})

	t.Run("add role to non-existent destination", func(t *testing.T) {
		_ = setup(t)
		ctx := context.Background()
		err := Run(ctx, "grants", "add", "existing@example.com", "the-destination", "--role", "nonexistent")
		assert.ErrorContains(t, err, "not a known role")
	})

	t.Run("force add grant for nonexistent destination", func(t *testing.T) {
		ch := setup(t)
		err := Run(context.Background(), "grants", "add", "existing@example.com", "nonexistent", "--force")
		assert.NilError(t, err)

		actual := <-ch
		expected := api.GrantRequest{
			User:      3000,
			Privilege: "connect",
			Resource:  "nonexistent",
		}

		assert.DeepEqual(t, actual, expected)
	})

	t.Run("force add grant for nonexistent namespace", func(t *testing.T) {
		ch := setup(t)
		err := Run(context.Background(), "grants", "add", "existing@example.com", "the-destination.nonexistent", "--force")
		assert.NilError(t, err)

		actual := <-ch
		expected := api.GrantRequest{
			User:      3000,
			Privilege: "connect",
			Resource:  "the-destination.nonexistent",
		}

		assert.DeepEqual(t, actual, expected)
	})

	t.Run("force add grant for nonexistent role", func(t *testing.T) {
		ch := setup(t)
		err := Run(context.Background(), "grants", "add", "existing@example.com", "the-destination", "--role", "nonexistent", "--force")
		assert.NilError(t, err)

		actual := <-ch
		expected := api.GrantRequest{
			User:      3000,
			Privilege: "nonexistent",
			Resource:  "the-destination",
		}

		assert.DeepEqual(t, actual, expected)
	})
}

func writeResponse(t *testing.T, resp io.Writer, body interface{}) {
	t.Helper()
	err := json.NewEncoder(resp).Encode(body)
	assert.Check(t, err, "failed to write API response")
}

func TestGrantRemoveCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // for windows

	setup := func(t *testing.T) chan string {
		requestCh := make(chan string, 5)

		handler := func(resp http.ResponseWriter, req *http.Request) {
			query := req.URL.Query()

			if requestMatches(req, http.MethodGet, "/api/users") {
				resp.WriteHeader(http.StatusOK)
				if query.Get("name") == "existing@example.com" {
					writeResponse(t, resp, api.ListResponse[api.User]{Count: 1, Items: []api.User{{ID: 3000}}})
				} else {
					writeResponse(t, resp, []api.User{})
				}
				return
			}

			if requestMatches(req, http.MethodGet, "/api/groups") {
				resp.WriteHeader(http.StatusOK)
				if query.Get("name") == "existingGroup" {
					writeResponse(t, resp, api.ListResponse[api.Group]{Count: 1, Items: []api.Group{{ID: 4000}}})
					return
				}
				writeResponse(t, resp, &api.ListResponse[api.Group]{})
				return
			}

			if requestMatches(req, http.MethodGet, "/api/grants") {
				resp.WriteHeader(http.StatusOK)
				if query.Get("resource") != "the-destination" {
					writeResponse(t, resp, api.ListResponse[api.Grant]{})
					return
				}

				if query.Get("privilege") == "custom" {
					if query.Get("user") == "TK" { // ID=3001
						writeResponse(t, resp, api.ListResponse[api.Grant]{Count: 1, Items: []api.Grant{{ID: 6001}, {ID: 6002}}})
						return
					}

					if query.Get("group") == "2bY" { // ID=4000
						writeResponse(t, resp, api.ListResponse[api.Grant]{Count: 1, Items: []api.Grant{{ID: 9001}, {ID: 9002}}})
						return
					}

					writeResponse(t, resp, api.ListResponse[api.Grant]{})
					return
				}

				if query.Get("privilege") != "" {
					writeResponse(t, resp, api.ListResponse[api.Grant]{})
					return
				}

				if query.Get("user") == "TJ" { // ID=3000
					writeResponse(t, resp, api.ListResponse[api.Grant]{Count: 1, Items: []api.Grant{{ID: 5001}, {ID: 5002}, {ID: 5003}}})
					return
				}

				if query.Get("group") == "2bY" { // ID=4000
					writeResponse(t, resp, api.ListResponse[api.Grant]{Count: 1, Items: []api.Grant{{ID: 7001}, {ID: 7002}}})
					return
				}

				writeResponse(t, resp, api.ListResponse[api.Grant]{})
				return
			}

			if requestMatchesPrefix(req, http.MethodDelete, "/api/grants") {
				resp.WriteHeader(http.StatusOK)

				requestCh <- path.Base(req.URL.Path)
				writeResponse(t, resp, map[string]interface{}{})
				return
			}

			resp.WriteHeader(http.StatusInternalServerError)
		}
		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.User{})
		err := writeConfig(&cfg)
		assert.NilError(t, err)
		return requestCh
	}

	t.Run("remove default grants from identity", func(t *testing.T) {
		ch := setup(t)
		ctx := context.Background()
		err := Run(ctx, "grants", "remove", "existing@example.com", "the-destination")
		assert.NilError(t, err)

		reqIDs := readChan(ch)
		expected := []string{"2ue", "2uf", "2ug"}
		assert.DeepEqual(t, reqIDs, expected)
	})
	t.Run("remove grant from group", func(t *testing.T) {
		ch := setup(t)
		ctx := context.Background()
		err := Run(ctx,
			"grants", "remove", "existingGroup", "the-destination",
			"--group", "--role", "custom")
		assert.NilError(t, err)

		reqIDs := readChan(ch)
		expected := []string{"3Fc", "3Fd"}
		assert.DeepEqual(t, reqIDs, expected)
	})
}

// readChan reads ch until there are no more buffered items
func readChan(ch chan string) []string {
	var items []string
	for {
		select {
		case item := <-ch:
			items = append(items, item)
		default:
			return items
		}
	}
}

func requestMatchesPrefix(req *http.Request, method string, path string) bool {
	return req.Method == method && strings.HasPrefix(req.URL.Path, path)
}

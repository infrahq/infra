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

	setup := func(t *testing.T) chan api.CreateGrantRequest {
		requestCh := make(chan api.CreateGrantRequest, 1)

		handler := func(resp http.ResponseWriter, req *http.Request) {
			query := req.URL.Query()

			if requestMatches(req, http.MethodGet, "/v1/identities") {
				resp.WriteHeader(http.StatusOK)
				switch query.Get("name") {
				case "existing@example.com":
					writeResponse(t, resp, []api.Identity{{ID: 3000}})
				case "existingMachine":
					writeResponse(t, resp, []api.Identity{{ID: 3001}})
				default:
					writeResponse(t, resp, map[string]interface{}{})
				}
				return
			}

			if requestMatches(req, http.MethodGet, "/v1/groups") {
				resp.WriteHeader(http.StatusOK)
				if query.Get("name") == "existingGroup" {
					writeResponse(t, resp, []api.Group{{ID: 4000}})
					return
				}
				writeResponse(t, resp, map[string]interface{}{})
				return
			}

			if !requestMatches(req, http.MethodPost, "/v1/grants") {
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}

			defer close(requestCh)
			var createReq api.CreateGrantRequest
			err := json.NewDecoder(req.Body).Decode(&createReq)
			assert.Check(t, err)

			requestCh <- createReq
			writeResponse(t, resp, api.Grant{ID: 7000})
		}
		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.Identity{})
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
		expected := api.CreateGrantRequest{
			Subject:   "i:TJ",
			Privilege: "connect",
			Resource:  "the-destination",
		}
		assert.DeepEqual(t, createReq, expected)
	})
	t.Run("add role to existing identity", func(t *testing.T) {
		ch := setup(t)
		ctx := context.Background()
		err := Run(ctx, "grants", "add", "existing@example.com", "the-destination", "--role", "role")
		assert.NilError(t, err)

		createReq := <-ch
		expected := api.CreateGrantRequest{
			Subject:   "i:TJ",
			Privilege: "role",
			Resource:  "the-destination",
		}
		assert.DeepEqual(t, createReq, expected)
	})
	t.Run("add role to existing machine identity", func(t *testing.T) {
		ch := setup(t)
		ctx := context.Background()
		err := Run(ctx, "grants", "add", "existingMachine", "the-destination", "--role", "role")
		assert.NilError(t, err)

		createReq := <-ch
		expected := api.CreateGrantRequest{
			Subject:   "i:TK",
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
		expected := api.CreateGrantRequest{
			Subject:   "g:2bY",
			Privilege: "role",
			Resource:  "the-destination",
		}
		assert.DeepEqual(t, createReq, expected)
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

			if requestMatches(req, http.MethodGet, "/v1/identities") {
				resp.WriteHeader(http.StatusOK)
				switch query.Get("name") {
				case "existing@example.com":
					writeResponse(t, resp, []api.Identity{{ID: 3000}})
				case "existingMachine":
					writeResponse(t, resp, []api.Identity{{ID: 3001}})
				default:
					writeResponse(t, resp, []api.Identity{})
				}
				return
			}

			if requestMatches(req, http.MethodGet, "/v1/groups") {
				resp.WriteHeader(http.StatusOK)
				if query.Get("name") == "existingGroup" {
					writeResponse(t, resp, []api.Group{{ID: 4000}})
					return
				}
				writeResponse(t, resp, map[string]interface{}{})
				return
			}

			if requestMatches(req, http.MethodGet, "/v1/grants") {
				resp.WriteHeader(http.StatusOK)
				if query.Get("resource") != "the-destination" {
					writeResponse(t, resp, []api.Grant{})
					return
				}

				if query.Get("privilege") == "custom" {
					switch query.Get("subject") {
					case "i:TK": // ID=3001
						writeResponse(t, resp, []api.Grant{{ID: 6001}, {ID: 6002}})
					case "g:2bY": // ID=4000
						writeResponse(t, resp, []api.Grant{{ID: 9001}, {ID: 9002}})
					default:
						writeResponse(t, resp, []api.Grant{})
					}
					return
				}

				if query.Get("privilege") != "" {
					writeResponse(t, resp, []api.Grant{})
					return
				}

				switch query.Get("subject") {
				case "i:TJ": // ID=3000
					writeResponse(t, resp, []api.Grant{{ID: 5001}, {ID: 5002}, {ID: 5003}})
				case "g:2bY": // ID=4000
					writeResponse(t, resp, []api.Grant{{ID: 7001}, {ID: 7002}})
				default:
					writeResponse(t, resp, []api.Grant{})
				}
				return
			}

			if requestMatchesPrefix(req, http.MethodDelete, "/v1/grants") {
				resp.WriteHeader(http.StatusOK)

				requestCh <- path.Base(req.URL.Path)
				writeResponse(t, resp, map[string]interface{}{})
				return
			}

			resp.WriteHeader(http.StatusInternalServerError)
		}
		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.Identity{})
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
	t.Run("remove grant from identity", func(t *testing.T) {
		ch := setup(t)
		ctx := context.Background()
		err := Run(ctx, "grants", "remove", "existingMachine", "the-destination", "--role", "custom")
		assert.NilError(t, err)

		reqIDs := readChan(ch)
		expected := []string{"2Mt", "2Mu"}
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

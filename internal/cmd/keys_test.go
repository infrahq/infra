package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

func TestKeysAddCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // for windows

	setup := func(t *testing.T) chan api.CreateAccessKeyRequest {
		requestCh := make(chan api.CreateAccessKeyRequest, 1)

		handler := func(resp http.ResponseWriter, req *http.Request) {
			// the command does a lookup for user ID
			if requestMatches(req, http.MethodGet, "/api/users") {
				if req.URL.Query().Get("name") != "my-user" {
					resp.WriteHeader(http.StatusBadRequest)
					return
				}
				resp.WriteHeader(http.StatusOK)
				err := json.NewEncoder(resp).Encode(api.ListResponse[api.User]{
					Count: 1,
					Items: []api.User{
						{ID: uid.ID(12345678)},
					},
				})
				assert.Check(t, err)
				return
			}

			if !requestMatches(req, http.MethodPost, "/api/access-keys") {
				resp.WriteHeader(http.StatusBadRequest)
				return
			}

			defer close(requestCh)
			var createRequest api.CreateAccessKeyRequest
			err := json.NewDecoder(req.Body).Decode(&createRequest)
			assert.Check(t, err)

			resp.WriteHeader(http.StatusOK)
			err = json.NewEncoder(resp).Encode(&api.CreateAccessKeyResponse{
				AccessKey: "the-access-key",
				Name:      "the-key-name",
			})
			assert.Check(t, err)
			requestCh <- createRequest
		}

		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.User{})
		err := writeConfig(&cfg)
		assert.NilError(t, err)

		return requestCh
	}

	t.Run("all flags", func(t *testing.T) {
		ch := setup(t)

		ctx, bufs := PatchCLI(context.Background())
		err := Run(ctx, "keys", "add", "--expiry=400h", "--inactivity-timeout=5h", "--name=the-name", "--user=my-user")
		assert.NilError(t, err)

		req := <-ch
		expected := api.CreateAccessKeyRequest{
			UserID:            uid.ID(12345678),
			Name:              "the-name",
			Expiry:            api.Duration(400 * time.Hour),
			InactivityTimeout: api.Duration(5 * time.Hour),
		}
		assert.DeepEqual(t, expected, req)
		assert.Equal(t, withNewline(bufs.Stdout.String()), expectedKeysAddOutput)
	})

	t.Run("automatic name", func(t *testing.T) {
		ch := setup(t)

		ctx, bufs := PatchCLI(context.Background())
		err := Run(ctx, "keys", "add", "--expiry=400h", "--inactivity-timeout=5h", "--user=my-user")
		assert.NilError(t, err)

		req := <-ch
		assert.Equal(t, req.Name, "") // filled by server
		assert.Equal(t, withNewline(bufs.Stdout.String()), expectedKeysAddOutput)
	})

	t.Run("with unexpected arguments", func(t *testing.T) {
		err := Run(context.Background(), "keys", "add", "something")
		assert.ErrorContains(t, err, `"infra keys add" accepts no arguments`)
		assert.ErrorContains(t, err, `Usage:  infra keys add`)
	})
}

// expectedKeysAddOutput can be updated automatically by running tests with -update
var expectedKeysAddOutput = `
Issued access key "the-key-name" for "my-user"
This key will expire in 400 hours

Key: the-access-key
`

// withNewline adds a preceding newline so that expected output that is managed
// in golden variables formats nicely.
func withNewline(v string) string {
	return "\n" + v
}

func requestMatches(req *http.Request, method string, path string) bool {
	return req.Method == method && req.URL.Path == path
}

func TestKeysListCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // for windows

	base := time.Now().Add(-24 * time.Hour)

	setup := func(t *testing.T) {
		handler := func(resp http.ResponseWriter, req *http.Request) {
			query := req.URL.Query()

			// the command does a lookup for user ID
			if requestMatches(req, http.MethodGet, "/api/users") {
				if query.Get("name") != "my-user" {
					resp.WriteHeader(http.StatusBadRequest)
					return
				}
				resp.WriteHeader(http.StatusOK)
				err := json.NewEncoder(resp).Encode(api.ListResponse[api.User]{
					Count: 1,
					Items: []api.User{
						{ID: uid.ID(12345678)},
					},
				})
				assert.Check(t, err)
				return
			}

			if !requestMatches(req, http.MethodGet, "/api/access-keys") {
				resp.WriteHeader(http.StatusBadRequest)
				return
			}

			resp.WriteHeader(http.StatusOK)
			if query.Get("userID") == uid.ID(12345678).String() {
				err := json.NewEncoder(resp).Encode(api.ListResponse[api.AccessKey]{
					Count: 1,
					Items: []api.AccessKey{
						{
							Name:          "user-key",
							IssuedFor:     uid.ID(12345678),
							IssuedForName: "my-user",
							Created:       api.Time(base.Add(5 * time.Minute)),
							Expires:       api.Time(base.Add(30 * time.Hour)),
						},
					},
				})
				assert.Check(t, err)
				return
			}
			err := json.NewEncoder(resp).Encode(api.ListResponse[api.AccessKey]{
				Count: 3,
				Items: []api.AccessKey{
					{
						Name:          "front-door",
						IssuedFor:     uid.ID(12345),
						IssuedForName: "admin",
						Created:       api.Time(base.Add(time.Minute)),
					},
					{
						Name:              "side-door",
						IssuedFor:         uid.ID(12345),
						IssuedForName:     "admin",
						Created:           api.Time(base.Add(time.Minute)),
						Expires:           api.Time(base.Add(30 * time.Hour)),
						InactivityTimeout: api.Time(base.Add(50 * time.Hour)),
					},
					{
						Name:          "storage",
						IssuedFor:     uid.ID(12349),
						IssuedForName: "clerk",
						Created:       api.Time(base.Add(4 * time.Hour)),
						Expires:       api.Time(base.Add(30 * time.Hour)),
					},
				},
			})
			assert.Check(t, err)
		}

		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.User{})
		err := writeConfig(&cfg)
		assert.NilError(t, err)
	}

	t.Run("list all", func(t *testing.T) {
		setup(t)
		ctx, bufs := PatchCLI(context.Background())

		err := Run(ctx, "keys", "list")
		assert.NilError(t, err)

		golden.Assert(t, bufs.Stdout.String(), t.Name())
	})

	t.Run("filter by user name", func(t *testing.T) {
		setup(t)
		ctx, bufs := PatchCLI(context.Background())

		err := Run(ctx, "keys", "list", "--user", "my-user")
		assert.NilError(t, err)

		golden.Assert(t, bufs.Stdout.String(), t.Name())
	})
}

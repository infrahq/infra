package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestUsersCmd(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir) // for windows

	providerIDs := []uid.ID{123}
	providerIdx := 0
	userIDs := []uid.ID{12, 23, 34, 45, 56}
	userIdx := 0

	setup := func(t *testing.T) *[]models.Identity {
		modifiedUsers := []models.Identity{}

		handler := func(resp http.ResponseWriter, req *http.Request) {
			if strings.HasPrefix(req.URL.Path, "/api/providers") {
				resp.WriteHeader(http.StatusOK)

				providers := []*api.Provider{
					{
						Name: "infra",
						ID:   providerIDs[providerIdx],
					},
				}
				providerIdx++
				b, err := json.Marshal(providers)
				assert.NilError(t, err)
				_, _ = resp.Write(b)
				return
			}

			if strings.HasPrefix(req.URL.Path, "/api/users") {
				switch req.Method {
				case http.MethodPost:
					createUserReq := api.CreateUserRequest{}

					err := json.NewDecoder(req.Body).Decode(&createUserReq)
					assert.NilError(t, err)

					newUser := models.Identity{
						Name: createUserReq.Name,
					}
					newUser.ID = userIDs[userIdx]
					userIdx++

					respBody := api.CreateUserResponse{
						ID:   newUser.ID,
						Name: newUser.Name,
					}
					modifiedUsers = append(modifiedUsers, newUser)

					b, err := json.Marshal(&respBody)
					assert.NilError(t, err)
					_, _ = resp.Write(b)
					return
				case http.MethodGet:
					name := req.URL.Query().Get("name")

					var apiUsers []api.User
					for _, mu := range modifiedUsers {
						if mu.Name == name || name == "" {
							apiUsers = append(apiUsers, *mu.ToAPI())
						}
					}
					b, err := json.Marshal(api.ListResponse[api.User]{
						Items: apiUsers,
						Count: len(apiUsers),
					})
					assert.NilError(t, err)
					_, _ = resp.Write(b)
					return
				case http.MethodDelete:
					id := req.URL.Path[len("/api/users/"):]

					uid, err := uid.Parse([]byte(id))
					assert.NilError(t, err)

					var found int
					for i := range modifiedUsers {
						if modifiedUsers[i].ID == uid {
							found = i
						}
					}
					modifiedUsers[found] = modifiedUsers[len(modifiedUsers)-1]
					modifiedUsers = modifiedUsers[:len(modifiedUsers)-1]
					resp.WriteHeader(http.StatusNoContent)
					return
				}
			}

			resp.WriteHeader(http.StatusBadRequest)
		}

		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.User{})
		err := writeConfig(&cfg)
		assert.NilError(t, err)

		return &modifiedUsers
	}

	t.Run("add user", func(t *testing.T) {
		modifiedUsers := setup(t)
		err := Run(context.Background(), "users", "add", "new-user@example.com")
		assert.NilError(t, err)

		assert.Equal(t, len(*modifiedUsers), 1)
	})

	t.Run("add user, not an email", func(t *testing.T) {
		_ = setup(t)
		err := Run(context.Background(), "users", "add", "new-user")
		assert.ErrorContains(t, err, "username must be a valid email")
	})

	t.Run("add without required argument", func(t *testing.T) {
		err := Run(context.Background(), "users", "add")
		assert.ErrorContains(t, err, `"infra users add" requires exactly 1 argument`)
		assert.ErrorContains(t, err, `Usage:  infra users add USER`)
	})

	t.Run("edit user no password flag", func(t *testing.T) {
		setup(t)
		err := Run(context.Background(), "users", "edit", "new-user@example.com")
		assert.ErrorContains(t, err, "Please specify a field to update. For options, run 'infra users edit --help'")
	})

	t.Run("edit without required argument", func(t *testing.T) {
		err := Run(context.Background(), "users", "edit")
		assert.ErrorContains(t, err, `"infra users edit" requires exactly 1 argument`)
		assert.ErrorContains(t, err, `Usage:  infra users edit USER`)
	})

	t.Run("removes only the specified user", func(t *testing.T) {
		users := setup(t)
		ctx := context.Background()
		err := Run(ctx, "users", "add", "to-delete-user@example.com")
		assert.NilError(t, err)
		assert.Equal(t, len(*users), 1)
		err = Run(ctx, "users", "add", "should-not-be-deleted-user@example.com")
		assert.NilError(t, err)
		assert.Equal(t, len(*users), 2)

		err = Run(ctx, "users", "remove", "to-delete-user@example.com")
		assert.NilError(t, err)
		assert.Equal(t, len(*users), 1)
		assert.Equal(t, (*users)[0].Name, "should-not-be-deleted-user@example.com")
	})

	t.Run("remove unknown user", func(t *testing.T) {
		setup(t)
		ctx := context.Background()
		err := Run(ctx, "users", "remove", "unknown@example.com")
		assert.ErrorContains(t, err, "No user named")
	})

	t.Run("remove without required argument", func(t *testing.T) {
		setup(t)
		err := Run(context.Background(), "users", "remove")
		assert.ErrorContains(t, err, `"infra users remove" requires exactly 1 argument`)
		assert.ErrorContains(t, err, `Usage:  infra users remove USER`)
	})

	t.Run("list with json", func(t *testing.T) {
		setup(t)
		err := Run(context.Background(), "users", "add", "apple@example.com")
		assert.NilError(t, err)
		ctx, bufs := PatchCLI(context.Background())
		err = Run(ctx, "users", "list", "--format=json")
		assert.NilError(t, err)

		golden.Assert(t, bufs.Stdout.String(), t.Name())
		assert.Assert(t, !strings.Contains(bufs.Stdout.String(), `count`))
		assert.Assert(t, !strings.Contains(bufs.Stdout.String(), `items`))
	})

	t.Run("list with yaml", func(t *testing.T) {
		setup(t)
		err := Run(context.Background(), "users", "add", "apple@example.com")
		assert.NilError(t, err)
		ctx, bufs := PatchCLI(context.Background())
		err = Run(ctx, "users", "list", "--format=yaml")
		assert.NilError(t, err)

		golden.Assert(t, bufs.Stdout.String(), t.Name())
	})
}

func TestUsersCmd_EditPassword(t *testing.T) {
	dir := setupEnv(t)

	opts := defaultServerOptions(dir)
	setupServerOptions(t, &opts)
	opts.BootstrapConfig.Users = []server.User{
		{
			Name:     "admin@local",
			Password: "password",
		},
	}
	srv, err := server.New(opts)
	assert.NilError(t, err)

	ctx := context.Background()
	runAndWait(ctx, t, srv.Run)

	runStep(t, "login", func(t *testing.T) {
		t.Setenv("INFRA_USER", "admin@local")
		t.Setenv("INFRA_PASSWORD", "password")
		t.Setenv("INFRA_SKIP_TLS_VERIFY", "true")

		err := Run(ctx, "login", srv.Addrs.HTTPS.String())
		assert.NilError(t, err)
	})

	t.Run("update user password", func(t *testing.T) {
		t.Setenv("INFRA_NON_INTERACTIVE", "false")

		ctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)

		console := newConsole(t)
		ctx = PatchCLIWithPTY(ctx, console.Tty())

		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			return Run(ctx, "users", "edit", "admin@local", "--password")
		})

		exp := expector{console: console}
		exp.ExpectString(t, "Old Password:")
		exp.Send(t, "password\n")
		exp.ExpectString(t, "New Password:")
		exp.Send(t, "p4ssword\n")
		exp.ExpectString(t, "Confirm New Password:")
		exp.Send(t, "p4ssword\n")
		exp.ExpectString(t, "Updated password")

		assert.NilError(t, g.Wait())
	})
}

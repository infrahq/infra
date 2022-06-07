package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestCheckPasswordRequirements(t *testing.T) {
	err := checkPasswordRequirements("")("password")
	assert.NilError(t, err)

	err = checkPasswordRequirements("")("passwor")
	assert.ErrorContains(t, err, "input must be at least 8 characters long")

	err = checkPasswordRequirements("password")("password")
	assert.ErrorContains(t, err, "input must be different than the current password")

	err = checkPasswordRequirements("password")(nil)
	assert.ErrorContains(t, err, "unexpected type for password")
}

func TestUsersCmd(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir) // for windows

	providerID := uid.New()

	setup := func(t *testing.T) *[]models.Identity {
		modifiedUsers := []models.Identity{}

		handler := func(resp http.ResponseWriter, req *http.Request) {
			if strings.Contains(req.URL.Path, "/api/providers") {
				resp.WriteHeader(http.StatusOK)

				providers := []*api.Provider{
					{
						Name: "infra",
						ID:   providerID,
					},
				}
				b, err := json.Marshal(providers)
				assert.NilError(t, err)
				_, _ = resp.Write(b)
				return
			}

			if strings.Contains(req.URL.Path, "/api/users") {
				switch req.Method {
				case http.MethodPost:
					createUserReq := api.CreateUserRequest{}

					err := json.NewDecoder(req.Body).Decode(&createUserReq)
					assert.NilError(t, err)

					newUser := models.Identity{
						Name: createUserReq.Name,
					}
					newUser.ID = uid.New()

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
						if mu.Name == name {
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
		err := Run(context.Background(), "users", "remove")
		assert.ErrorContains(t, err, `"infra users remove" requires exactly 1 argument`)
		assert.ErrorContains(t, err, `Usage:  infra users remove USER`)
	})
}

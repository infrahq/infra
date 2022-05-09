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

func TestCheckConfirmPassword(t *testing.T) {
	password := "password"

	err := checkConfirmPassword(&password)("password")
	assert.NilError(t, err)

	err = checkConfirmPassword(&password)("drowssap")
	assert.ErrorContains(t, err, "input must match the new password")

	err = checkConfirmPassword(&password)(nil)
	assert.ErrorContains(t, err, "unexpected type for password")
}

func TestIdentitiesCmd(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir) // for windows

	providerID := uid.New()

	setup := func(t *testing.T) *[]models.Identity {
		modifiedIdentities := []models.Identity{}

		handler := func(resp http.ResponseWriter, req *http.Request) {
			if strings.Contains(req.URL.Path, "/v1/providers") {
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

			if strings.Contains(req.URL.Path, "/v1/identities") {
				switch req.Method {
				case http.MethodPost:
					createIdentityReq := api.CreateIdentityRequest{}

					err := json.NewDecoder(req.Body).Decode(&createIdentityReq)
					assert.NilError(t, err)

					respBody := api.CreateIdentityResponse{
						ID:   uid.New(),
						Name: createIdentityReq.Name,
					}

					modifiedIdentities = append(modifiedIdentities, models.Identity{Name: createIdentityReq.Name})

					b, err := json.Marshal(&respBody)
					assert.NilError(t, err)
					_, _ = resp.Write(b)
					return
				case http.MethodGet:
					var apiIdentities []api.Identity
					for _, mi := range modifiedIdentities {
						apiIdentities = append(apiIdentities, *mi.ToAPI())
					}
					b, err := json.Marshal(api.ListResponse[api.Identity]{
						Items: apiIdentities,
						Count: len(apiIdentities),
					},
					)
					assert.NilError(t, err)
					_, _ = resp.Write(b)
					return
				case http.MethodDelete:
					id := req.URL.Path[len("/v1/identities/"):]

					uid, err := uid.Parse([]byte(id))
					assert.NilError(t, err)

					var found int
					for i := range modifiedIdentities {
						if modifiedIdentities[i].ID == uid {
							found = i
						}
					}
					modifiedIdentities[found] = modifiedIdentities[len(modifiedIdentities)-1]
					modifiedIdentities = modifiedIdentities[:len(modifiedIdentities)-1]

					resp.WriteHeader(http.StatusNoContent)
					return
				}
			}

			resp.WriteHeader(http.StatusBadRequest)
		}

		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.Identity{})
		err := writeConfig(&cfg)
		assert.NilError(t, err)

		return &modifiedIdentities
	}

	t.Run("add identity", func(t *testing.T) {
		modifiedIdentities := setup(t)
		err := Run(context.Background(), "id", "add", "new-user@example.com")
		assert.NilError(t, err)

		assert.Equal(t, len(*modifiedIdentities), 1)
	})

	t.Run("add without required argument", func(t *testing.T) {
		err := Run(context.Background(), "id", "add")
		assert.ErrorContains(t, err, `"infra identities add" requires exactly 1 argument`)
		assert.ErrorContains(t, err, `Usage:  infra identities add IDENTITY`)
	})

	t.Run("edit identity no password flag", func(t *testing.T) {
		setup(t)
		err := Run(context.Background(), "id", "edit", "new-user@example.com")
		assert.ErrorContains(t, err, "Please specify a field to update. For options, run 'infra identities edit --help'")
	})

	t.Run("edit identity interactive with password", func(t *testing.T) {
		setup(t)
		t.Setenv("INFRA_PASSWORD", "true")
		t.Setenv("INFRA_NON_INTERACTIVE", "true")
		err := Run(context.Background(), "id", "edit", "new-user@example.com")
		assert.ErrorContains(t, err, "Non-interactive mode is not supported to edit sensitive fields.")
	})

	t.Run("edit without required argument", func(t *testing.T) {
		err := Run(context.Background(), "id", "edit")
		assert.ErrorContains(t, err, `"infra identities edit" requires exactly 1 argument`)
		assert.ErrorContains(t, err, `Usage:  infra identities edit IDENTITY`)
	})

	t.Run("removes only the specified identity", func(t *testing.T) {
		modifiedIdentities := setup(t)
		ctx := context.Background()
		err := Run(ctx, "id", "add", "to-delete-user@example.com")
		assert.NilError(t, err)
		assert.Equal(t, len(*modifiedIdentities), 1)

		err = Run(ctx, "id", "remove", "to-delete-user@example.com")
		assert.NilError(t, err)
		assert.Equal(t, len(*modifiedIdentities), 0)
	})

	t.Run("remove non-existing user will error", func(t *testing.T) {
		_ = setup(t)
		ctx := context.Background()
		err := Run(ctx, "id", "remove", "non-existing-user@example.com")
		assert.ErrorContains(t, err, "No identities named")
	})

	t.Run("remove non-existing user with force will not error", func(t *testing.T) {
		_ = setup(t)
		ctx := context.Background()
		err := Run(ctx, "id", "remove", "--force", "non-existing-user@example.com")
		assert.NilError(t, err)
	})

	t.Run("remove without required argument", func(t *testing.T) {
		err := Run(context.Background(), "id", "remove")
		assert.ErrorContains(t, err, `"infra identities remove" requires exactly 1 argument`)
		assert.ErrorContains(t, err, `Usage:  infra identities remove IDENTITY`)
	})
}

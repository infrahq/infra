package cmd

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func random(n int) string {
	rand.Seed(time.Now().UnixNano())

	upper := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	lower := []rune("abcdefghijklmnopqrstuvwxyz")
	digit := []rune("0123456789")
	special := []rune("-_/")

	charset := make([]rune, 0)
	charset = append(charset, upper...)
	charset = append(charset, lower...)
	charset = append(charset, digit...)
	charset = append(charset, special...)

	var b strings.Builder
	for i := 0; i < n; i++ {
		//nolint:gosec
		b.WriteRune(charset[rand.Intn(len(charset))])
	}

	return b.String()
}

func TestCheckUserOrMachine(t *testing.T) {
	kind, err := checkUserOrMachine("alice")
	assert.NilError(t, err)
	assert.Equal(t, models.MachineKind, kind)

	kind, err = checkUserOrMachine("alice@example.com")
	assert.NilError(t, err)
	assert.Equal(t, models.UserKind, kind)

	kind, err = checkUserOrMachine("Alice <alice@example.com>")
	assert.NilError(t, err)
	assert.Equal(t, models.UserKind, kind)

	kind, err = checkUserOrMachine("<alice@example.com>")
	assert.NilError(t, err)
	assert.Equal(t, models.UserKind, kind)
}

func TestCheckUserOrMachineInvalidName(t *testing.T) {
	_, err := checkUserOrMachine(random(257))
	assert.ErrorContains(t, err, "invalid name: exceed maximum length requirement of 256 characters")

	// inputs with illegal runes are _not_ considered a name so it will
	// be passed to email validation instead
	illegalRunes := []rune("!@#$%^&*()=+[]{}\\|;:'\",<>?")
	for _, r := range illegalRunes {
		_, err = checkUserOrMachine(string(r))
		assert.ErrorContains(t, err, "input must be a valid email")
	}
}

func TestCheckUserOrMachineInvalidEmail(t *testing.T) {
	_, err := checkUserOrMachine("@example.com")
	assert.ErrorContains(t, err, "input must be a valid email")

	_, err = checkUserOrMachine("alice@")
	assert.ErrorContains(t, err, "input must be a valid email")
}

func TestCheckEmailRequirements(t *testing.T) {
	err := checkEmailRequirements("valid@email")
	assert.NilError(t, err)

	err = checkEmailRequirements("invalid")
	assert.ErrorContains(t, err, "input must be a valid email")

	err = checkEmailRequirements("invalid@")
	assert.ErrorContains(t, err, "input must be a valid email")

	err = checkEmailRequirements("@invalid")
	assert.ErrorContains(t, err, "input must be a valid email")

	err = checkEmailRequirements(nil)
	assert.ErrorContains(t, err, "unexpected type for email")
}

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

					kind, err := models.ParseIdentityKind(createIdentityReq.Kind)
					assert.NilError(t, err)

					respBody := api.CreateIdentityResponse{
						ID:   uid.New(),
						Name: createIdentityReq.Name,
					}

					if kind == models.UserKind {
						respBody.OneTimePassword = "abc"
					}

					modifiedIdentities = append(modifiedIdentities, models.Identity{Kind: kind, Name: createIdentityReq.Name})

					b, err := json.Marshal(&respBody)
					assert.NilError(t, err)
					_, _ = resp.Write(b)
					return
				case http.MethodGet:
					b, err := json.Marshal([]models.Identity{{Model: models.Model{ID: uid.New()}, Name: "to-delete-user@example.com", Kind: models.UserKind}})
					assert.NilError(t, err)
					_, _ = resp.Write(b)
					return
				case http.MethodDelete:
					id := req.URL.Path[len("/v1/identities/"):]

					uid, err := uid.ParseString(id)
					assert.NilError(t, err)

					modifiedIdentities = append(modifiedIdentities, models.Identity{Model: models.Model{ID: uid}})

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

	t.Run("add machine identity", func(t *testing.T) {
		modifiedIdentities := setup(t)
		err := Run(context.Background(), "id", "add", "new-user")
		assert.NilError(t, err)

		assert.Equal(t, len(*modifiedIdentities), 1)
		assert.Equal(t, models.MachineKind, (*modifiedIdentities)[0].Kind)
	})

	t.Run("add user identity", func(t *testing.T) {
		modifiedIdentities := setup(t)
		err := Run(context.Background(), "id", "add", "new-user@example.com")
		assert.NilError(t, err)

		assert.Equal(t, len(*modifiedIdentities), 1)
		assert.Equal(t, models.UserKind, (*modifiedIdentities)[0].Kind)
	})

	t.Run("add without required argument", func(t *testing.T) {
		err := Run(context.Background(), "id", "add")
		assert.ErrorContains(t, err, `"infra identities add" requires exactly 1 argument`)
		assert.ErrorContains(t, err, `Usage:  infra identities add IDENTITY`)
	})

	t.Run("edit machine identity fails", func(t *testing.T) {
		setup(t)
		err := Run(context.Background(), "id", "edit", "HAL")
		assert.ErrorContains(t, err, "machine identities have no editable fields")
	})

	t.Run("edit user identity no password flag", func(t *testing.T) {
		setup(t)
		err := Run(context.Background(), "id", "edit", "new-user@example.com")
		assert.ErrorContains(t, err, "Please specify a field to update. For options, run 'infra identities edit --help'")
	})

	t.Run("edit user identity interactive with password", func(t *testing.T) {
		setup(t)
		t.Setenv("INFRA_PASSWORD", "true")
		t.Setenv("INFRA_NON_INTERACTIVE", "true")
		err := Run(context.Background(), "id", "edit", "new-user@example.com")
		assert.ErrorContains(t, err, "Interactive mode is required to edit sensitive fields")
	})

	t.Run("edit without required argument", func(t *testing.T) {
		err := Run(context.Background(), "id", "edit")
		assert.ErrorContains(t, err, `"infra identities edit" requires exactly 1 argument`)
		assert.ErrorContains(t, err, `Usage:  infra identities edit IDENTITY`)
	})

	t.Run("removes only the specified identity", func(t *testing.T) {
		modifiedIdentities := setup(t)
		ctx := context.Background()
		err := Run(ctx, "id", "remove", "to-delete-user@example.com")
		assert.NilError(t, err)

		assert.Equal(t, len(*modifiedIdentities), 1)
	})

	t.Run("remove without required argument", func(t *testing.T) {
		err := Run(context.Background(), "id", "remove")
		assert.ErrorContains(t, err, `"infra identities remove" requires exactly 1 argument`)
		assert.ErrorContains(t, err, `Usage:  infra identities remove IDENTITY`)
	})
}

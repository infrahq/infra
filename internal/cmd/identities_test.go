package cmd

import (
	"encoding/json"
	"fmt"
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
		assert.ErrorContains(t, err, fmt.Sprintf("invalid email: %q", string(r)))
	}
}

func TestCheckUserOrMachineInvalidEmail(t *testing.T) {
	_, err := checkUserOrMachine("@example.com")
	assert.ErrorContains(t, err, "invalid email: \"@example.com\"")

	_, err = checkUserOrMachine("alice@")
	assert.ErrorContains(t, err, "invalid email: \"alice@\"")
}

func TestIdentities(t *testing.T) {
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
						ID:           uid.New(),
						Name:         createIdentityReq.Name,
						ProviderName: "infra",
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

		cfg := ClientConfig{
			Version: "0.3",
			Hosts: []ClientHostConfig{
				{
					PolymorphicID: "i:1234",
					Name:          "self@example.com",
					ProviderID:    providerID,
					Host:          srv.Listener.Addr().String(),
					Current:       true,
					AccessKey:     "the-access-key",
					SkipTLSVerify: true,
				},
			},
		}
		err := writeConfig(&cfg)
		assert.NilError(t, err)

		return &modifiedIdentities
	}

	t.Run("add machine identity", func(t *testing.T) {
		modifiedIdentities := setup(t)
		cmd := newIdentitiesAddCmd()
		cmd.SetArgs([]string{"new-user"})
		err := cmd.Execute()
		assert.NilError(t, err)

		assert.Equal(t, len(*modifiedIdentities), 1)
		assert.Equal(t, models.MachineKind, (*modifiedIdentities)[0].Kind)
	})

	t.Run("add user identity", func(t *testing.T) {
		modifiedIdentities := setup(t)
		cmd := newIdentitiesAddCmd()
		cmd.SetArgs([]string{"new-user@example.com"})
		err := cmd.Execute()
		assert.NilError(t, err)

		assert.Equal(t, len(*modifiedIdentities), 1)
		assert.Equal(t, models.UserKind, (*modifiedIdentities)[0].Kind)
	})

	t.Run("edit user identity no password flag", func(t *testing.T) {
		setup(t)
		cmd := newIdentitiesEditCmd()
		cmd.SetArgs([]string{"new-user@example.com"})
		err := cmd.Execute()

		assert.ErrorContains(t, err, "Specify a field to update")
	})

	t.Run("removes only the specified identity", func(t *testing.T) {
		modifiedIdentities := setup(t)
		cmd := newIdentitiesRemoveCmd()
		cmd.SetArgs([]string{"to-delete-user@example.com"})
		err := cmd.Execute()
		assert.NilError(t, err)

		assert.Equal(t, len(*modifiedIdentities), 1)
	})
}

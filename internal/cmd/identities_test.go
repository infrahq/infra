package cmd

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/infrahq/infra/internal/server/models"
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
	require.NoError(t, err)
	require.Equal(t, models.MachineKind, kind)

	kind, err = checkUserOrMachine("alice@example.com")
	require.NoError(t, err)
	require.Equal(t, models.UserKind, kind)

	kind, err = checkUserOrMachine("Alice <alice@example.com>")
	require.NoError(t, err)
	require.Equal(t, models.UserKind, kind)

	kind, err = checkUserOrMachine("<alice@example.com>")
	require.NoError(t, err)
	require.Equal(t, models.UserKind, kind)
}

func TestCheckUserOrMachineInvalidName(t *testing.T) {
	_, err := checkUserOrMachine(random(257))
	require.ErrorContains(t, err, "invalid name: exceed maximum length requirement of 256 characters")

	// inputs with illegal runes are _not_ considered a name so it will
	// be passed to email validation instead
	illegalRunes := []rune("!@#$%^&*()=+[]{}\\|;:'\",<>?")
	for _, r := range illegalRunes {
		_, err = checkUserOrMachine(string(r))
		require.ErrorContains(t, err, fmt.Sprintf("invalid email: %q", string(r)))
	}
}

func TestCheckUserOrMachineInvalidEmail(t *testing.T) {
	_, err := checkUserOrMachine("@example.com")
	require.ErrorContains(t, err, "invalid email: \"@example.com\"")

	_, err = checkUserOrMachine("alice@")
	require.ErrorContains(t, err, "invalid email: \"alice@\"")
}

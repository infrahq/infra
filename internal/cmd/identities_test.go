package cmd

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

func TestCheckNameOrEmail(t *testing.T) {
	name, email, err := checkNameOrEmail("alice")
	require.NoError(t, err)
	require.Equal(t, "alice", name)
	require.Empty(t, email)

	name, email, err = checkNameOrEmail("alice@example.com")
	require.NoError(t, err)
	require.Empty(t, name)
	require.Equal(t, "alice@example.com", email)

	name, email, err = checkNameOrEmail("Alice <alice@example.com>")
	require.NoError(t, err)
	require.Empty(t, name)
	require.Equal(t, "alice@example.com", email)

	name, email, err = checkNameOrEmail("<alice@example.com>")
	require.NoError(t, err)
	require.Empty(t, name)
	require.Equal(t, "alice@example.com", email)
}

func TestCheckNameOrEmailInvalidName(t *testing.T) {
	_, _, err := checkNameOrEmail(random(257))
	require.ErrorContains(t, err, "invalid name: exceed maximum length requirement of 256 characters")

	// inputs with illegal runes are _not_ considered a name so it will
	// be passed to email validation instead
	illegalRunes := []rune("!@#$%^&*()=+[]{}\\|;:'\",.<>?")
	for _, r := range illegalRunes {
		_, _, err = checkNameOrEmail(string(r))
		require.ErrorContains(t, err, fmt.Sprintf("invalid email: %q", string(r)))
	}
}

func TestCheckNameOrEmailInvalidEmail(t *testing.T) {
	_, _, err := checkNameOrEmail("@example.com")
	require.ErrorContains(t, err, "invalid email: \"@example.com\"")

	_, _, err = checkNameOrEmail("alice@")
	require.ErrorContains(t, err, "invalid email: \"alice@\"")
}

package cmd

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func random(n int, includeIllegalRunes bool) string {
	rand.Seed(time.Now().UnixNano())

	upper := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	lower := []rune("abcdefghijklmnopqrstuvwxyz")
	digit := []rune("0123456789")
	special := []rune("-_/")
	illegal := []rune("!@#$%^&*()=+[];:<>?")

	charset := make([]rune, 0)
	charset = append(charset, upper...)
	charset = append(charset, lower...)
	charset = append(charset, digit...)
	charset = append(charset, special...)

	if includeIllegalRunes {
		charset = append(charset, illegal...)
	}

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
	_, _, err := checkNameOrEmail(random(257, false))
	require.Error(t, err)

	_, _, err = checkNameOrEmail(random(16, true))
	require.Error(t, err)
}

func TestCheckNameOrEmailInvalidEmail(t *testing.T) {
	_, _, err := checkNameOrEmail("@example.com")
	require.Error(t, err)

	_, _, err = checkNameOrEmail("alice@")
	require.Error(t, err)
}

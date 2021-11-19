package generate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCryptoRandomNegativeLen(t *testing.T) {
	s, err := CryptoRandom(-1)
	require.NoError(t, err)
	assert.Equal(t, s, "")
}

func TestCryptoRandomLen(t *testing.T) {
	s, err := CryptoRandom(20)
	require.NoError(t, err)
	assert.Equal(t, len(s), 20)
}

func TestCryptoRandomCanGenerateEdgeCharacters(t *testing.T) {
	// check for off-by-one errors by making sure the random string generated can contain
	// both the first character in the list, and the last.
	// this test will time out or error on exhausting the entropy pool if it fails.
	testForCharacters := []byte{alphanum[0], alphanum[len(alphanum)-1]}
	for _, char := range testForCharacters {
		s, err := CryptoRandom(50)
		require.NoError(t, err)

		if strings.Contains(s, string(char)) {
			continue // found it we're good.
		}
	}
}

func TestSeedHasBeenInitialized(t *testing.T) {
	s := MathRandom(10)
	// the default seed of 1 will always generate RFbD56TI2s.
	require.NotEqual(t, "RFbD56TI2s", s)
}

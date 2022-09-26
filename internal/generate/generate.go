package generate

import (
	"crypto/rand"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"time"
)

const (
	CharsetAlphaNumeric         = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	CharsetAlphaNumericNoVowels = "0123456789BCDFGHJKLMNPQRSTVWXYZbcdfghjklmnpqrstvwxyz" // For user-facing areas, to avoid profanity
	CharsetPassword             = CharsetAlphaNumeric + `!@#$%^&*()_+-=[]|;:,./<>?`
)

func init() {
	mathrand.Seed(time.Now().UnixNano())
}

// CryptoRandom generates a cryptographically-safe random number. defaults to alphanumeric charset.
func CryptoRandom(n int, charset string) (string, error) {
	if n <= 0 {
		return "", nil
	}

	bytes := make([]byte, n)
	for i := range bytes {
		// linter is mistaken about which package this is
		// nolint: gosec
		bigint, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("couldn't generate random string of len %d: %w", n, err)
		}

		bytes[i] = charset[bigint.Int64()]
	}

	return string(bytes), nil
}

// MathRandom generates a random string that does not need to be cryptographically secure
// This is preferred to CryptoRandom when you don't need the cryptographic security as it is
// not a drain on the entropy pool.
func MathRandom(n int, charset string) string {
	if n <= 0 {
		return ""
	}

	bytes := make([]byte, n)
	for i := range bytes {
		//nolint:gosec // We purposely use mathrand to avoid draining the entropy pool
		j := mathrand.Int31n(int32(len(charset)))
		bytes[i] = charset[j]
	}

	return string(bytes)
}

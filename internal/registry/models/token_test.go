package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/infrahq/infra/internal/generate"
)

func TestIssueToken(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"TokenKeyAndSecretEmpty": {
			"token": &Token{SessionDuration: 1 * time.Hour},
			"verifyFunc": func(t *testing.T, before, result *Token, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, result.Key)
				require.NotEmpty(t, result.Secret)
				require.NotEmpty(t, result.Checksum)
				require.True(t, time.Now().Before(result.Expires))
			},
		},
		"TokenKeySet": {
			"token": &Token{Key: generate.MathRandom(TokenKeyLength), SessionDuration: 1 * time.Hour},
			"verifyFunc": func(t *testing.T, before, result *Token, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, result.Secret)
				require.NotEmpty(t, result.Checksum)
				require.Equal(t, before.Key, result.Key)
				require.True(t, time.Now().Before(result.Expires))
			},
		},
		"TokenSecretSet": {
			"token": &Token{Secret: generate.MathRandom(TokenSecretLength), SessionDuration: 1 * time.Hour},
			"verifyFunc": func(t *testing.T, before, result *Token, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, result.Key)
				require.NotEmpty(t, result.Checksum)
				require.Equal(t, before.Secret, result.Secret)
				require.True(t, time.Now().Before(result.Expires))
			},
		},
		"TokenKeyAndSecretSet": {
			"token": &Token{Key: generate.MathRandom(TokenKeyLength), Secret: generate.MathRandom(TokenSecretLength), SessionDuration: 1 * time.Hour},
			"verifyFunc": func(t *testing.T, before, result *Token, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, result.Checksum)
				require.Equal(t, before.Key, result.Key)
				require.Equal(t, before.Secret, result.Secret)
				require.True(t, time.Now().Before(result.Expires))
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			token, ok := v["token"].(*Token)
			require.True(t, ok)

			before := &Token{Key: token.Key, Secret: token.Secret}
			fmt.Println(before.Key)
			err := Issue(token)

			verifyFunc, ok := v["verifyFunc"].(func(*testing.T, *Token, *Token, error))
			require.True(t, ok)

			verifyFunc(t, before, token, err)
		})
	}
}

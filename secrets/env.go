package secrets

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// implements env storage for secret config

type EnvSecretProvider struct {
	GenericConfig
}

func NewEnvSecretProviderFromConfig(cfg GenericConfig) *EnvSecretProvider {
	return &EnvSecretProvider{
		GenericConfig: cfg,
	}
}

var _ SecretStorage = &EnvSecretProvider{}

func (fp *EnvSecretProvider) SetSecret(name string, secret []byte) error {
	if strings.Contains(name, "$") {
		return errors.New("ENV secrets cannot contain $")
	}

	name = invalidNameChars.ReplaceAllString(name, "_")

	var b []byte

	if fp.Base64 {
		b = make([]byte, fp.encoder().EncodedLen(len(secret)))
		fp.encoder().Encode(b, secret)
	} else {
		b = make([]byte, len(secret))
		copy(b, secret)
	}

	if err := os.Setenv(name, string(b)); err != nil {
		return fmt.Errorf("setenv: %w", err)
	}

	return nil
}

var invalidNameChars = regexp.MustCompile(`[^\w\d-]`)

func (fp *EnvSecretProvider) GetSecret(name string) (secret []byte, err error) {
	var b []byte
	if strings.Contains(name, "$") {
		b = []byte(os.ExpandEnv(name))
	} else {
		name = invalidNameChars.ReplaceAllString(name, "_")
		b = []byte(os.Getenv(name))
	}

	_, present := os.LookupEnv(name)
	if !present {
		return nil, ErrNotFound
	}

	var result []byte
	if fp.Base64 {
		result = make([]byte, fp.encoder().DecodedLen(len(b)))

		written, err := fp.encoder().Decode(result, b)
		if err != nil {
			return nil, fmt.Errorf("base64 decoding %q: %w", name, err)
		}

		result = result[:written]

		return result, nil
	}

	return b, nil
}

func (fp *EnvSecretProvider) encoder() *base64.Encoding {
	if fp.Base64URLEncoded {
		if fp.Base64Raw {
			return base64.RawURLEncoding
		} else {
			return base64.URLEncoding
		}
	} else { // std encoding
		if fp.Base64Raw {
			return base64.RawStdEncoding
		} else {
			return base64.StdEncoding
		}
	}
}

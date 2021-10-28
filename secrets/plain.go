package secrets

import (
	"encoding/base64"
	"errors"
	"fmt"
)

var ErrNotImplemented = errors.New("not implemented")

// implements plain "storage" for secret config

type PlainSecretProvider struct {
	GenericConfig
}

func NewPlainSecretProviderFromConfig(cfg GenericConfig) *PlainSecretProvider {
	return &PlainSecretProvider{
		GenericConfig: cfg,
	}
}

var _ SecretStorage = &PlainSecretProvider{}

func (fp *PlainSecretProvider) SetSecret(name string, secret []byte) error {
	return ErrNotImplemented // and not really possible to implement...
}

func (fp *PlainSecretProvider) GetSecret(name string) (secret []byte, err error) {
	b := []byte(name)

	var result []byte
	if fp.Base64 {
		result = make([]byte, fp.encoder().DecodedLen(len(b)))

		written, err := fp.encoder().Decode(result, b)
		if err != nil {
			return nil, fmt.Errorf("base64 decoding: %w", err)
		}

		result = result[:written]

		return result, nil
	}

	return b, nil
}

func (fp *PlainSecretProvider) encoder() *base64.Encoding {
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

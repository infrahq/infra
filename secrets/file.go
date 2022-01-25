package secrets

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

// implements file storage for secret config

type GenericConfig struct {
	Base64           bool `yaml:"base64"`
	Base64URLEncoded bool `yaml:"base64UrlEncoded"`
	Base64Raw        bool `yaml:"base64Raw"`
}

type FileConfig struct {
	GenericConfig
	Path string `yaml:"path" validate:"required"`
}

type FileSecretProvider struct {
	FileConfig
}

func NewFileSecretProviderFromConfig(cfg FileConfig) *FileSecretProvider {
	return &FileSecretProvider{
		FileConfig: cfg,
	}
}

var _ SecretStorage = &FileSecretProvider{}

func (fp *FileSecretProvider) SetSecret(name string, secret []byte) error {
	fullPath := name

	if len(fp.Path) > 0 {
		fullPath = path.Join(fp.Path, name)
	}

	dir := path.Dir(fullPath)
	if len(dir) > 0 {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %q: %w", fp.Path, err)
		}
	}

	var b []byte

	if fp.Base64 {
		b = make([]byte, fp.encoder().EncodedLen(len(secret)))
		fp.encoder().Encode(b, secret)
	} else {
		b = make([]byte, len(secret))
		copy(b, secret)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("creating file %q: %w", fullPath, err)
	}

	if _, err := f.Write(b); err != nil {
		return fmt.Errorf("writing file %q: %w", fullPath, err)
	}

	return nil
}

func (fp *FileSecretProvider) GetSecret(name string) (secret []byte, err error) {
	fullPath := path.Join(fp.Path, name)

	f, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("opening file %q: %w", fullPath, err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading file %q: %w", fullPath, err)
	}

	var result []byte
	if fp.Base64 {
		result = make([]byte, fp.encoder().DecodedLen(len(b)))

		written, err := fp.encoder().Decode(result, b)
		if err != nil {
			return nil, fmt.Errorf("base64 decoding file %q: %w", fullPath, err)
		}

		result = result[:written]

		return result, nil
	}

	return b, nil
}

func (fp *FileSecretProvider) encoder() *base64.Encoding {
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

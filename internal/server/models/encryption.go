package models

import (
	"database/sql/driver"
	"fmt"

	"github.com/infrahq/infra/internal/server/data/encrypt"
)

// EncryptedAtRest defines a field that knows how to encrypt and decrypt itself with Gorm
// it depends on the SymmetricKey being set for this package.
type EncryptedAtRest string

// SymmetricKey is the key used to encrypt and decrypt this field.
var SymmetricKey *encrypt.SymmetricKey

// SkipSymmetricKey is used for tests that specifically want to avoid field encryption
var SkipSymmetricKey bool

func (s EncryptedAtRest) Encrypt() (string, error) {
	if SkipSymmetricKey || s == "" {
		return string(s), nil
	}

	if SymmetricKey == nil {
		return "", fmt.Errorf("models.SymmetricKey is not set")
	}

	b, err := encrypt.Seal(SymmetricKey, []byte(s))
	if err != nil {
		return "", fmt.Errorf("sealing secret field: %w", err)
	}

	return string(b), err
}

func (s EncryptedAtRest) Value() (driver.Value, error) {
	return s.Encrypt()
}

func decrypt(s string) (string, error) {
	if SkipSymmetricKey || s == "" {
		return s, nil
	}

	if SymmetricKey == nil {
		return "", fmt.Errorf("models.SymmetricKey is not set")
	}

	b, err := encrypt.Unseal(SymmetricKey, []byte(s))
	if err != nil {
		return "", fmt.Errorf("unsealing secret field: %w", err)
	}

	return string(b), err
}

func (s *EncryptedAtRest) Scan(v interface{}) error {
	var vStr string
	switch typ := v.(type) {
	case string:
		vStr = typ
	case []byte:
		vStr = string(typ)
	case nil:
		vStr = ""
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}

	str, err := decrypt(vStr)
	if err != nil {
		return err
	}

	*s = EncryptedAtRest(str)

	return nil
}

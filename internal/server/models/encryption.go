package models

import (
	"database/sql/driver"
	"fmt"

	"github.com/infrahq/secrets"
)

// EncryptedAtRest defines a field that knows how to encrypt and decrypt itself with Gorm
// it depends on the SymmetricKey being set for this package.
type EncryptedAtRest string

// SymmetricKey is the key used to encrypt and decrypt this field.
var SymmetricKey *secrets.SymmetricKey

// SkipSymmetricKey is used for tests that specifically want to avoid field encryption
var SkipSymmetricKey bool

func (s EncryptedAtRest) Encrypt() (string, error) {
	if SkipSymmetricKey {
		return string(s), nil
	}

	if SymmetricKey == nil {
		return "", fmt.Errorf("models.SymmetricKey is not set")
	}

	b, err := secrets.Seal(SymmetricKey, []byte(s))
	if err != nil {
		return "", fmt.Errorf("sealing secret field: %w", err)
	}

	return string(b), err
}

func (s EncryptedAtRest) Value() (driver.Value, error) {
	return s.Encrypt()
}

func (s EncryptedAtRest) Decrypt() (string, error) {
	if SkipSymmetricKey {
		return string(s), nil
	}

	if SymmetricKey == nil {
		return "", fmt.Errorf("models.SymmetricKey is not set")
	}

	b, err := secrets.Unseal(SymmetricKey, []byte(s))
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
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}

	if SkipSymmetricKey {
		*s = EncryptedAtRest(vStr)
		return nil
	}

	str, err := EncryptedAtRest(vStr).Decrypt()
	if err != nil {
		return err
	}

	*s = EncryptedAtRest(str)

	return nil
}

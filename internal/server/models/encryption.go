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

func (s EncryptedAtRest) Value() (driver.Value, error) {
	if SkipSymmetricKey {
		return string(s), nil
	}

	if SymmetricKey == nil {
		return nil, fmt.Errorf("models.SymmetricKey is not set")
	}

	b, err := secrets.Seal(SymmetricKey, []byte(s))
	if err != nil {
		return nil, fmt.Errorf("sealing secret field: %w", err)
	}

	return string(b), err
}

func (s *EncryptedAtRest) Scan(v interface{}) error {
	vStr, ok := v.(string)
	if !ok {
		return fmt.Errorf("unsupported type: %T", v)
	}

	if SkipSymmetricKey {
		*s = EncryptedAtRest(vStr)
		return nil
	}

	if SymmetricKey == nil {
		return fmt.Errorf("models.SymmetricKey is not set")
	}

	b, err := secrets.Unseal(SymmetricKey, []byte(vStr))
	if err != nil {
		return fmt.Errorf("unsealing secret field: %w", err)
	}

	*s = EncryptedAtRest(b)

	return nil
}

// EncryptedAtRestBytes defines a field that knows how to encrypt and decrypt itself with Gorm
// it depends on the SymmetricKey being set for this package.
type EncryptedAtRestBytes []byte

func (b EncryptedAtRestBytes) Value() (driver.Value, error) {
	if SkipSymmetricKey {
		return []byte(b), nil
	}

	if SymmetricKey == nil {
		return nil, fmt.Errorf("models.SymmetricKey is not set")
	}

	be, err := secrets.Seal(SymmetricKey, b)
	if err != nil {
		return nil, fmt.Errorf("sealing secret field: %w", err)
	}

	return be, err
}

func (b *EncryptedAtRestBytes) Scan(v interface{}) error {
	vBytes, ok := v.([]byte)
	if !ok {
		return fmt.Errorf("unsupported type: %T", v)
	}

	if SkipSymmetricKey {
		*b = EncryptedAtRestBytes(vBytes)
		return nil
	}

	if SymmetricKey == nil {
		return fmt.Errorf("models.SymmetricKey is not set")
	}

	plain, err := secrets.Unseal(SymmetricKey, vBytes)
	if err != nil {
		return fmt.Errorf("unsealing secret field: %w", err)
	}

	*b = EncryptedAtRestBytes(plain)

	return nil
}

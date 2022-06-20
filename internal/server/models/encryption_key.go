package models

type EncryptionKey struct {
	Model

	KeyID     int32 `gorm:"uniqueIndex:idx_encryption_keys_key_id"` // a short identifier for the key that can be embedded with the encrypted payload
	Name      string
	Encrypted []byte
	Algorithm string
	RootKeyID string
}

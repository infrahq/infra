package models

type EncryptionKey struct {
	Model

	// KeyID is not used yet. KeyID is intended to be a short identifier for the key
	// that can be embedded with the encrypted payload. Today we use the first 4
	// bytes of a checksum of the encrypted data key instead of this identifier.
	// In the future we will use this identifier to support key rotation.
	KeyID int32
	// TODO: missing a unique index on name
	Name      string
	Encrypted []byte
	Algorithm string
	RootKeyID string
}

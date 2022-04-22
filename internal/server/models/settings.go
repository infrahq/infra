package models

type Settings struct {
	Model

	PrivateJWK EncryptedAtRestBytes
	PublicJWK  []byte

	SignupEnabled bool
}

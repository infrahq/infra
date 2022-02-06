package models

type Settings struct {
	Model

	PrivateJWK []byte
	PublicJWK  []byte
}

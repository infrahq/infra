package models

type Settings struct {
	Model
	OrganizationMember

	PrivateJWK EncryptedAtRest
	PublicJWK  []byte
}

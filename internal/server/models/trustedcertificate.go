package models

import (
	"time"
)

type TrustedCertificate struct {
	Model

	KeyAlgorithm     string `validate:"required"`
	SigningAlgorithm string `validate:"required"`
	PublicKey        Base64 `validate:"required"`
	CertPEM          []byte `validate:"required"` // pem encoded
	Identity         string `validate:"required"`
	ExpiresAt        time.Time
	OneTimeUse       bool
}

type RootCertificate struct {
	Model

	KeyAlgorithm     string          `validate:"required"`
	SigningAlgorithm string          `validate:"required"`
	PublicKey        Base64          `validate:"required"`
	PrivateKey       EncryptedAtRest `validate:"required"`
	SignedCert       EncryptedAtRest `validate:"required"` // contains private key? probably not pem encoded
	ExpiresAt        time.Time       `validate:"required"`
}

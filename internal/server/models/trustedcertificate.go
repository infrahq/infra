package models

import "time"

type TrustedCertificate struct {
	Model

	PublicKey []byte
	CertPEM   []byte
	Identity  string
	ExpiresAt time.Time
}

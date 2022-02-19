package data

import (
	"errors"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
	"gorm.io/gorm"
)

// TrustPublicKey trusts a public key (in base64 format) from a user or service
// Callers must have received the key from a mTLS/e2ee (mutually encrypted), trusted source.
func TrustPublicKey(db *gorm.DB, tc *models.TrustedCertificate) error {
	_, err := get[models.TrustedCertificate](db, ByPublicKey(tc.PublicKey))
	if err != nil && !errors.Is(err, internal.ErrNotFound) {
		return err
	}

	if err == nil {
		// this one already exists
		return nil
	}

	return add(db, tc)
}

func TrustedCertificates(db *gorm.DB) ([]models.TrustedCertificate, error) {
	return list[models.TrustedCertificate](db)
}

func ByPublicKey(key []byte) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("public_key = ?", key)
	}
}

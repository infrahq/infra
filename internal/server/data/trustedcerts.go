package data

import (
	"encoding/base64"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
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

func ListTrustedClientCertificates(db *gorm.DB) ([]models.TrustedCertificate, error) {
	return list[models.TrustedCertificate](db)
}

func ListRootCertificates(db *gorm.DB) ([]models.RootCertificate, error) {
	return list[models.RootCertificate](db, OrderBy("id desc"), ByNotExpired(), Limit(2))
}

func GetRootCertificate(db *gorm.DB, selectors ...SelectorFunc) (*models.RootCertificate, error) {
	return get[models.RootCertificate](db, selectors...)
}

func AddRootCertificate(db *gorm.DB, cert *models.RootCertificate) error {
	return add(db, cert)
}

func ByPublicKey(key []byte) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		k := base64.StdEncoding.EncodeToString(key)
		return db.Where("public_key = ?", k)
	}
}

func OrderBy(order string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order(order)
	}
}

func Limit(limit int) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Limit(limit)
	}
}

func ByNotExpired() SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("expires_at is null or expires_at > ?", time.Now().UTC())
	}
}

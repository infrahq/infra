package data

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ssoroka/slice"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type accessKeyTable models.AccessKey

func (a accessKeyTable) Table() string {
	return "access_keys"
}

func (a accessKeyTable) Columns() []string {
	return []string{"created_at", "deleted_at", "expires_at", "extension", "extension_deadline", "id", "issued_for", "key_id", "name", "provider_id", "scopes", "secret_checksum", "updated_at"}
}

func (a accessKeyTable) Values() []any {
	return []any{a.CreatedAt, a.DeletedAt, a.ExpiresAt, a.Extension, a.ExtensionDeadline, a.ID, a.IssuedFor, a.KeyID, a.Name, a.ProviderID, a.Scopes, a.SecretChecksum, a.UpdatedAt}
}

func (a *accessKeyTable) ScanFields() []any {
	return []any{&a.CreatedAt, &a.DeletedAt, &a.ExpiresAt, &a.Extension, &a.ExtensionDeadline, &a.ID, &a.IssuedFor, &a.KeyID, &a.Name, &a.ProviderID, &a.Scopes, &a.SecretChecksum, &a.UpdatedAt}
}

var (
	ErrAccessKeyExpired          = fmt.Errorf("access key expired")
	ErrAccessKeyDeadlineExceeded = fmt.Errorf("%w: extension deadline exceeded", ErrAccessKeyExpired)
)

func secretChecksum(secret string) []byte {
	chksm := sha256.Sum256([]byte(secret))
	return chksm[:]
}

func CreateAccessKey(db GormTxn, accessKey *models.AccessKey) (body string, err error) {
	switch {
	case accessKey.IssuedFor == 0:
		return "", fmt.Errorf("issusedFor is required")
	case accessKey.ProviderID == 0:
		return "", fmt.Errorf("providerID is required")
	}

	if accessKey.KeyID == "" {
		accessKey.KeyID = generate.MathRandom(models.AccessKeyKeyLength, generate.CharsetAlphaNumeric)
	}

	if len(accessKey.KeyID) != models.AccessKeyKeyLength {
		return "", fmt.Errorf("invalid key length")
	}

	if accessKey.Secret == "" {
		secret, err := generate.CryptoRandom(models.AccessKeySecretLength, generate.CharsetAlphaNumeric)
		if err != nil {
			return "", err
		}

		accessKey.Secret = secret
	}

	if len(accessKey.Secret) != models.AccessKeySecretLength {
		return "", fmt.Errorf("invalid secret length")
	}

	accessKey.SecretChecksum = secretChecksum(accessKey.Secret)

	if accessKey.ExpiresAt.IsZero() {
		accessKey.ExpiresAt = time.Now().Add(time.Hour * 12).UTC()
	}

	if accessKey.Name == "" {
		// set a default name for look-up and CLI usage
		if accessKey.ID == 0 {
			accessKey.ID = uid.New()
		}

		identityIssuedFor, err := GetIdentity(db, ByID(accessKey.IssuedFor))
		if err != nil {
			return "", fmt.Errorf("key name from identity: %w", err)
		}

		accessKey.Name = fmt.Sprintf("%s-%s", identityIssuedFor.Name, accessKey.ID.String())
	}

	if err := add(db, accessKey); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s", accessKey.KeyID, accessKey.Secret), nil
}

func SaveAccessKey(db GormTxn, key *models.AccessKey) error {
	if key.Secret != "" {
		key.SecretChecksum = secretChecksum(key.Secret)
	}

	return save(db, key)
}

func ListAccessKeys(db GormTxn, p *Pagination, selectors ...SelectorFunc) ([]models.AccessKey, error) {
	return list[models.AccessKey](db, p, selectors...)
}

func GetAccessKey(tx GormTxn, selectors ...SelectorFunc) (*models.AccessKey, error) {
	db := tx.GormDB()
	// GetAccessKey by keyID needs to not set an organization_id in the query.
	// keyID should be globally unique.
	for _, selector := range selectors {
		db = selector(db)
	}
	result := new(models.AccessKey)
	if err := db.Model(result).First(result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, internal.ErrNotFound
		}
		return nil, err
	}
	return result, nil
}

func DeleteAccessKey(db GormTxn, id uid.ID) error {
	return delete[models.AccessKey](db, id)
}

func DeleteAccessKeys(db GormTxn, selectors ...SelectorFunc) error {
	toDelete, err := list[models.AccessKey](db, nil, selectors...)
	if err != nil {
		return err
	}

	ids := slice.Map[models.AccessKey, uid.ID](toDelete, func(k models.AccessKey) uid.ID {
		return k.ID
	})

	return deleteAll[models.AccessKey](db, ByIDs(ids))
}

func ValidateAccessKey(tx GormTxn, authnKey string) (*models.AccessKey, error) {
	keyID, secret, ok := strings.Cut(authnKey, ".")
	if !ok {
		return nil, fmt.Errorf("invalid access key format")
	}

	t, err := GetAccessKey(tx, ByKeyID(keyID))
	if err != nil {
		return nil, fmt.Errorf("%w: could not get access key from database, it may not exist", err)
	}

	sum := secretChecksum(secret)

	if subtle.ConstantTimeCompare(t.SecretChecksum, sum) != 1 {
		return nil, fmt.Errorf("access key invalid secret")
	}

	if time.Now().UTC().After(t.ExpiresAt) {
		return nil, ErrAccessKeyExpired
	}

	if !t.ExtensionDeadline.IsZero() {
		if time.Now().UTC().After(t.ExtensionDeadline) {
			return nil, ErrAccessKeyDeadlineExceeded
		}

		t.ExtensionDeadline = time.Now().UTC().Add(t.Extension)
		if err := SaveAccessKey(tx, t); err != nil {
			return nil, err
		}
	}

	return t, nil
}

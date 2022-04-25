package data

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func secretChecksum(secret string) []byte {
	chksm := sha256.Sum256([]byte(secret))
	return chksm[:]
}

func CreateAccessKey(db *gorm.DB, accessKey *models.AccessKey) (body string, err error) {
	if accessKey.KeyID == "" {
		key, err := generate.CryptoRandom(models.AccessKeyKeyLength)
		if err != nil {
			return "", err
		}

		accessKey.KeyID = key
	}

	if len(accessKey.KeyID) != models.AccessKeyKeyLength {
		return "", fmt.Errorf("invalid key length")
	}

	if accessKey.Secret == "" {
		secret, err := generate.CryptoRandom(models.AccessKeySecretLength)
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
		if accessKey.ID == 0 {
			accessKey.ID = uid.New()
		}

		accessKey.Name = accessKey.ID.String()
	}

	if err := add(db, accessKey); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s", accessKey.KeyID, accessKey.Secret), nil
}

func SaveAccessKey(db *gorm.DB, key *models.AccessKey) error {
	if key.Secret != "" {
		key.SecretChecksum = secretChecksum(key.Secret)
	}

	return save(db, key)
}

func ListAccessKeys(db *gorm.DB, selectors ...SelectorFunc) ([]models.AccessKey, error) {
	return list[models.AccessKey](db, selectors...)
}

func GetAccessKey(db *gorm.DB, selectors ...SelectorFunc) (*models.AccessKey, error) {
	return get[models.AccessKey](db, selectors...)
}

func DeleteAccessKey(db *gorm.DB, id uid.ID) error {
	return delete[models.AccessKey](db, id)
}

func DeleteAccessKeys(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := list[models.AccessKey](db, selectors...)
	if err != nil {
		return err
	}

	ids := make([]uid.ID, 0)
	for _, k := range toDelete {
		ids = append(ids, k.ID)
	}

	return deleteAll[models.AccessKey](db, ByIDs(ids))
}

func ValidateAccessKey(db *gorm.DB, authnKey string) (*models.AccessKey, error) {
	keyID, secret, ok := strings.Cut(authnKey, ".")
	if !ok {
		return nil, fmt.Errorf("invalid access key format")
	}

	t, err := GetAccessKey(db, ByKeyID(keyID))
	if err != nil {
		return nil, fmt.Errorf("%w: could not get access key from database, it may not exist", err)
	}

	sum := secretChecksum(secret)

	if subtle.ConstantTimeCompare(t.SecretChecksum, sum) != 1 {
		return nil, fmt.Errorf("access key invalid secret")
	}

	if time.Now().After(t.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	if !t.ExtensionDeadline.IsZero() {
		if time.Now().After(t.ExtensionDeadline) {
			return nil, fmt.Errorf("token extension deadline exceeded")
		}

		t.ExtensionDeadline = time.Now().Add(t.Extension).UTC()
		if err := SaveAccessKey(db, t); err != nil {
			return nil, err
		}
	}

	return t, nil
}

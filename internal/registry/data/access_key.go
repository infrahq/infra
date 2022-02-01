package data

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/uid"
	"gorm.io/gorm"
)

func CreateAccessKey(db *gorm.DB, authnKey *models.AccessKey) (body string, err error) {
	if authnKey.Key == "" {
		key, err := generate.CryptoRandom(models.AccessKeyKeyLength)
		if err != nil {
			return "", err
		}

		authnKey.Key = key
	}

	if authnKey.Secret == "" {
		secret, err := generate.CryptoRandom(models.AccessKeySecretLength)
		if err != nil {
			return "", err
		}

		authnKey.Secret = secret
	}

	chksm := sha256.Sum256([]byte(authnKey.Secret))
	authnKey.SecretChecksum = chksm[:]

	if authnKey.ExpiresAt.IsZero() {
		authnKey.ExpiresAt = time.Now().Add(time.Hour * 12)
	}

	if err := add(db, authnKey); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s", authnKey.Key, authnKey.Secret), nil
}

func ListAccessKeys(db *gorm.DB, selectors ...SelectorFunc) ([]models.AccessKey, error) {
	return list[models.AccessKey](db, selectors...)
}

func GetAccessKeys(db *gorm.DB, selectors ...SelectorFunc) (*models.AccessKey, error) {
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

func LookupAccessKey(db *gorm.DB, authnKey string) (*models.AccessKey, error) {
	parts := strings.Split(authnKey, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("rejected access key format")
	}

	t, err := GetAccessKeys(db, ByKey(parts[0]))
	if err != nil {
		return nil, fmt.Errorf("%w could not get access key from database, it may not exist", err)
	}

	sum := sha256.Sum256([]byte(parts[1]))
	if subtle.ConstantTimeCompare(t.SecretChecksum, sum[:]) != 1 {
		return nil, fmt.Errorf("access key invalid secret")
	}

	if time.Now().After(t.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	return t, nil
}

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

func CreateAPIToken(db *gorm.DB, token *models.APIToken) (body string, err error) {
	generated, err := generate.CryptoRandom(models.APITokenSecretLength)
	if err != nil {
		return "", err
	}

	token.Secret = generated

	chksm := sha256.Sum256([]byte(token.Secret))
	token.SecretChecksum = chksm[:]

	if token.ExpiresAt.IsZero() {
		token.ExpiresAt = time.Now().Add(time.Hour * 12)
	}

	if err := add(db, token); err != nil {
		return "", err
	}

	return token.ID.String() + "." + token.Secret, nil
}

func ListAPITokens(db *gorm.DB, selectors ...SelectorFunc) ([]models.APIToken, error) {
	return list[models.APIToken](db, selectors...)
}

func GetAPIToken(db *gorm.DB, selectors ...SelectorFunc) (*models.APIToken, error) {
	return get[models.APIToken](db, selectors...)
}

func DeleteAPIToken(db *gorm.DB, id uid.ID) error {
	return delete[models.APIToken](db, id)
}

func DeleteAPITokens(db *gorm.DB, selectors ...SelectorFunc) error {
	return deleteAll[models.APIToken](db, selectors...)
}

func LookupAPIToken(db *gorm.DB, token string) (*models.APIToken, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("rejected token format")
	}

	id := uid.New()
	if err := id.UnmarshalText([]byte(parts[0])); err != nil {
		return nil, fmt.Errorf("%w: rejected token format", err)
	}

	t, err := GetAPIToken(db, ByID(id))
	if err != nil {
		return nil, fmt.Errorf("%w could not get token from database, it may not exist", err)
	}

	sum := sha256.Sum256([]byte(parts[1]))
	if subtle.ConstantTimeCompare(t.SecretChecksum, sum[:]) != 1 {
		return nil, fmt.Errorf("token invalid secret")
	}

	if time.Now().After(t.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	return t, nil
}

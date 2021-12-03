package data

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/registry/models"
)

func CreateToken(db *gorm.DB, token *models.Token) (*models.Token, error) {
	secret, err := generate.CryptoRandom(models.TokenSecretLength)
	if err != nil {
		return nil, err
	}

	chksm := sha256.Sum256([]byte(secret))
	token.Checksum = chksm[:]
	token.Secret = secret
	token.Key = generate.MathRandom(models.TokenKeyLength)
	token.Expires = time.Now().Add(token.SessionDuration)

	if err := add(db, &models.Token{}, token, &models.Token{}); err != nil {
		return nil, err
	}

	return token, nil
}

func UpdateToken(db *gorm.DB, token *models.Token) (*models.Token, error) {
	if err := update(db, &models.Token{}, token, db.Where(token, "id")); err != nil {
		return nil, err
	}

	return token, nil
}

func GetToken(db *gorm.DB, condition interface{}) (*models.Token, error) {
	var token models.Token
	if err := get(db, &models.Token{}, &token, condition); err != nil {
		return nil, err
	}

	return &token, nil
}

func CheckTokenExpired(t *models.Token) error {
	if time.Now().After(t.Expires) {
		return fmt.Errorf("token expired")
	}

	return nil
}

func CheckTokenSecret(t *models.Token, authorization string) error {
	sum := sha256.Sum256([]byte(authorization[models.TokenKeyLength:]))
	if subtle.ConstantTimeCompare(t.Checksum, sum[:]) != 1 {
		return fmt.Errorf("token invalid secret")
	}

	return nil
}

func DeleteToken(db *gorm.DB, condition interface{}) error {
	toDelete, err := GetToken(db, condition)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return err
		}
	}

	if toDelete != nil {
		return remove(db, &models.Token{}, toDelete.ID)
	}

	return nil
}

func CreateAPIKey(db *gorm.DB, apiKey *models.APIKey) (*models.APIKey, error) {
	if apiKey.Key == "" {
		key, err := generate.CryptoRandom(models.APIKeyLength)
		if err != nil {
			return nil, err
		}

		apiKey.Key = key
	}

	if err := add(db, &models.APIKey{}, apiKey, &models.APIKey{Name: apiKey.Name}); err != nil {
		return nil, err
	}

	return apiKey, nil
}

func GetAPIKey(db *gorm.DB, condition interface{}) (*models.APIKey, error) {
	var apiKey models.APIKey
	if err := get(db, &models.APIKey{}, &apiKey, condition); err != nil {
		return nil, err
	}

	return &apiKey, nil
}

func ListAPIKeys(db *gorm.DB, condition interface{}) ([]models.APIKey, error) {
	apiKeys := make([]models.APIKey, 0)
	if err := list(db, &models.APIKey{}, &apiKeys, condition); err != nil {
		return nil, err
	}

	return apiKeys, nil
}

func DeleteAPIKey(db *gorm.DB, condition interface{}) error {
	toDelete, err := GetAPIKey(db, condition)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return err
		}
	}

	if toDelete != nil {
		return remove(db, &models.APIKey{}, toDelete.ID)
	}

	return nil
}

func UserTokenSelector(db *gorm.DB, authorization string) *gorm.DB {
	return db.Where(
		"id = (?)",
		db.Model(&models.Token{}).Select("user_id").Where(&models.Token{Key: authorization[:models.TokenKeyLength]}),
	)
}

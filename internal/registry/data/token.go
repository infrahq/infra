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

func CreateAPIToken(db *gorm.DB, apiToken *models.APIToken) (*models.APIToken, error) {
	if apiToken.Key == "" {
		key, err := generate.CryptoRandom(models.APITokenLength)
		if err != nil {
			return nil, err
		}

		apiToken.Key = key
	}

	if err := add(db, &models.APIToken{}, apiToken, &models.APIToken{Name: apiToken.Name}); err != nil {
		return nil, err
	}

	return apiToken, nil
}

func GetAPIToken(db *gorm.DB, condition interface{}) (*models.APIToken, error) {
	var apiToken models.APIToken
	if err := get(db, &models.APIToken{}, &apiToken, condition); err != nil {
		return nil, err
	}

	return &apiToken, nil
}

func ListAPITokens(db *gorm.DB, condition interface{}) ([]models.APIToken, error) {
	apiTokens := make([]models.APIToken, 0)
	if err := list(db, &models.APIToken{}, &apiTokens, condition); err != nil {
		return nil, err
	}

	return apiTokens, nil
}

func DeleteAPIToken(db *gorm.DB, condition interface{}) error {
	toDelete, err := GetAPIToken(db, condition)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return err
		}
	}

	if toDelete != nil {
		return remove(db, &models.APIToken{}, toDelete.ID)
	}

	return nil
}

func UserTokenSelector(db *gorm.DB, authorization string) *gorm.DB {
	return db.Where(
		"id = (?)",
		db.Model(&models.Token{}).Select("user_id").Where(&models.Token{Key: authorization[:models.TokenKeyLength]}),
	)
}

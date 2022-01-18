package data

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/registry/models"
)

func CreateToken(db *gorm.DB, token *models.Token) error {
	if token.Key == "" {
		key := generate.MathRandom(models.TokenKeyLength)
		token.Key = key
	}

	if token.Secret == "" {
		generated, err := generate.CryptoRandom(models.TokenSecretLength)
		if err != nil {
			return err
		}

		token.Secret = generated
	}

	chksm := sha256.Sum256([]byte(token.Secret))
	token.Checksum = chksm[:]
	token.Expires = time.Now().Add(token.SessionDuration)

	if err := add(db, &models.Token{}, token, &models.Token{}); err != nil {
		return err
	}

	return nil
}

func UpdateToken(db *gorm.DB, token *models.Token, selector SelectorFunc) error {
	existing, err := GetToken(db, selector)
	if err != nil {
		return err
	}

	token.ID = existing.ID

	if token.Key == "" {
		token.Key = existing.Key
	}

	if token.Secret != "" {
		chksm := sha256.Sum256([]byte(token.Secret))
		token.Checksum = chksm[:]
	}

	if err := update(db, &models.Token{}, token, ByID(existing.ID)); err != nil {
		return err
	}

	return nil
}

func GetToken(db *gorm.DB, selector SelectorFunc) (*models.Token, error) {
	var token models.Token
	if err := get(db, &models.Token{}, &token, selector); err != nil {
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

func DeleteToken(db *gorm.DB, selector SelectorFunc) error {
	return remove(db, &models.Token{}, selector)
}

func CreateAPIToken(db *gorm.DB, apiToken *models.APIToken) error {
	if err := add(db, &models.APIToken{}, apiToken, &models.APIToken{}); err != nil {
		return fmt.Errorf("new api token: %w", err)
	}

	return nil
}

func UpdateAPIToken(db *gorm.DB, apiToken *models.APIToken) error {
	return update(db, &models.APIToken{}, apiToken, ByID(apiToken.ID))
}

func GetAPIToken(db *gorm.DB, selector SelectorFunc) (*models.APIToken, error) {
	var apiToken models.APIToken
	if err := get(db, &models.APIToken{}, &apiToken, selector); err != nil {
		return nil, err
	}

	return &apiToken, nil
}

func ListAPITokens(db *gorm.DB, condition interface{}) ([]models.APITokenTuple, error) {
	apiTokens := make([]models.APIToken, 0)
	if err := list(db, &models.APIToken{}, &apiTokens, condition); err != nil {
		return nil, err
	}

	apiTokenTuples := make([]models.APITokenTuple, 0)

	for _, apiTkn := range apiTokens {
		// need to get the token to find the expiry
		var tkn models.Token
		if err := get(db, &models.Token{}, &tkn, &models.Token{APITokenID: apiTkn.ID}); err != nil {
			return nil, err
		}

		apiTokenTuples = append(apiTokenTuples, models.APITokenTuple{APIToken: apiTkn, Token: tkn})
	}

	return apiTokenTuples, nil
}

func DeleteAPIToken(db *gorm.DB, id uuid.UUID) error {
	toDelete, err := GetAPIToken(db, ByID(id))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return fmt.Errorf("delete api token: %w", err)
		}

		return err
	}

	if err := DeleteToken(db, ByAPITokenID(toDelete.ID)); err != nil {
		return fmt.Errorf("delete token for api token: %w", err)
	}

	// proceed with deletion of API client even if there is no token for some reason

	return remove(db, &models.APIToken{}, toDelete.ID)
}

func UserTokenSelector(db *gorm.DB, authorization string) *gorm.DB {
	return db.Where(
		"id = (?)",
		db.Model(&models.Token{}).Select("user_id").Where(&models.Token{Key: authorization[:models.TokenKeyLength]}),
	)
}

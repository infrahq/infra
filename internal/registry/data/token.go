package data

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
)

func CreateToken(db *gorm.DB, token *models.Token) (*models.Token, error) {
	if err := models.Issue(token); err != nil {
		return nil, fmt.Errorf("issue token create: %w", err)
	}

	if err := add(db, &models.Token{}, token, &models.Token{}); err != nil {
		return nil, err
	}

	return token, nil
}

func CreateOrUpdateToken(db *gorm.DB, token *models.Token, condition interface{}) (*models.Token, error) {
	existing, err := GetToken(db, condition)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		token, err := CreateToken(db, token)
		if err != nil {
			return nil, err
		}

		return token, nil
	}

	if err := models.Issue(token); err != nil {
		return nil, fmt.Errorf("issue token update: %w", err)
	}

	if err := update(db, &models.Token{}, token, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	result, err := GetToken(db, db.Where(existing, "id"))
	if err != nil {
		return nil, fmt.Errorf("get token after update: %w", err)
	}

	// the secret needs to be set, because it is not stored in the database
	result.Secret = token.Secret

	return result, nil
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

func CreateAPIToken(db *gorm.DB, apiToken *models.APIToken, token *models.Token) (*models.APIToken, *models.Token, error) {
	if err := add(db, &models.APIToken{}, apiToken, &models.APIToken{}); err != nil {
		return nil, nil, fmt.Errorf("new api token: %w", err)
	}

	token.APITokenID = apiToken.ID
	token.SessionDuration = apiToken.TTL

	token, err := CreateToken(db, token)
	if err != nil {
		return nil, nil, fmt.Errorf("create api token issue: %w", err)
	}

	return apiToken, token, nil
}

func CreateOrUpdateAPIToken(db *gorm.DB, apiToken *models.APIToken, token *models.Token, condition interface{}) (*models.APIToken, error) {
	existing, err := GetAPIToken(db, condition)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		apiToken, _, err := CreateAPIToken(db, apiToken, token)
		if err != nil {
			return nil, err
		}

		return apiToken, nil
	}

	existingTkn, err := GetToken(db, &models.Token{APITokenID: existing.ID})
	if err != nil {
		return nil, fmt.Errorf("get old API token: %w", err)
	}

	if err := update(db, &models.APIToken{}, apiToken, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	token.APITokenID = existing.ID
	token.SessionDuration = apiToken.TTL

	_, err = CreateToken(db, token)
	if err != nil {
		return nil, fmt.Errorf("update api token issue: %w", err)
	}

	if err := DeleteToken(db, db.Where(existingTkn, "id")); err != nil {
		return nil, fmt.Errorf("remove old api token: %w", err)
	}

	return GetAPIToken(db, &models.APIToken{})
}

func GetAPIToken(db *gorm.DB, condition interface{}) (*models.APIToken, error) {
	var apiToken models.APIToken
	if err := get(db, &models.APIToken{}, &apiToken, condition); err != nil {
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

func DeleteAPIToken(db *gorm.DB, condition interface{}) error {
	toDelete, err := GetAPIToken(db, condition)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return fmt.Errorf("delete api token: %w", err)
		}

		return err
	}

	token, err := GetToken(db, &models.Token{APITokenID: toDelete.ID})
	if err == nil {
		if err := DeleteToken(db, token); err != nil {
			return fmt.Errorf("delete token for api token: %w", err)
		}
	} else if !errors.Is(err, internal.ErrNotFound) {
		return fmt.Errorf("get token for api: %w", err)
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

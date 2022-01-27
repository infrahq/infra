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
	"github.com/infrahq/infra/uid"
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

	if err := add(db, token); err != nil {
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

	if err := save(db, token); err != nil {
		return err
	}

	return nil
}

func GetToken(db *gorm.DB, selector SelectorFunc) (*models.Token, error) {
	return get[models.Token](db, selector)
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
	return removeAll[models.Token](db, selector)
}

func CreateProviderToken(db *gorm.DB, token *models.ProviderToken) error {
	return add(db, token)
}

func UpdateProviderToken(db *gorm.DB, token *models.ProviderToken) error {
	return update(db, token, ByID(token.ID))
}

func GetProviderToken(db *gorm.DB, selector SelectorFunc) (*models.ProviderToken, error) {
	result, err := get[models.ProviderToken](db, selector)
	if err != nil {
		return nil, fmt.Errorf("get provider token: %w", err)
	}

	return result, nil
}

func CreateAPIToken(db *gorm.DB, apiToken *models.APIToken) error {
	if err := add(db, apiToken); err != nil {
		return fmt.Errorf("new api token: %w", err)
	}

	return nil
}

func UpdateAPIToken(db *gorm.DB, apiToken *models.APIToken) error {
	return save(db, apiToken)
}

func GetAPIToken(db *gorm.DB, selector SelectorFunc) (*models.APIToken, error) {
	return get[models.APIToken](db, selector)
}

func ListAPITokens(db *gorm.DB, selectors ...SelectorFunc) ([]models.APITokenTuple, error) {
	ids := []uid.ID{}
	apiTokens, err := list[models.APIToken](db, selectors...)
	if err != nil {
		return nil, err
	}
	for _, t := range apiTokens {
		ids = append(ids, t.ID)
	}

	apiTokenTuples := make([]models.APITokenTuple, 0)
	tokens, err := list[models.Token](db, ByAPITokenIDs(ids))

	for _, apiTkn := range apiTokens {
		for _, tkn := range tokens {
			if tkn.APITokenID == apiTkn.ID {
				apiTokenTuples = append(apiTokenTuples, models.APITokenTuple{APIToken: apiTkn, Token: tkn})
				break
			}
		}
	}

	return apiTokenTuples, nil
}

func DeleteAPIToken(db *gorm.DB, id uid.ID) error {
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

	return delete[models.APIToken](db, toDelete.ID)
}

// IssueUserSessionToken creates an Infra session token for the specified user
func IssueUserSessionToken(db *gorm.DB, user *models.User, sessionDuration time.Duration) (string, error) {
	token := models.Token{
		UserID:          user.ID,
		SessionDuration: sessionDuration,
	}

	if err := CreateToken(db, &token); err != nil {
		return "", fmt.Errorf("create user session token: %w", err)
	}

	return token.SessionToken(), nil
}

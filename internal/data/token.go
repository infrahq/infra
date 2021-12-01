package data

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/generate"
)

const (
	TokenKeyLength    = 12
	TokenSecretLength = 24
	TokenLength       = TokenKeyLength + TokenSecretLength

	APIKeyLength = 24
)

type Token struct {
	Model

	User   User
	UserID uuid.UUID

	Key         string `gorm:"<-:create"`
	Secret      string `gorm:"-"`
	Checksum    []byte
	Permissions string `gorm:"<-:create"`

	SessionDuration time.Duration `gorm:"-"`

	Expires time.Time
}

func (t *Token) SessionToken() string {
	return t.Key + string(t.Secret)
}

type APIKey struct {
	Model

	Name string

	Key         string `gorm:"<-:create;index"`
	Permissions string `gorm:"<-:create"`
}

func (k *APIKey) ToAPI() (*api.InfraAPIKey, error) {
	result := api.InfraAPIKey{
		Id:      k.ID.String(),
		Created: k.CreatedAt.Unix(),

		Name: k.Name,
	}

	result.Permissions = append(result.Permissions, strings.Split(k.Permissions, " ")...)

	return &result, nil
}

func (k *APIKey) ToAPICreateResponse() (*api.InfraAPIKeyCreateResponse, error) {
	result := api.InfraAPIKeyCreateResponse{
		Id:      k.ID.String(),
		Created: k.CreatedAt.Unix(),

		Name: k.Name,
		Key:  k.Key,
	}

	result.Permissions = append(result.Permissions, strings.Split(k.Permissions, " ")...)

	return &result, nil
}

func (k *APIKey) FromAPICreateRequest(r *api.InfraAPIKeyCreateRequest) error {
	k.Name = r.Name

	permissions := make([]string, 0)
	for i := range r.Permissions {
		permissions = append(permissions, string(r.Permissions[i]))
	}

	k.Permissions = strings.Join(permissions, " ")

	return nil
}

func NewAPIKey(id string) (*APIKey, error) {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	return &APIKey{
		Model: Model{
			ID: uuid,
		},
	}, nil
}

func CreateToken(db *gorm.DB, token *Token) (*Token, error) {
	secret, err := generate.CryptoRandom(TokenSecretLength)
	if err != nil {
		return nil, err
	}

	chksm := sha256.Sum256([]byte(secret))
	token.Checksum = chksm[:]
	token.Secret = secret
	token.Key = generate.MathRandom(TokenKeyLength)
	token.Expires = time.Now().Add(token.SessionDuration)

	if err := add(db, &Token{}, token, &Token{}); err != nil {
		return nil, err
	}

	return token, nil
}

func GetToken(db *gorm.DB, condition interface{}) (*Token, error) {
	var token Token
	if err := get(db, &Token{}, &token, condition); err != nil {
		return nil, err
	}

	return &token, nil
}

func CheckTokenExpired(t *Token) error {
	if time.Now().After(t.Expires) {
		return fmt.Errorf("token expired")
	}

	return nil
}

func CheckTokenSecret(t *Token, authorization string) error {
	sum := sha256.Sum256([]byte(authorization[TokenKeyLength:]))
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
		return remove(db, &Token{}, toDelete.ID)
	}

	return nil
}

func CreateAPIKey(db *gorm.DB, apiKey *APIKey) (*APIKey, error) {
	if apiKey.Key == "" {
		key, err := generate.CryptoRandom(APIKeyLength)
		if err != nil {
			return nil, err
		}

		apiKey.Key = key
	}

	if err := add(db, &APIKey{}, apiKey, &APIKey{Name: apiKey.Name}); err != nil {
		return nil, err
	}

	return apiKey, nil
}

func GetAPIKey(db *gorm.DB, condition interface{}) (*APIKey, error) {
	var apiKey APIKey
	if err := get(db, &APIKey{}, &apiKey, condition); err != nil {
		return nil, err
	}

	return &apiKey, nil
}

func ListAPIKeys(db *gorm.DB, condition interface{}) ([]APIKey, error) {
	apiKeys := make([]APIKey, 0)
	if err := list(db, &APIKey{}, &apiKeys, condition); err != nil {
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
		return remove(db, &APIKey{}, toDelete.ID)
	}

	return nil
}

func UserTokenSelector(db *gorm.DB, authorization string) *gorm.DB {
	return db.Where(
		"id = (?)",
		db.Model(&Token{}).Select("user_id").Where(&Token{Key: authorization[:TokenKeyLength]}),
	)
}

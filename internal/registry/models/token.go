package models

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/generate"
)

const (
	TokenKeyLength    = 12
	TokenSecretLength = 24
	TokenLength       = TokenKeyLength + TokenSecretLength
)

type Token struct {
	Model

	User   User
	UserID uuid.UUID

	APIToken   APIToken
	APITokenID uuid.UUID

	Key      string `gorm:"<-;uniqueIndex:,where:deleted_at is NULL"`
	Secret   string `gorm:"-"`
	Checksum []byte

	SessionDuration time.Duration `gorm:"-"`

	Expires time.Time
}

// Issue sets the secret fields of a token, and the token expiry based on its lifetime
func Issue(token *Token) error {
	if token.Key == "" {
		key := generate.MathRandom(TokenKeyLength)
		token.Key = key
	}

	if token.Secret == "" {
		generated, err := generate.CryptoRandom(TokenSecretLength)
		if err != nil {
			return err
		}

		token.Secret = generated
	}

	chksm := sha256.Sum256([]byte(token.Secret))
	token.Checksum = chksm[:]
	token.Expires = time.Now().Add(token.SessionDuration)

	return nil
}

func KeyAndSecret(sessionToken string) (key, secret string) {
	return sessionToken[:TokenKeyLength], sessionToken[TokenKeyLength:]
}

func (t *Token) SessionToken() string {
	return t.Key + t.Secret
}

type APIToken struct {
	Model

	Name        string `gorm:"uniqueIndex:,where:deleted_at is NULL"`
	Permissions string
	TTL         time.Duration
}

type APITokenTuple struct {
	APIToken APIToken
	Token    Token
}

func (k *APIToken) ToAPI() *api.InfraAPIToken {
	ttl := k.TTL.String()

	return &api.InfraAPIToken{
		ID:      k.ID.String(),
		Created: k.CreatedAt.Unix(),

		Name:        k.Name,
		Permissions: strings.Split(k.Permissions, " "),
		Ttl:         &ttl,
	}
}

func (t *APITokenTuple) ToAPI() *api.InfraAPIToken {
	ttl := t.APIToken.TTL.String()
	exp := t.Token.Expires.Unix()

	return &api.InfraAPIToken{
		ID:      t.APIToken.ID.String(),
		Created: t.APIToken.CreatedAt.Unix(),

		Expires:     &exp,
		Name:        t.APIToken.Name,
		Permissions: strings.Split(t.APIToken.Permissions, " "),
		Ttl:         &ttl,
	}
}

func (k *APIToken) ToAPICreateResponse(tkn *Token) *api.InfraAPITokenCreateResponse {
	ttl := k.TTL.String()
	exp := tkn.Expires.Unix()

	return &api.InfraAPITokenCreateResponse{
		ID:      k.ID.String(),
		Created: k.CreatedAt.Unix(),

		Expires:     &exp,
		Name:        k.Name,
		Permissions: strings.Split(k.Permissions, " "),
		Ttl:         &ttl,

		Token: tkn.SessionToken(), // be cautious, this is a secret value
	}
}

func (k *APIToken) FromAPI(from interface{}, defaultSessionDuration time.Duration) error {
	if createRequest, ok := from.(*api.InfraAPITokenCreateRequest); ok {
		sessionDuration := defaultSessionDuration

		if createRequest.Ttl != nil && *createRequest.Ttl != "" {
			var err error

			sessionDuration, err = time.ParseDuration(*createRequest.Ttl)
			if err != nil {
				return fmt.Errorf("parse ttl: %w", err)
			}
		}

		k.Name = createRequest.Name
		k.Permissions = strings.Join(createRequest.Permissions, " ")
		k.TTL = sessionDuration

		return nil
	}

	return fmt.Errorf("unknown request")
}

func NewAPIToken(id string) (*APIToken, error) {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	return &APIToken{
		Model: Model{
			ID: uuid,
		},
	}, nil
}

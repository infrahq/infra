package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/infrahq/infra/internal/api"
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

	Key      string `gorm:"<-:create;index"`
	Secret   string `gorm:"-"`
	Checksum []byte

	SessionDuration time.Duration `gorm:"-"`

	Expires time.Time
}

func KeyAndSecret(sessionToken string) (key, secret string) {
	return sessionToken[:TokenKeyLength], sessionToken[TokenKeyLength:]
}

func (t *Token) SessionToken() string {
	return t.Key + string(t.Secret)
}

type APIToken struct {
	Model

	Name        string
	Permissions string
	TTL         time.Duration
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

func (k *APIToken) ToAPICreateResponse(tkn *Token) *api.InfraAPITokenCreateResponse {
	ttl := k.TTL.String()

	return &api.InfraAPITokenCreateResponse{
		ID:      k.ID.String(),
		Created: k.CreatedAt.Unix(),

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

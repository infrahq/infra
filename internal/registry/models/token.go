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
		ID:      k.ID.String(),
		Created: k.CreatedAt.Unix(),

		Name: k.Name,
	}

	result.Permissions = append(result.Permissions, strings.Split(k.Permissions, " ")...)

	return &result, nil
}

func (k *APIKey) ToAPICreateResponse() (*api.InfraAPIKeyCreateResponse, error) {
	result := api.InfraAPIKeyCreateResponse{
		ID:      k.ID.String(),
		Created: k.CreatedAt.Unix(),

		Name: k.Name,
		Key:  k.Key,
	}

	result.Permissions = append(result.Permissions, strings.Split(k.Permissions, " ")...)

	return &result, nil
}

func (k *APIKey) FromAPI(from interface{}) error {
	if createRequest, ok := from.(*api.InfraAPIKeyCreateRequest); ok {
		k.Name = createRequest.Name

		permissions := make([]string, 0)
		for i := range createRequest.Permissions {
			permissions = append(permissions, string(createRequest.Permissions[i]))
		}

		k.Permissions = strings.Join(permissions, " ")

		return nil
	}

	return fmt.Errorf("unknown request")
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

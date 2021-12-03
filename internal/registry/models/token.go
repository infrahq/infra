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

	APITokenLength = 24
)

type Token struct {
	Model

	User   User
	UserID uuid.UUID

	Key         string `gorm:"<-:create;uniqueIndex"` // ID
	Secret      string `gorm:"-"`
	Checksum    []byte
	Permissions string `gorm:"<-:create"`

	SessionDuration time.Duration `gorm:"-"`

	Expires time.Time
}

func (t *Token) SessionToken() string {
	return t.Key + string(t.Secret)
}

type APIToken struct {
	Model

	Name string

	Key         string `gorm:"<-:create;index"`
	Permissions string `gorm:"<-:create"`
}

func (k *APIToken) ToAPI() (*api.InfraAPIToken, error) {
	result := api.InfraAPIToken{
		ID:      k.ID.String(),
		Created: k.CreatedAt.Unix(),

		Name: k.Name,
	}

	result.Permissions = append(result.Permissions, strings.Split(k.Permissions, " ")...)

	return result
}

func (k *APIToken) ToAPICreateResponse() (*api.InfraAPITokenCreateResponse, error) {
	result := api.InfraAPITokenCreateResponse{
		ID:      k.ID.String(),
		Created: k.CreatedAt.Unix(),

		Name:  k.Name,
		Token: k.Key,
	}

	result.Permissions = append(result.Permissions, strings.Split(k.Permissions, " ")...)

	return result
}

func (k *APIToken) FromAPI(from interface{}) error {
	if createRequest, ok := from.(*api.InfraAPITokenCreateRequest); ok {
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

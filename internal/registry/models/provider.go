package models

import (
	"github.com/google/uuid"

	"github.com/infrahq/infra/internal/api"
)

type ProviderKind string

var ProviderKindOkta ProviderKind = "okta"

type Provider struct {
	Model

	Kind ProviderKind

	Domain       string
	ClientID     string
	ClientSecret EncryptedAtRest

	Okta ProviderOkta

	Users  []User  `gorm:"many2many:users_providers"`
	Groups []Group `gorm:"many2many:groups_providers"`
}

type ProviderOkta struct {
	Model

	APIToken EncryptedAtRest

	ProviderID uuid.UUID
}

func (p *Provider) ToAPI() api.Provider {
	result := api.Provider{
		ID:      p.ID.String(),
		Created: p.CreatedAt.Unix(),
		Updated: p.UpdatedAt.Unix(),

		Kind:     api.ProviderKind(p.Kind),
		Domain:   p.Domain,
		ClientID: p.ClientID,
	}

	return result
}

func NewProvider(id string) (*Provider, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	return &Provider{
		Model: Model{
			ID: uid,
		},
	}, nil
}

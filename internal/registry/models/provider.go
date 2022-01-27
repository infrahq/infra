package models

import (
	"github.com/infrahq/infra/internal/api"
)

type ProviderKind string

var ProviderKindOkta ProviderKind = "okta"

type Provider struct {
	Model

	Kind ProviderKind

	Domain       string `gorm:"uniqueIndex:,where:deleted_at is NULL"`
	ClientID     string
	ClientSecret EncryptedAtRest

	Users  []User  `gorm:"many2many:users_providers"`
	Groups []Group `gorm:"many2many:groups_providers"`
}

func (p *Provider) ToAPI() api.Provider {
	result := api.Provider{
		ID:      p.ID,
		Created: p.CreatedAt.Unix(),
		Updated: p.UpdatedAt.Unix(),

		Kind:     api.ProviderKind(p.Kind),
		Domain:   p.Domain,
		ClientID: p.ClientID,
	}

	return result
}

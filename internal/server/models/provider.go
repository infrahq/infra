package models

import (
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

const InternalInfraProviderName = "infra"

type Provider struct {
	Model

	Name         string `gorm:"uniqueIndex:,where:deleted_at is NULL" validate:"required"`
	URL          string
	ClientID     string
	ClientSecret EncryptedAtRest
	CreatedBy    uid.ID
}

func (p *Provider) ToAPI() *api.Provider {
	return &api.Provider{
		Name:    p.Name,
		ID:      p.ID,
		Created: api.Time(p.CreatedAt),
		Updated: api.Time(p.UpdatedAt),

		URL:      p.URL,
		ClientID: p.ClientID,
	}
}

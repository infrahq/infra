package models

import (
	"fmt"

	"github.com/infrahq/infra/uuid"

	"github.com/infrahq/infra/internal"
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

func (p *Provider) FromAPI(from interface{}) error {
	if request, ok := from.(*api.CreateProviderRequest); ok {
		p.Kind = ProviderKind(request.Kind)
		p.Domain = request.Domain
		p.ClientID = request.ClientID
		p.ClientSecret = EncryptedAtRest(request.ClientSecret)

		if request.Okta != nil {
			p.Okta = ProviderOkta{
				APIToken: EncryptedAtRest(request.Okta.APIToken),
			}
		}

		return nil
	}

	return fmt.Errorf("%w: unknown provider kind", internal.ErrBadRequest)
}

func NewProvider(id uuid.UUID) *Provider {
	return &Provider{
		Model: Model{
			ID: id,
		},
	}
}

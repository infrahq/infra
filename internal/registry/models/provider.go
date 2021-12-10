package models

import (
	"fmt"

	"github.com/google/uuid"

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
	if request, ok := from.(*api.ProviderRequest); ok {
		p.Kind = ProviderKind(request.Kind)
		p.Domain = request.Domain
		p.ClientID = request.ClientID
		p.ClientSecret = EncryptedAtRest(request.ClientSecret)

		if okta, ok := request.GetOktaOK(); ok {
			p.Okta = ProviderOkta{
				APIToken: EncryptedAtRest(okta.APIToken),
			}
		}

		return nil
	}

	return fmt.Errorf("unknown provider kind")
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

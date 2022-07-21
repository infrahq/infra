package models

import (
	"fmt"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

const InternalInfraProviderName = "infra"

type ProviderKind string

const (
	ProviderKindInfra  ProviderKind = "infra"
	ProviderKindOIDC   ProviderKind = "oidc"
	ProviderKindOkta   ProviderKind = "okta"
	ProviderKindAzure  ProviderKind = "azure"
	ProviderKindGoogle ProviderKind = "google"
)

func (p ProviderKind) String() string {
	return string(p)
}

var providerKindMap = map[string]ProviderKind{
	"":                          ProviderKindOIDC, // set empty provider kind to OIDC
	ProviderKindInfra.String():  ProviderKindInfra,
	ProviderKindOIDC.String():   ProviderKindOIDC,
	ProviderKindOkta.String():   ProviderKindOkta,
	ProviderKindAzure.String():  ProviderKindAzure,
	ProviderKindGoogle.String(): ProviderKindGoogle,
}

// ParseProviderKind validates that a string is valid kind then returns the ProviderKind
func ParseProviderKind(kind string) (ProviderKind, error) {
	providerKind, ok := providerKindMap[kind]
	if !ok {
		return ProviderKindOIDC, fmt.Errorf("%s is not a valid provider kind", kind)
	}

	return providerKind, nil
}

type Provider struct {
	Model

	Name         string       `gorm:"uniqueIndex:idx_providers_name,where:deleted_at is NULL" validate:"required"`
	Kind         ProviderKind `validate:"required"`
	URL          string
	ClientID     string
	ClientSecret EncryptedAtRest
	AuthURL      string
	Scopes       CommaSeparatedStrings
	CreatedBy    uid.ID

	// fields used to directly query an external API
	PrivateKey       EncryptedAtRest
	ClientEmail      string
	DomainAdminEmail string
}

func (p *Provider) ToAPI() *api.Provider {
	return &api.Provider{
		Name:    p.Name,
		ID:      p.ID,
		Created: api.Time(p.CreatedAt),
		Updated: api.Time(p.UpdatedAt),

		URL:      p.URL,
		ClientID: p.ClientID,
		Kind:     p.Kind.String(),
		AuthURL:  p.AuthURL,
		Scopes:   p.Scopes,
	}
}

package models

import (
	"fmt"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

const InternalInfraProviderName = "infra"

type ProviderKind string

const (
	InfraKind  ProviderKind = "infra"
	OIDCKind   ProviderKind = "oidc"
	OktaKind   ProviderKind = "okta"
	AzureKind  ProviderKind = "azure"
	GoogleKind ProviderKind = "google"
)

func (p ProviderKind) String() string {
	return string(p)
}

var providerKindMap = map[string]ProviderKind{
	"":                  OIDCKind, // set empty provider kind to OIDC
	InfraKind.String():  InfraKind,
	OIDCKind.String():   OIDCKind,
	OktaKind.String():   OktaKind,
	AzureKind.String():  AzureKind,
	GoogleKind.String(): GoogleKind,
}

// ParseProviderKind validates that a string is valid kind then returns the ProviderKind
func ParseProviderKind(kind string) (ProviderKind, error) {
	providerKind, ok := providerKindMap[kind]
	if !ok {
		return OIDCKind, fmt.Errorf("%s is not a valid provider kind", kind)
	}

	return providerKind, nil
}

type Provider struct {
	Model

	Name         string `gorm:"uniqueIndex:idx_providers_name,where:deleted_at is NULL" validate:"required"`
	URL          string
	ClientID     string
	ClientSecret EncryptedAtRest
	AuthURL      string
	Scopes       CommaSeparatedStrings
	Kind         ProviderKind
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
		Kind:     p.Kind.String(),
		AuthURL:  p.AuthURL,
		Scopes:   p.Scopes,
	}
}

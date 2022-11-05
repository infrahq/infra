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
	OrganizationMember

	Name           string
	URL            string
	ClientID       string
	ClientSecret   EncryptedAtRest
	CreatedBy      uid.ID
	Kind           ProviderKind
	AuthURL        string
	Scopes         CommaSeparatedStrings
	AllowedDomains CommaSeparatedStrings // the email domains that are allowed to login using this provider

	// fields used to directly query an external API
	PrivateKey       EncryptedAtRest
	ClientEmail      string
	DomainAdminEmail string

	SocialLogin bool // this is a social login client that is owned by Infra not the orgs themselves
	Managed     bool // this is a client that the organization admin cannot edit, it is owned by Infra as as social login
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
		Managed:  p.Managed,
		// TODO: specify allowed domains here once login providers are a separate endpoint (#3599)
	}
}

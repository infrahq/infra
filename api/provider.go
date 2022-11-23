package api

import (
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

// ProviderAPICredentials contain sensitive fields, it should not be sent on a response
type ProviderAPICredentials struct {
	PrivateKey       PEM    `json:"privateKey" example:"-----BEGIN PRIVATE KEY-----\nMIIDNTCCAh2gAwIBAgIRALRetnpcTo9O3V2fAK3ix+c\n-----END PRIVATE KEY-----\n"`
	ClientEmail      string `json:"clientEmail"`
	DomainAdminEmail string `json:"domainAdminEmail"`
}

func (r ProviderAPICredentials) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Email("clientEmail", r.ClientEmail),
		validate.Email("domainAdminEmail", r.DomainAdminEmail),
	}
}

type Provider struct {
	ID       uid.ID   `json:"id" note:"Provider ID"`
	Name     string   `json:"name" example:"okta" note:"Name of the provider"`
	Created  Time     `json:"created"`
	Updated  Time     `json:"updated"`
	URL      string   `json:"url" example:"infrahq.okta.com" note:"URL of the Infra Server"`
	ClientID string   `json:"clientID" example:"0oapn0qwiQPiMIyR35d6" note:"Client ID for the OIDC provider"`
	Kind     string   `json:"kind" example:"oidc" note:"Kind of provider"`
	AuthURL  string   `json:"authURL" example:"https://example.com/oauth2/v1/authorize" note:"Authorize endpoint for the OIDC provider"`
	Scopes   []string `json:"scopes" example:"['openid', 'email']" note:"Scopes set in the OIDC provider configuration"`
}

type CreateProviderRequest struct {
	Name           string                  `json:"name" example:"okta"`
	URL            string                  `json:"url" example:"infrahq.okta.com"`
	ClientID       string                  `json:"clientID" example:"0oapn0qwiQPiMIyR35d6"`
	ClientSecret   string                  `json:"clientSecret" example:"jmda5eG93ax3jMDxTGrbHd_TBGT6kgNZtrCugLbU"`
	AllowedDomains []string                `json:"allowedDomains" example:"['example.com', 'infrahq.com']"`
	Kind           string                  `json:"kind" example:"oidc"`
	API            *ProviderAPICredentials `json:"api"`
}

var kinds = []string{"oidc", "okta", "azure", "google"}

func ValidateAllowedDomains(value []string) validate.StringSliceRule {
	return validate.StringSliceRule{
		Value:     value,
		Name:      "allowedDomains",
		MaxLength: 20,
		ItemRule: validate.StringRule{
			Name:      "allowedDomains.values",
			MaxLength: 254,
			CharacterRanges: []validate.CharRange{
				validate.AlphabetLower,
				validate.AlphabetUpper,
				validate.Numbers,
				validate.Dash,
				validate.Dot,
				validate.Underscore,
			},
			FirstCharacterRange: validate.AlphaNumeric,
		},
	}
}

func (r CreateProviderRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		ValidateName(r.Name),
		validate.Required("url", r.URL),
		validate.Required("clientID", r.ClientID),
		validate.Required("clientSecret", r.ClientSecret),
		validate.Enum("kind", r.Kind, kinds),
		ValidateAllowedDomains(r.AllowedDomains),
	}
}

type PatchProviderRequest struct {
	ID           uid.ID `uri:"id" json:"-"`
	Name         string `json:"name" example:"okta"`
	ClientSecret string `json:"clientSecret" example:"jmda5eG93ax3jMDxTGrbHd_TBGT6kgNZtrCugLbU"`
}

type UpdateProviderRequest struct {
	ID             uid.ID                  `uri:"id" json:"-"`
	Name           string                  `json:"name" example:"okta"`
	URL            string                  `json:"url" example:"infrahq.okta.com"`
	ClientID       string                  `json:"clientID" example:"0oapn0qwiQPiMIyR35d6"`
	ClientSecret   string                  `json:"clientSecret" example:"jmda5eG93ax3jMDxTGrbHd_TBGT6kgNZtrCugLbU"`
	AllowedDomains []string                `json:"allowedDomains" example:"['example.com', 'infrahq.com']"`
	Kind           string                  `json:"kind" example:"oidc"`
	API            *ProviderAPICredentials `json:"api"`
}

func (r UpdateProviderRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		ValidateName(r.Name),
		validate.Required("id", r.ID),
		validate.Required("name", r.Name),
		validate.Required("url", r.URL),
		validate.Required("clientID", r.ClientID),
		validate.Required("clientSecret", r.ClientSecret),
		validate.Enum("kind", r.Kind, kinds),
		ValidateAllowedDomains(r.AllowedDomains),
	}
}

type ListProvidersRequest struct {
	Name string `form:"name" example:"okta" note:"Name of the provider"`
	PaginationRequest
}

func (r ListProvidersRequest) ValidationRules() []validate.ValidationRule {
	// no-op ValidationRules implementation so that the rules from the
	// embedded PaginationRequest struct are not applied twice.
	return nil
}

func (req ListProvidersRequest) SetPage(page int) Paginatable {
	req.PaginationRequest.Page = page

	return req
}

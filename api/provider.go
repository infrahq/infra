package api

import (
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type Provider struct {
	ID       uid.ID   `json:"id"`
	Name     string   `json:"name" example:"okta"`
	Created  Time     `json:"created"`
	Updated  Time     `json:"updated"`
	URL      string   `json:"url" example:"infrahq.okta.com"`
	ClientID string   `json:"clientID" example:"0oapn0qwiQPiMIyR35d6"`
	Kind     string   `json:"kind" example:"oidc"`
	AuthURL  string   `json:"authURL" example:"https://example.com/oauth2/v1/authorize"`
	Scopes   []string `json:"scopes" example:"['openid', 'email']"`
}

type CreateProviderRequest struct {
	Name         string `json:"name" example:"okta"`
	URL          string `json:"url" example:"infrahq.okta.com"`
	ClientID     string `json:"clientID" example:"0oapn0qwiQPiMIyR35d6"`
	ClientSecret string `json:"clientSecret" example:"jmda5eG93ax3jMDxTGrbHd_TBGT6kgNZtrCugLbU"`
	Kind         string `json:"kind" example:"oidc"`
}

var kinds = []string{"oidc", "okta", "azure", "google"}

func (r CreateProviderRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("name", r.Name),
		validate.Required("url", r.URL),
		validate.Required("clientID", r.ClientID),
		validate.Required("clientSecret", r.ClientSecret),
		validate.Enum("kind", r.Kind, kinds),
	}
}

type UpdateProviderRequest struct {
	ID           uid.ID `uri:"id" json:"-"`
	Name         string `json:"name" example:"okta"`
	URL          string `json:"url" example:"infrahq.okta.com"`
	ClientID     string `json:"clientID" example:"0oapn0qwiQPiMIyR35d6"`
	ClientSecret string `json:"clientSecret" example:"jmda5eG93ax3jMDxTGrbHd_TBGT6kgNZtrCugLbU"`
	Kind         string `json:"kind" example:"oidc"`
}

func (r UpdateProviderRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("id", r.ID),
		validate.Required("name", r.Name),
		validate.Required("url", r.URL),
		validate.Required("clientID", r.ClientID),
		validate.Required("clientSecret", r.ClientSecret),
		validate.Enum("kind", r.Kind, kinds),
	}
}

type ListProvidersRequest struct {
	Name string `form:"name" example:"okta"`
	PaginationRequest
}

func (r ListProvidersRequest) ValidationRules() []validate.ValidationRule {
	// no-op ValidationRules implementation so that the rules from the
	// embedded PaginationRequest struct are not applied twice.
	return nil
}

func (req ListProvidersRequest) GetPaginationRequest() PaginationRequest {
	return req.PaginationRequest
}

func (req ListProvidersRequest) SetPage(page int) Paginatable {
	req.PaginationRequest.Page = page

	return req
}

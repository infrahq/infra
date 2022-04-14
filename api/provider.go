package api

import (
	"github.com/infrahq/infra/uid"
)

type Provider struct {
	ID       uid.ID `json:"id"`
	Name     string `json:"name" example:"okta"`
	Created  Time   `json:"created"`
	Updated  Time   `json:"updated"`
	URL      string `json:"url" validate:"fqdn,required" example:"infrahq.okta.com"`
	ClientID string `json:"clientID" validate:"required" example:"0oapn0qwiQPiMIyR35d6"`
}

type CreateProviderRequest struct {
	Name         string `json:"name" validate:"required" example:"okta"`
	URL          string `json:"url" validate:"fqdn,required" example:"infrahq.okta.com"`
	ClientID     string `json:"clientID" validate:"required" example:"0oapn0qwiQPiMIyR35d6"`
	ClientSecret string `json:"clientSecret" validate:"required" example:"jmda5eG93ax3jMDxTGrbHd_TBGT6kgNZtrCugLbU"`
}

type UpdateProviderRequest struct {
	ID           uid.ID `uri:"id" json:"-" validate:"required"`
	Name         string `json:"name" validate:"fqdn,required" example:"okta"`
	URL          string `json:"url" example:"infrahq.okta.com"`
	ClientID     string `json:"clientID" validate:"required" example:"0oapn0qwiQPiMIyR35d6"`
	ClientSecret string `json:"clientSecret" validate:"required" example:"jmda5eG93ax3jMDxTGrbHd_TBGT6kgNZtrCugLbU"`
}

type ListProvidersRequest struct {
	Name string `form:"name" example:"okta"`
}

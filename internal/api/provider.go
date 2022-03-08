package api

import "github.com/infrahq/infra/uid"

type Provider struct {
	ID       uid.ID `json:"id" swaggertype:"string" example:"3VGSwuC7zf"`
	Name     string `json:"name" example:"okta"`
	Created  int64  `json:"created" example:"1646427487"`
	Updated  int64  `json:"updated" example:"1646427981"`
	URL      string `json:"url" validate:"fqdn,required" example:"infrahq.okta.com"`
	ClientID string `json:"clientID" validate:"required" example:"0oapn0qwiQPiMIyR35d6"`
}

type CreateProviderRequest struct {
	Name         string `json:"name" validate:"required" example:"okta"`
	URL          string `json:"url" validate:"required" example:"infrahq.okta.com"`
	ClientID     string `json:"clientID" example:"0oapn0qwiQPiMIyR35d6"`
	ClientSecret string `json:"clientSecret" example:"jmda5eG93ax3jMDxTGrbHd_TBGT6kgNZtrCugLbU"`
}

type UpdateProviderRequest struct {
	ID           uid.ID `uri:"id" json:"-" validate:"required" swaggertype:"string" example:"3VGSwuC7zf"`
	Name         string `json:"name" example:"okta"`
	URL          string `json:"url" example:"infrahq.okta.com"`
	ClientID     string `json:"clientID" example:"0oapn0qwiQPiMIyR35d6"`
	ClientSecret string `json:"clientSecret" example:"jmda5eG93ax3jMDxTGrbHd_TBGT6kgNZtrCugLbU"`
}

type ListProvidersRequest struct {
	Name string `form:"name" example:"okta"`
}

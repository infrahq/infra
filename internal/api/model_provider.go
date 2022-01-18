package api

import "github.com/infrahq/infra/uid"

type Provider struct {
	ID       uid.ID `json:"id"`
	Name     string `json:"name"`
	Created  int64  `json:"created"`
	Updated  int64  `json:"updated"`
	URL      string `json:"url" validate:"fqdn,required"`
	ClientID string `json:"clientID" validate:"required"`
}

type CreateProviderRequest struct {
	Name         string `json:"name" validate:"required"`
	URL          string `json:"url" validate:"required"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
}

type UpdateProviderRequest struct {
	ID           uid.ID `uri:"id" json:"-" validate:"required"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
}

type ListProvidersRequest struct {
	Name string `form:"name"`
}

package api

import "github.com/infrahq/infra/uid"

type LoginRequestOIDC struct {
	ProviderID  uid.ID `json:"providerID" validate:"required"`
	RedirectURL string `json:"redirectURL" validate:"required"`
	Code        string `json:"code" validate:"required"`
}

type LoginRequestPasswordCredentials struct {
	Name     string `json:"name" validate:"required_without=Email"`
	Email    string `json:"email" validate:"required_without=Name"` // #1825: remove, this is for migration
	Password string `json:"password" validate:"required"`
}

type LoginRequest struct {
	AccessKey           string                           `json:"accessKey" validate:"excluded_with=OIDC,excluded_with=PasswordCredentials"`
	PasswordCredentials *LoginRequestPasswordCredentials `json:"passwordCredentials" validate:"excluded_with=OIDC,excluded_with=AccessKey"`
	OIDC                *LoginRequestOIDC                `json:"oidc" validate:"excluded_with=KeyExchange,excluded_with=PasswordCredentials"`
}

type LoginResponse struct {
	ID                     uid.ID `json:"id"`
	Name                   string `json:"name"`
	AccessKey              string `json:"accessKey"`
	PasswordUpdateRequired bool   `json:"passwordUpdateRequired,omitempty"`
	Expires                Time   `json:"expires"`
}

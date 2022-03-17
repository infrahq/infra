package api

import "github.com/infrahq/infra/uid"

type LoginRequestOIDC struct {
	ProviderID  uid.ID `json:"providerID" validate:"required"`
	RedirectURL string `json:"redirectURL" validate:"required"`
	Code        string `json:"code" validate:"required"`
}

type LoginRequest struct {
	OIDC      *LoginRequestOIDC `json:"oidc" validate:"excluded_with=KeyExchange"`
	AccessKey string            `json:"accessKey"  validate:"excluded_with=OIDC,excluded_with=Password"`
	Email     string            `json:"email" validate:"required_with=Password,excluded_with=OIDC,excluded_with=AccessKey"`
	Password  string            `json:"password"  validate:"required_with=Email,excluded_with=OIDC,excluded_with=AccessKey"`
}

type LoginResponse struct {
	PolymorphicID          uid.PolymorphicID `json:"polymorphicId"`
	Name                   string            `json:"name"`
	AccessKey              string            `json:"accessKey"`
	PasswordUpdateRequired bool              `json:"passwordUpdateRequired,omitempty"`
}

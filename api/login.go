package api

import "github.com/infrahq/infra/uid"

type LoginRequestOIDC struct {
	ProviderID  uid.ID `json:"providerID" validate:"required"`
	RedirectURL string `json:"redirectURL" validate:"required"`
	Code        string `json:"code" validate:"required"`
}

type LoginRequestCredentials struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password"  validate:"required"`
}

type LoginRequest struct {
	OIDC        *LoginRequestOIDC        `json:"oidc" validate:"excluded_with=KeyExchange,excluded_with=Credentials"`
	AccessKey   string                   `json:"accessKey"  validate:"excluded_with=OIDC,excluded_with=Credentials"`
	Credentials *LoginRequestCredentials `json:"credentials" validate:"excluded_with=OIDC,excluded_with=AccessKey"`
}

type LoginResponse struct {
	PolymorphicID          uid.PolymorphicID `json:"polymorphicId"`
	Name                   string            `json:"name"`
	AccessKey              string            `json:"accessKey"`
	PasswordUpdateRequired bool              `json:"passwordUpdateRequired,omitempty"`
}

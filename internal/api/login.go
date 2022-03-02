package api

import "github.com/infrahq/infra/uid"

type LoginRequestOIDC struct {
	ProviderID  uid.ID `json:"providerID" validate:"required"`
	RedirectURL string `json:"redirectURL" validate:"required"`
	Code        string `json:"code" validate:"required"`
}

type LoginRequestKeyExchange struct {
	AccessKey string `json:"accessKey" validate:"required"`
}

type LoginRequest struct {
	OIDC        *LoginRequestOIDC        `json:"oidc" validate:"excluded_with=KeyExchange"`
	KeyExchange *LoginRequestKeyExchange `json:"exchange" validate:"excluded_with=OIDC"`
}

type LoginResponse struct {
	PolymorphicID uid.PolymorphicID `json:"polymorphicId"`
	Name          string            `json:"name"`
	AccessKey     string            `json:"accessKey"`
}

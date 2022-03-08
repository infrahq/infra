package api

import "github.com/infrahq/infra/uid"

type LoginRequestOIDC struct {
	ProviderID  uid.ID `json:"providerID" validate:"required" swaggertype:"string"`
	RedirectURL string `json:"redirectURL" validate:"required"`
	Code        string `json:"code" validate:"required"`
}

type LoginRequest struct {
	OIDC      *LoginRequestOIDC `json:"oidc" validate:"excluded_with=KeyExchange"`
	AccessKey string            `json:"accessKey"  validate:"excluded_with=OIDC"`
}

type LoginResponse struct {
	PolymorphicID uid.PolymorphicID `json:"polymorphicId" swaggertype:"string"`
	Name          string            `json:"name"`
	AccessKey     string            `json:"accessKey"`
}

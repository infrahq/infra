package api

import "github.com/infrahq/infra/uid"

type LoginRequest struct {
	ProviderID uid.ID `json:"providerID" validate:"required"`
	Code       string `json:"code" validate:"required"`
}

type LoginResponse struct {
	ID    uid.ID `json:"id"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

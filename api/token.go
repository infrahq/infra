package api

import (
	"github.com/infrahq/infra/uid"
)

type CreateTokenRequest struct {
	UserID uid.ID `json:"userID" validate:"required"`
}

type CreateTokenResponse struct {
	Expires Time   `json:"expires"`
	Token   string `json:"token"`
}

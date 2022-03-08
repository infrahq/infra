package api

import (
	"time"

	"github.com/infrahq/infra/uid"
)

type CreateTokenRequest struct {
	UserID uid.ID `json:"userID" validate:"required" swaggertype:"string"`
}

type CreateTokenResponse struct {
	Expires time.Time `json:"expires"`
	Token   string    `json:"token"`
}

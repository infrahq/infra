package api

import (
	"github.com/infrahq/infra/uid"
)

type Organization struct {
	ID      uid.ID `json:"id"`
	Name    string `json:"name"`
	Created Time   `json:"created"`
	Updated Time   `json:"updated"`
}

type CreateOrganizationRequest struct {
	Name string `json:"name" validate:"required"`
}

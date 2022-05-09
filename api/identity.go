package api

import (
	"github.com/infrahq/infra/uid"
)

type GetIdentityRequest struct {
	ID IDOrSelf `uri:"id" validate:"required"`
}

type Identity struct {
	ID         uid.ID `json:"id"`
	Created    Time   `json:"created"`
	Updated    Time   `json:"updated"`
	LastSeenAt Time   `json:"lastSeenAt"`
	Name       string `json:"name" validate:"required"`
}

type ListIdentitiesRequest struct {
	Name string   `form:"name"`
	IDs  []uid.ID `form:"ids"`
}

type CreateIdentityRequest struct {
	Name               string `json:"name" validate:"required"`
	SetOneTimePassword bool   `json:"setOneTimePassword"`
}

type CreateIdentityResponse struct {
	ID              uid.ID `json:"id"`
	Name            string `json:"name" validate:"required"`
	OneTimePassword string `json:"oneTimePassword,omitempty"`
}

type UpdateIdentityRequest struct {
	ID       uid.ID `uri:"id" json:"-" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

package api

import (
	"github.com/infrahq/infra/uid"
)

type Identity struct {
	ID         uid.ID `json:"id"`
	Created    Time   `json:"created"`
	Updated    Time   `json:"updated"`
	LastSeenAt Time   `json:"lastSeenAt"`
	Name       string `json:"name" validate:"required"`
	Kind       string `json:"kind" validate:"required,oneof=user machine"`
}

type ListIdentitiesRequest struct {
	Name            string `form:"name"`
	IncludeUnlinked bool   `form:"includeUnlinked" note:"Show identities that exist in grants but are not linked to an identity provider"`
}

type CreateIdentityRequest struct {
	Name         string `json:"name" validate:"required"`
	Kind         string `json:"kind" validate:"required,oneof=user machine"`
	ProviderName string `json:"providerName"`
}

type UpdateIdentityRequest struct {
	ID       uid.ID `uri:"id" json:"-" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type CreateIdentityResponse struct {
	ID              uid.ID `json:"id"`
	Name            string `json:"name" validate:"required"`
	ProviderName    string `json:"providerName,omitempty" `
	OneTimePassword string `json:"oneTimePassword,omitempty"`
}

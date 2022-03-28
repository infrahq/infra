package api

import (
	"github.com/infrahq/infra/uid"
)

type User struct {
	ID         uid.ID `json:"id"`
	Email      string `json:"email" validate:"email,required"`
	Created    Time   `json:"created"`
	Updated    Time   `json:"updated"`
	LastSeenAt Time   `json:"lastSeenAt"`
	ProviderID uid.ID `json:"providerID"`
}

type ListUsersRequest struct {
	Email      string `form:"email"`
	ProviderID uid.ID `form:"provider_id"`
}

type CreateUserRequest struct {
	Email      string `json:"email" validate:"email,required"`
	ProviderID uid.ID `json:"providerID" validate:"required"`
}

type UpdateUserRequest struct {
	ID       uid.ID `uri:"id" json:"-" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type CreateUserResponse struct {
	ID              uid.ID `json:"id"`
	Email           string `json:"email" validate:"email,required"`
	ProviderID      uid.ID `json:"providerID" validate:"required"`
	OneTimePassword string `json:"oneTimePassword,omitempty"`
}

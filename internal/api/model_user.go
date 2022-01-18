package api

import "github.com/infrahq/infra/uid"

type User struct {
	ID          uid.ID   `json:"id"`
	Email       string   `json:"email" validate:"email,required"`
	Created     int64    `json:"created"`
	Updated     int64    `json:"updated"`
	LastSeenAt  int64    `json:"lastSeenAt"`
	ProviderID  uid.ID   `json:"providerID"`
	Permissions []string `json:"permissions"`
}

type ListUsersRequest struct {
	Email      string `form:"email"`
	ProviderID uid.ID `form:"provider_id"`
}

type CreateUserRequest struct {
	Email      string `json:"email" validate:"email,required"`
	ProviderID uid.ID `json:"providerID" validate:"required"`
}

package api

import "github.com/infrahq/infra/uid"

// User struct for User
type User struct {
	ID    uid.ID `json:"id"`
	Email string `json:"email" validate:"email,required"`
	// created time in seconds since 1970-01-01
	Created int64 `json:"created"`
	// updated time in seconds since 1970-01-01
	Updated int64 `json:"updated"`
	// timestamp of this user's last interaction with Infra in seconds since 1970-01-01
	LastSeenAt int64      `json:"lastSeenAt"`
	Groups     []Group    `json:"groups,omitempty"`
	Grants     []Grant    `json:"grants,omitempty"`
	Providers  []Provider `json:"providers,omitempty"`
}

type ListUsersRequest struct {
	Email string `form:"email"`
}

type Resource struct {
	ID uid.ID `uri:"id" validate:"required"`
}

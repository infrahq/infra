package api

import (
	"github.com/infrahq/infra/uid"
)

type Group struct {
	ID      uid.ID `json:"id"`
	Name    string `json:"name"`
	Created Time   `json:"created"`
	Updated Time   `json:"updated"`
}

type ListGroupsRequest struct {
	PaginationRequest
	// Name filters the results to only the group matching this name.
	Name string `form:"name"`
	// UserID filters the results to only groups where this user is a member.
	UserID uid.ID `form:"userID"`
}

type CreateGroupRequest struct {
	Name string `json:"name" validate:"required"`
}

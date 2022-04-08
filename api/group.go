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
	Name string `form:"name"`
}

type CreateGroupRequest struct {
	Name string `json:"name" validate:"required"`
}

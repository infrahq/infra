package api

import (
	"github.com/infrahq/infra/uid"
)

type Grant struct {
	ID uid.ID `json:"id"`

	Created Time `json:"created"`
	Updated Time `json:"updated"`

	Identity  uid.ID `json:"identity,omitempty"`
	Group     uid.ID `json:"group,omitempty"`
	Privilege string `json:"privilege" note:"a role or permission"`
	Resource  string `json:"resource" note:"a resource name in Infra's Universal Resource Notation"`
}

type ListGrantsRequest struct {
	Identity  uid.ID `form:"identity" validate:"excluded_with=Group"`
	Group     uid.ID `form:"group" validate:"excluded_with=Identity"`
	Resource  string `form:"resource" example:"production"`
	Privilege string `form:"privilege" example:"view"`
}

type CreateGrantRequest struct {
	Identity  uid.ID `json:"identity" validate:"required_without=Group"`
	Group     uid.ID `json:"group" validate:"required_without=Identity"`
	Privilege string `json:"privilege" validate:"required" example:"view" note:"a role or permission"`
	Resource  string `json:"resource" validate:"required" example:"production" note:"a resource name in Infra's Universal Resource Notation"`
}

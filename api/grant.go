package api

import (
	"github.com/infrahq/infra/uid"
)

type Grant struct {
	ID uid.ID `json:"id"`

	Created   int64  `json:"created"`    // created time in seconds since 1970-01-01 00:00:00 UTC
	CreatedBy uid.ID `json:"created_by"` // id of user who created the grant
	Updated   int64  `json:"updated"`    // updated time in seconds since 1970-01-01 00:00:00 UTC

	Identity  uid.PolymorphicID `json:"identity"`
	Privilege string            `json:"privilege"` // role or permission
	Resource  string            `json:"resource"`  // Universal Resource Notation

	ExpiresAt *int64 `json:"expires_at"` // time this grant expires at in seconds since 1970-01-01 00:00:00 UTC
}

type ListGrantsRequest struct {
	Identity  uid.PolymorphicID `form:"identity"`
	Resource  string            `form:"resource" example:"kubernetes.production"`
	Privilege string            `form:"privilege" example:"view"`
}

type CreateGrantRequest struct {
	Identity  uid.PolymorphicID `json:"identity" validate:"required"`
	Resource  string            `json:"resource" validate:"required" example:"kubernetes.production"`
	Privilege string            `json:"privilege" validate:"required" example:"view"`
}

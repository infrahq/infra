package api

import (
	"github.com/infrahq/infra/uid"
)

type Grant struct {
	ID uid.ID `json:"id"`

	Created   int64  `json:"created"` // created time in seconds since 1970-01-01 00:00:00 UTC
	CreatedBy uid.ID `json:"created_by" note:"id of user who created the grant"`
	Updated   int64  `json:"updated"` // updated time in seconds since 1970-01-01 00:00:00 UTC

	Subject   uid.PolymorphicID `json:"subject" note:"a polymorphic field primarily expecting an user, or group ID"`
	Privilege string            `json:"privilege" note:"a role or permission"`
	Resource  string            `json:"resource" note:"a resource name in Infra's Universal Resource Notation"`

	ExpiresAt *int64 `json:"expires_at" note:"grant expires after this time"`
}

type ListGrantsRequest struct {
	Subject   uid.PolymorphicID `form:"subject"`
	Resource  string            `form:"resource" example:"kubernetes.production"`
	Privilege string            `form:"privilege" example:"view"`
}

type CreateGrantRequest struct {
	Subject   uid.PolymorphicID `json:"subject" validate:"required" note:"a polymorphic field primarily expecting an user, or group ID"`
	Privilege string            `json:"privilege" validate:"required" example:"view" note:"a role or permission"`
	Resource  string            `json:"resource" validate:"required" example:"kubernetes.production" note:"a resource name in Infra's Universal Resource Notation"`
}

package api

import (
	"github.com/infrahq/infra/uid"
)

type Grant struct {
	ID uid.ID `json:"id"`

	Created   Time   `json:"created"`
	CreatedBy uid.ID `json:"created_by" note:"id of the identity that created the grant"`
	Updated   Time   `json:"updated"`

	Subject   uid.PolymorphicID `json:"subject" note:"a polymorphic field expecting an identity or group ID"`
	Privilege string            `json:"privilege" note:"a role or permission"`
	Resource  string            `json:"resource" note:"a resource name in Infra's Universal Resource Notation"`
}

type ListGrantsRequest struct {
	Subject   uid.PolymorphicID `form:"subject"`
	Resource  string            `form:"resource" example:"production"`
	Privilege string            `form:"privilege" example:"view"`
}

type CreateGrantRequest struct {
	Subject   uid.PolymorphicID `json:"subject" validate:"required" note:"a polymorphic field expecting an identity or group ID"`
	Privilege string            `json:"privilege" validate:"required" example:"view" note:"a role or permission"`
	Resource  string            `json:"resource" validate:"required" example:"production" note:"a resource name in Infra's Universal Resource Notation"`
}

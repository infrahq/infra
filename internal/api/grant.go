package api

import (
	"github.com/infrahq/infra/uid"
)

type Grant struct {
	ID        uid.ID            `json:"id" swaggertype:"string" example:"42Fjr45Kqm"`
	Created   int64             `json:"created" example:"1646359105"`                         // created time in seconds since 1970-01-01 00:00:00 UTC
	CreatedBy uid.ID            `json:"created_by" swaggertype:"string" example:"42MobtmDmU"` // id of user who created the grant
	Updated   int64             `json:"updated" example:"1646359105"`                         // updated time in seconds since 1970-01-01 00:00:00 UTC
	Identity  uid.PolymorphicID `json:"identity" swaggertype:"string" example:"u:42MobtmDmU"`
	Privilege string            `json:"privilege" example:"admin"`                // role or permission
	Resource  string            `json:"resource" example:"kubernetes.production"` // destination resource
	ExpiresAt *int64            `json:"expires_at" example:"1646365215"`          // time this grant expires at in seconds since 1970-01-01 00:00:00 UTC
}

type ListGrantsRequest struct {
	Identity  uid.PolymorphicID `form:"identity" swaggertype:"string" example:"u:42MobtmDmU"`
	Resource  string            `form:"resource" example:"kubernetes.production"`
	Privilege string            `form:"privilege" example:"admin"`
}

type CreateGrantRequest struct {
	Identity  uid.PolymorphicID `json:"identity" validate:"required" swaggertype:"string" example:"u:42MobtmDmU"`
	Resource  string            `json:"resource" validate:"required" example:"kubernetes.production"`
	Privilege string            `json:"privilege" validate:"required" example:"admin"`
}

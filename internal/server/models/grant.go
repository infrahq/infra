package models

import (
	"time"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

const (
	InfraAdminRole     = "admin"
	InfraUserRole      = "user"
	InfraConnectorRole = "connector"
)

const (
	CreatedBySystem = 0
	CreatedByConfig = 1
)

// Grant is a lean tuple of identity <-> privilege <-> resource (URN) relationships.
// bloat should be avoided here since this model is going to be used heavily.
//
// Identity
// 		Identity is a string specifying a user, group, the name of a role, or another grant
// 			- a user: u:E97WmsYfvo
// 			- a group: g:CCoJ1ornpf
// 			- a role: ?
// 			- a grant: ?
// Privilege
// 		Privilege is a predicate that describes what sort of access the identity has to the resource
// URN
// 		URN is Universal Resource Notation.
// Expiry
//    time you want the grant to expire at
//
// Defining
type Grant struct {
	Model

	Identity  uid.PolymorphicID `validate:"required"`
	Privilege string            `validate:"required"` // role or permission
	Resource  string            `validate:"required"` // Universal Resource Notation

	CreatedBy uid.ID

	ExpiresAt          *time.Time
	LastUsedAt         *time.Time
	ExpiresAfterUnused time.Duration
}

func (r *Grant) ToAPI() api.Grant {
	result := api.Grant{
		ID:        r.ID,
		Created:   r.CreatedAt.Unix(),
		Updated:   r.UpdatedAt.Unix(),
		CreatedBy: r.CreatedBy,

		Identity:  r.Identity,
		Privilege: r.Privilege,
		Resource:  r.Resource,
	}

	if r.ExpiresAt != nil {
		u := r.ExpiresAt.Unix()
		result.ExpiresAt = &u
	}

	return result
}

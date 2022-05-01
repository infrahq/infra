package models

import (
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

const (
	InfraAdminRole     = "admin"
	InfraViewRole      = "view"
	InfraConnectorRole = "connector"
)

// BasePermissionConnect is the first-principle permission that all other permissions are defined from.
// This permission gives you permission to authenticate with a destination
const BasePermissionConnect = "connect"

// Grant is a lean tuple of subject(identity) <-> privilege <-> resource (URN) relationships.
// field bloat should be avoided here since this model is going to be used heavily.
//
// Subject
// 		Subject is mostly an Identity, which is a string specifying a user, group, the name of a role, or another grant
// 			- an identity:  	i:E97WmsYfvo   		 - a user reference
// 			- a group: 			g:CCoJ1ornpf   		 - a group reference
// 			- a role:  			r:role-name   		 - a role definition
// 			- a permission: p:permissionn-name - a permission definition
// Privilege
// 		Privilege is a predicate that describes what sort of access the identity has to the resource
// URN
// 		URN is Universal Resource Notation.
// Expiry
//    time you want the grant to expire at
//
type Grant struct {
	Model

	Subject   uid.PolymorphicID `validate:"required"` // usually an identity, but could be a role definition
	Privilege string            `validate:"required"` // role or permission
	Resource  string            `validate:"required"` // Universal Resource Notation
}

func (r *Grant) ToAPI() *api.Grant {
	grant := &api.Grant{
		ID:        r.ID,
		Created:   api.Time(r.CreatedAt),
		Updated:   api.Time(r.UpdatedAt),
		Privilege: r.Privilege,
		Resource:  r.Resource,
	}

	switch {
	case r.Subject.IsIdentity():
		identity, err := r.Subject.ID()
		if err != nil {
			return nil
		}

		grant.Identity = identity
	case r.Subject.IsGroup():
		group, err := r.Subject.ID()
		if err != nil {
			return nil
		}

		grant.Group = group
	}

	return grant
}

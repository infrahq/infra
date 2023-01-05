package models

import (
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

const (
	InfraSupportAdminRole = "support-admin"
	InfraAdminRole        = "admin"
	InfraViewRole         = "view"
	InfraConnectorRole    = "connector"
)

// BasePermissionConnect is the first-principle permission that all other permissions are defined from.
// This permission gives you permission to authenticate with a destination
const BasePermissionConnect = "connect"

// Grant is an access grant.
type Grant struct {
	Model
	OrganizationMember

	// Subject is the ID of the user or group that is granted access to a resource.
	Subject uid.PolymorphicID
	// Privilege is the role or permission being granted.
	Privilege string
	// Resource identifies the resource the privilege applies to.
	Resource    string
	CreatedBy   uid.ID
	UpdateIndex int64 `db:"-"`
}

func (r *Grant) ToAPI() *api.Grant {
	grant := &api.Grant{
		ID:        r.ID,
		Created:   api.Time(r.CreatedAt),
		Updated:   api.Time(r.UpdatedAt),
		CreatedBy: r.CreatedBy,
		Privilege: r.Privilege,
		Resource:  r.Resource,
	}

	switch {
	case r.Subject.IsIdentity():
		identity, err := r.Subject.ID()
		if err != nil {
			return nil
		}

		grant.User = identity
	case r.Subject.IsGroup():
		group, err := r.Subject.ID()
		if err != nil {
			return nil
		}

		grant.Group = group
	}

	return grant
}

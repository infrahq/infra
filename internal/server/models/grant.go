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

type Subject struct {
	ID   uid.ID
	Kind SubjectKind
}

// TODO: rename to IsUser
func (s Subject) IsIdentity() bool {
	return s.Kind == SubjectUser
}

func (s Subject) IsGroup() bool {
	return s.Kind == SubjectGroup
}

type SubjectKind int

func (k SubjectKind) String() string {
	switch k {
	case SubjectUser:
		return "user"
	case SubjectGroup:
		return "group"
	default:
		return ""
	}
}

const (
	SubjectUser  SubjectKind = 1
	SubjectGroup SubjectKind = 2
)

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

func NewSubjectForUser(id uid.ID) uid.PolymorphicID {
	return uid.NewPolymorphicID("i", id)
}

func NewSubjectForGroup(id uid.ID) uid.PolymorphicID {
	return uid.NewPolymorphicID("g", id)
}

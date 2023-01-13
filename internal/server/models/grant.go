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
	Subject Subject
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

type SubjectKind int

func (k SubjectKind) String() string {
	switch k {
	case SubjectKindUser:
		return "user"
	case SubjectKindGroup:
		return "group"
	default:
		return ""
	}
}

const (
	SubjectKindUser  SubjectKind = 1
	SubjectKindGroup SubjectKind = 2
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

	switch r.Subject.Kind {
	case SubjectKindUser:
		grant.User = r.Subject.ID
	case SubjectKindGroup:
		grant.Group = r.Subject.ID
	}
	return grant
}

func NewSubjectForUser(id uid.ID) Subject {
	return Subject{ID: id, Kind: SubjectKindUser}
}

func NewSubjectForGroup(id uid.ID) Subject {
	return Subject{ID: id, Kind: SubjectKindGroup}
}

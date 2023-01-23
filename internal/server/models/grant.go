package models

import (
	"fmt"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

const (
	InfraSupportAdminRole = "support-admin"
	InfraAdminRole        = "admin"
	InfraViewRole         = "view"
	InfraConnectorRole    = "connector"

	GrantDestinationInfra = "infra"
)

// Grant is an access grant.
type Grant struct {
	Model
	OrganizationMember

	CreatedBy   uid.ID
	UpdateIndex int64 `db:"-"`

	// Subject is the ID of the user or group that is granted access to a resource.
	Subject Subject
	// Privilege is the role or permission being granted.
	Privilege string
	// DestinationName identifies the destination the grant applies to
	DestinationName string
	// DestinationResource identifies the destination-specific resource the grant applies to
	DestinationResource string
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
	}

	switch {
	case r.DestinationResource != "":
		grant.Resource = fmt.Sprintf("%s.%s", r.DestinationName, r.DestinationResource)
	default:
		grant.Resource = r.DestinationName
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

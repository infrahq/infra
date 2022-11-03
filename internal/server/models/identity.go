package models

import (
	"time"

	"github.com/ssoroka/slice"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

const (
	InternalInfraConnectorIdentityName = "connector"
)

type Identity struct {
	Model
	OrganizationMember

	Name              string
	LastSeenAt        time.Time // updated on when an identity uses a session token
	CreatedBy         uid.ID
	Verified          bool
	VerificationToken string

	// Groups may be populated by some queries to contain the list of groups
	// the user is a member of.  Some test helpers may also use this to add
	// users to groups, but data.CreateUser does not read this field.
	Groups []Group `db:"-"`
	// Providers may be populated by some queries to contain the list of
	// providers that provide this user.
	Providers []Provider `db:"-"`
}

func (i *Identity) ToAPI() *api.User {
	return &api.User{
		ID:         i.ID,
		Created:    api.Time(i.CreatedAt),
		Updated:    api.Time(i.UpdatedAt),
		LastSeenAt: api.Time(i.LastSeenAt),
		Name:       i.Name,
		ProviderNames: slice.Map[Provider, string](i.Providers, func(p Provider) string {
			return p.Name
		}),
	}
}

// PolyID is a polymorphic name that points to both a model type and an ID
func (i *Identity) PolyID() uid.PolymorphicID {
	return uid.NewIdentityPolymorphicID(i.ID)
}

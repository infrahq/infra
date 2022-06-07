package models

import (
	"time"

	"github.com/ssoroka/slice"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

const (
	InternalInfraAdminIdentityName     = "admin"
	InternalInfraConnectorIdentityName = "connector"
)

type Identity struct {
	Model

	Name       string    `gorm:"uniqueIndex:idx_identities_name,where:deleted_at is NULL"`
	LastSeenAt time.Time // updated on when an identity uses a session token
	CreatedBy  uid.ID

	// for eager loading, don't use these for saving.
	Groups    []Group    `gorm:"many2many:identities_groups"`
	Providers []Provider `gorm:"many2many:provider_users;"`
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

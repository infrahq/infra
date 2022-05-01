package models

import (
	"time"

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

	Groups []Group `gorm:"many2many:identities_groups"`
}

func (i *Identity) ToAPI() *api.Identity {
	return &api.Identity{
		ID:         i.ID,
		Created:    api.Time(i.CreatedAt),
		Updated:    api.Time(i.UpdatedAt),
		LastSeenAt: api.Time(i.LastSeenAt),
		Name:       i.Name,
	}
}

// PolyID is a polymorphic name that points to both a model type and an ID
func (i *Identity) PolyID() uid.PolymorphicID {
	return uid.NewIdentityPolymorphicID(i.ID)
}

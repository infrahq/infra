package models

import (
	"time"

	"github.com/ssoroka/slice"
	"gorm.io/gorm"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/uid"
)

const (
	InternalInfraConnectorIdentityName = "connector"
)

type Identity struct {
	Model
	OrganizationMember

	Name              string    `gorm:"uniqueIndex:idx_identities_name,where:deleted_at is NULL"`
	LastSeenAt        time.Time // updated on when an identity uses a session token
	CreatedBy         uid.ID
	Verified          bool
	VerificationToken string

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

func (i *Identity) BeforeSave(_ *gorm.DB) error {
	if len(i.VerificationToken) == 0 {
		i.VerificationToken = generate.MathRandom(10, generate.CharsetAlphaNumeric)
	}

	return nil
}

package models

import (
	"time"

	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/uid"
)

type User struct {
	Model

	Email      string    `gorm:"uniqueIndex:idx_users_email_provider_id,where:deleted_at is NULL"`
	LastSeenAt time.Time // updated on when user uses a session token

	ProviderID uid.ID `gorm:"uniqueIndex:idx_users_email_provider_id,where:deleted_at is NULL"`

	Groups []Group `gorm:"many2many:users_groups"`
}

func (u *User) ToAPI() *api.User {
	result := &api.User{
		ID:         u.ID,
		Created:    u.CreatedAt.Unix(),
		Updated:    u.UpdatedAt.Unix(),
		Email:      u.Email,
		ProviderID: u.ProviderID,
		LastSeenAt: u.LastSeenAt.Unix(),
	}

	return result
}

func (u *User) PolymorphicIdentifier() uid.PolymorphicID {
	return uid.NewUserPolymorphicID(u.ID)
}

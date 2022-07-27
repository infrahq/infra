package models

import (
	"time"

	"github.com/infrahq/infra/uid"
)

// ProviderUser is a cache of the provider's user and their groups, plus any authentication-specific information for that provider.
type ProviderUser struct {
	ProviderID uid.ID `validate:"required" gorm:"primaryKey"`
	IdentityID uid.ID `validate:"required" gorm:"primaryKey"`

	Email      string `validate:"required"`
	Groups     CommaSeparatedStrings
	LastUpdate time.Time `validate:"required"`

	RedirectURL string // needs to match the redirect URL specified when the token was issued for refreshing

	AccessToken  EncryptedAtRest
	RefreshToken EncryptedAtRest
	ExpiresAt    time.Time
}

func (ProviderUser) IsAModel() {}

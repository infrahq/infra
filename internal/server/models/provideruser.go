package models

import (
	"time"

	"github.com/infrahq/infra/uid"
)

// ProviderUser is a cache of the provider's user and their groups, plus any authentication-specific information for that provider.
type ProviderUser struct {
	IdentityID uid.ID `gorm:"primaryKey"`
	ProviderID uid.ID `gorm:"primaryKey"`

	Email      string
	Groups     CommaSeparatedStrings
	LastUpdate time.Time

	RedirectURL string // needs to match the redirect URL specified when the token was issued for refreshing

	AccessToken  EncryptedAtRest
	RefreshToken EncryptedAtRest
	ExpiresAt    time.Time
}

func (ProviderUser) IsAModel() {}

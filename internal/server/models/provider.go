package models

import (
	"time"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

const InternalInfraProviderName = "infra"

type Provider struct {
	Model

	Name         string `gorm:"uniqueIndex:,where:deleted_at is NULL" validate:"required"`
	URL          string
	ClientID     string
	ClientSecret EncryptedAtRest
	CreatedBy    uid.ID

	Users []Identity
}

func (p *Provider) ToAPI() *api.Provider {
	return &api.Provider{
		Name:    p.Name,
		ID:      p.ID,
		Created: api.Time(p.CreatedAt),
		Updated: api.Time(p.UpdatedAt),

		URL:      p.URL,
		ClientID: p.ClientID,
	}
}

// ProviderToken tracks the access and refresh tokens from an identity provider associated with a user
type ProviderToken struct {
	Model

	UserID      uid.ID
	ProviderID  uid.ID
	RedirectURL string `validate:"required"` // needs to match the redirect URL specified when the token was issued for refreshing

	AccessToken  EncryptedAtRest
	RefreshToken EncryptedAtRest
	ExpiresAt    time.Time
}

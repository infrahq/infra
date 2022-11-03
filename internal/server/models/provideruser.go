package models

import (
	"time"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

// ProviderUser is a cache of the provider's user and their groups, plus any authentication-specific information for that provider.
type ProviderUser struct {
	IdentityID uid.ID
	ProviderID uid.ID

	Email      string
	GivenName  string
	FamilyName string
	Groups     CommaSeparatedStrings
	LastUpdate time.Time

	RedirectURL string // needs to match the redirect URL specified when the token was issued for refreshing

	AccessToken  EncryptedAtRest
	RefreshToken EncryptedAtRest
	ExpiresAt    time.Time

	Active bool
}

func (pu *ProviderUser) ToAPI() *api.SCIMUser {
	return &api.SCIMUser{
		Schemas:  []string{api.UserSchema},
		ID:       pu.IdentityID.String(),
		UserName: pu.Email,
		Name: api.SCIMUserName{
			GivenName:  pu.GivenName,
			FamilyName: pu.FamilyName,
		},
		Emails: []api.SCIMUserEmail{
			{
				Primary: true,
				Value:   pu.Email,
			},
		},
		Active: pu.Active,
		Meta: api.SCIMMetadata{
			ResourceType: "User",
		},
	}
}

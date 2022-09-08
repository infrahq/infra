package models

import (
	"time"

	"github.com/infrahq/infra/uid"
)

// ProviderGroup is a local copy of the of a group from an identity provider
// See this diagram for more details about how this model relates to a group
// https://github.com/infrahq/infra/blob/main/docs/dev/identity-provider-tracking.md
type ProviderGroup struct {
	OrganizationMember
	CreatedAt time.Time
	UpdatedAt time.Time

	ProviderID uid.ID
	Name       string

	// for loading, not for saving
	Members []ProviderUser
}

package models

import (
	"github.com/infrahq/infra/uid"
)

// Credential stores information required for local users to login to Infra
type Credential struct {
	Model
	OrganizationMember

	IdentityID      uid.ID `gorm:"<-;uniqueIndex:idx_credentials_identity_id,where:deleted_at is NULL"`
	PasswordHash    []byte
	OneTimePassword bool
}

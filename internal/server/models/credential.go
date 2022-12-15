package models

import (
	"github.com/infrahq/infra/uid"
)

// Credential stores information required for local users to login to Infra
type Credential struct {
	Model
	OrganizationMember

	IdentityID      uid.ID
	PasswordHash    []byte
	OneTimePassword bool
}

package models

import "github.com/infrahq/infra/uid"

type Credential struct {
	Model
	OrganizationMember

	IdentityID      uid.ID
	PasswordHash    []byte
	OneTimePassword bool
}

package models

import "github.com/infrahq/infra/uid"

type Credential struct {
	Model

	IdentityID      uid.ID `gorm:"<-"`
	PasswordHash    []byte
	OneTimePassword bool
}

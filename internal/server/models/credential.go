package models

import "github.com/infrahq/infra/uid"

type Credential struct {
	Model

	IdentityID          uid.ID `gorm:"<-;uniqueIndex:,where:deleted_at is NULL"`
	PasswordHash        []byte `validate:"required"`
	OneTimePassword     bool
	OneTimePasswordUsed bool
}

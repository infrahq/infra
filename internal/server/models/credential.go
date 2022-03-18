package models

import "github.com/infrahq/infra/uid"

type Credential struct {
	Model

	Identity            uid.PolymorphicID `gorm:"<-;uniqueIndex:,where:deleted_at is NULL"`
	PasswordHash        []byte            `validate:"required"`
	OneTimePassword     bool
	OneTimePasswordUsed bool
}

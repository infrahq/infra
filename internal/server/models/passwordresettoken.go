package models

import (
	"time"

	"github.com/infrahq/infra/uid"
)

type PasswordResetToken struct {
	ID uid.ID
	OrganizationMember

	Token      string    `validate:"required" gorm:"uniqueIndex"`
	IdentityID uid.ID    `validate:"required"`
	ExpiresAt  time.Time `validate:"required"`
}

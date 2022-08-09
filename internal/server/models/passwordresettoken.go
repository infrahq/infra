package models

import (
	"time"

	"github.com/infrahq/infra/uid"
)

type PasswordResetToken struct {
	ID             uid.ID
	OrganizationID uid.ID

	Token      string    `validate:"required" gorm:"uniqueIndex"`
	IdentityID uid.ID    `validate:"required"`
	ExpiresAt  time.Time `validate:"required"`
}

func (PasswordResetToken) IsAModel() {}

package models

import (
	"time"

	"github.com/infrahq/infra/uid"
)

var (
	AccessKeyKeyLength    = 10
	AccessKeySecretLength = 24
)

// AccessKey is a session token presented to the Infra server as proof of authentication
type AccessKey struct {
	Model
	Name              string    `gorm:"uniqueIndex:,where:deleted_at is NULL" validate:"excludes= "`
	IssuedFor         uid.ID    `validate:"required"`
	IssuedForIdentity *Identity `gorm:"foreignKey:IssuedFor"`
	ProviderID        uid.ID    `validate:"required"`

	ExpiresAt         time.Time     `validate:"required"`
	Extension         time.Duration // how long to increase the lifetime extension deadline by
	ExtensionDeadline time.Time

	KeyID          string `gorm:"<-;uniqueIndex:,where:deleted_at is NULL"`
	Secret         string `gorm:"-"`
	SecretChecksum []byte
}

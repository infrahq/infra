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
	Name      string            `gorm:"uniqueIndex:,where:deleted_at is NULL"`
	IssuedFor uid.PolymorphicID `validate:"required"`

	ExpiresAt         time.Time     `validate:"required"`
	Extension         time.Duration // how long to increase the lifetime extension deadline by
	ExtensionDeadline time.Time

	Key            string `gorm:"<-;uniqueIndex:,where:deleted_at is NULL"`
	Secret         string `gorm:"-"`
	SecretChecksum []byte
}

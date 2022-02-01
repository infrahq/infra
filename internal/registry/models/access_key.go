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
	Name      string // optional name
	UserID    uid.ID
	ExpiresAt time.Time

	// TODO: remove me with machine identities
	Permissions string

	Key            string
	Secret         string `gorm:"-"`
	SecretChecksum []byte
}

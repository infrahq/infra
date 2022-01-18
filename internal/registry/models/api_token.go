package models

import (
	"time"

	"github.com/infrahq/infra/uid"
)

var APITokenSecretLength = 24

type APIToken struct {
	Model
	Name      string // optional name
	UserID    uid.ID
	ExpiresAt time.Time

	// TODO: remove me with machine identities
	Permissions string

	Secret         string `gorm:"-"`
	SecretChecksum []byte
}

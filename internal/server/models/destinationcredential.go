package models

import (
	"time"

	"github.com/infrahq/infra/uid"
)

// DestinationCredential represents a request from a user to get login credentials to a destination.
type DestinationCredential struct {
	ID uid.ID
	OrganizationMember

	RequestExpiresAt time.Time

	UpdateIndex int64
	Answered    bool

	////////////////////
	// request fields //
	////////////////////
	UserID        uid.ID
	DestinationID uid.ID

	/////////////////////
	// response fields //
	/////////////////////

	CredentialExpiresAt *time.Time
	BearerToken         EncryptedAtRest
}

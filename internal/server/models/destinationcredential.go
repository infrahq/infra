package models

import (
	"database/sql"
	"time"

	"github.com/infrahq/infra/uid"
)

// DestinationCredential represents a request from a user to get login credentials to a destination.
type DestinationCredential struct {
	ID uid.ID
	OrganizationMember

	ExpiresAt time.Time

	UpdateIndex int64
	Answered    bool

	////////////////////
	// request fields //
	////////////////////
	UserID        uid.ID
	DestinationID uid.ID

	// // certificate
	// PublicCertificate []byte // supplied if the user is planning to connect via client-generated certificate pair

	// // ssh
	// PublicKey []byte // supplied if the user is planning to connect via client-generated key pair

	/////////////////////
	// response fields //
	/////////////////////

	// // username & pw
	// Username string
	// Password string

	// // API key
	BearerToken sql.NullString

	// // Certificate

	// // JWT or generic headers
	// HeaderName string
	// Token      string
}

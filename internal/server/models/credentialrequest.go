package models

import (
	"time"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

// CredentialRequest represents a request from a user to get login credentials to a destination.
type CredentialRequest struct {
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
	BearerToken string

	// // Certificate

	// // JWT or generic headers
	// HeaderName string
	// Token      string
}

func (c CredentialRequest) ToAPI() api.CredentialRequest {
	return api.CredentialRequest{
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		ExpiresAt:      api.Time(c.ExpiresAt),
		UserID:         c.UserID,
		BearerToken:    c.BearerToken,
	}
}

func (c *CredentialRequest) FromUpdateAPI(r *api.UpdateCredentialRequest) {
	c.BearerToken = r.BearerToken
}

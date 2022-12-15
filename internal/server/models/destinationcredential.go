package models

import (
	"time"

	"github.com/infrahq/infra/api"
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

func (c DestinationCredential) ToAPI() api.DestinationCredential {
	dc := api.DestinationCredential{
		ID:               c.ID,
		OrganizationID:   c.OrganizationID,
		RequestExpiresAt: api.Time(c.RequestExpiresAt),
		UserID:           c.UserID,
		BearerToken:      string(c.BearerToken),
		UpdateIndex:      c.UpdateIndex,
	}
	if c.CredentialExpiresAt != nil {
		dc.CredentialExpiresAt = api.Time(*c.CredentialExpiresAt)
	}
	return dc
}

func (c *DestinationCredential) FromUpdateAPI(r *api.AnswerDestinationCredential) {
	c.BearerToken = EncryptedAtRest(r.BearerToken)
	c.CredentialExpiresAt = (*time.Time)(&r.CredentialExpiresAt)
}

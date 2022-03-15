package api

import "github.com/infrahq/infra/uid"

// Introspect returns information about the party that the calling token was issued for
type Introspect struct {
	ID           uid.ID `json:"id"`
	Name         string `json:"name"`         // the machine name or the user email
	IdentityType string `json:"identityType"` // user or machine
}

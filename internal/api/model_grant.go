package api

import (
	"github.com/infrahq/infra/uid"
)

type Grant struct {
	ID uid.ID `json:"id"`

	Created   int64  `json:"created"`    // created time in seconds since 1970-01-01 00:00:00 UTC
	CreatedBy uid.ID `json:"created_by"` // id of user who created the grant
	Updated   int64  `json:"updated"`    // updated time in seconds since 1970-01-01 00:00:00 UTC

	Identity  string `json:"identity"`  // format is "u:<idstr>" for users, "g:<idstr>" for groups
	Privilege string `json:"privilege"` // role or permission
	Resource  string `json:"resource"`  // Universal Resource Notation

	ExpiresAt *int64 `json:"expires_at"` // time this grant expires at in seconds since 1970-01-01 00:00:00 UTC
}

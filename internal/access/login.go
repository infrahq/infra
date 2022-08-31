package access

import (
	"time"

	"github.com/infrahq/infra/internal/server/authn"
)

// Login uses a login method to authenticate a user
func Login(c RequestContext, loginMethod authn.LoginMethod, keyExpiresAt time.Time, keyExtension time.Duration) (authn.LoginResult, error) {
	return authn.Login(c.Request.Context(), c.DBTxn, loginMethod, keyExpiresAt, keyExtension)
}

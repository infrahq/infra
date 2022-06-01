package access

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/models"
)

// Login uses a login method to authenticate a user
func Login(c *gin.Context, loginMethod authn.LoginMethod, keyExpiresAt time.Time, keyExtension time.Duration) (*models.AccessKey, string, bool, error) {
	db := getDB(c)
	key, bearer, err := authn.Login(db, loginMethod, keyExpiresAt, keyExtension)
	if err != nil {
		return nil, "", false, err
	}

	// In the case of username/password credentials,
	// the login may fail if the password presented was a one-time password that has been used.
	// This can be removed when #1441 is resolved
	requiresUpdate, err := loginMethod.RequiresUpdate(db)
	if err != nil {
		return nil, bearer, false, err
	}

	return key, bearer, requiresUpdate, nil
}

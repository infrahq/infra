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
	result, err := authn.Login(c.Request.Context(), db, loginMethod, keyExpiresAt, keyExtension)
	if err != nil {
		return nil, "", false, err
	}

	return result.AccessKey, result.Bearer, result.CredentialUpdateRequired, nil
}

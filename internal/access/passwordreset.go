package access

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func VerifiedPasswordReset(c *gin.Context, token, newPassword string) (*models.Identity, error) {
	// no auth required
	rCtx := GetRequestContext(c)
	tx := rCtx.DBTxn

	userID, err := data.ClaimPasswordResetToken(tx, token)
	if err != nil {
		return nil, err
	}

	user, err := data.GetIdentity(tx, data.GetIdentityOptions{ByID: userID})
	if err != nil {
		return nil, err
	}

	if !user.Verified {
		user.Verified = true
		if err = data.UpdateIdentity(tx, user); err != nil {
			return nil, err
		}
	}

	if err := updateCredential(c, user, newPassword, true); err != nil {
		return nil, err
	}
	return user, nil
}

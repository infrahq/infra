package access

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func VerifiedPasswordReset(c *gin.Context, token, password string) (*models.Identity, error) {
	// no auth required
	tx := GetRequestContext(c).DBTxn

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

	credential, err := data.GetCredentialByUserID(tx, user.ID)
	if err != nil {
		return nil, err
	}

	if err := checkPasswordRequirements(tx, password); err != nil {
		return nil, err
	}

	credential.OneTimePassword = false

	if err := updateCredential(tx, credential, password); err != nil {
		return nil, err
	}

	return user, nil
}

package access

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func PasswordResetRequest(c *gin.Context, email string, ttl time.Duration) (token string, user *models.Identity, err error) {
	// no auth required
	rCtx := GetRequestContext(c)
	db := rCtx.DBTxn

	opts := data.GetIdentityOptions{
		ByName: email,
	}
	user, err = data.GetIdentity(db, opts)
	if err != nil {
		return "", nil, err
	}

	_, err = data.GetCredentialByUserID(db, user.ID)
	if err != nil {
		// if credential is not found, the user cannot reset their password.
		return "", nil, err
	}

	token, err = data.CreatePasswordResetToken(db, user.ID, ttl)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

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

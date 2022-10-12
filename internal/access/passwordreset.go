package access

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func PasswordResetRequest(c *gin.Context, email string, ttl time.Duration) (token string, user *models.Identity, err error) {
	// no auth required
	rCtx := GetRequestContext(c)
	db := rCtx.DBTxn

	users, err := data.ListIdentities(db, &data.Pagination{Limit: 1}, data.ByName(email))
	if err != nil {
		return "", nil, err
	}

	if len(users) != 1 {
		return "", nil, internal.ErrNotFound
	}

	_, err = data.GetCredential(db, data.ByIdentityID(users[0].ID))
	if err != nil {
		// if credential is not found, the user cannot reset their password.
		return "", nil, err
	}

	prt, err := data.CreatePasswordResetToken(db, &users[0], ttl)
	if err != nil {
		return "", nil, err
	}

	return prt.Token, &users[0], nil
}

func VerifiedPasswordReset(c *gin.Context, token, newPassword string) (*models.Identity, error) {
	// no auth required
	rCtx := GetRequestContext(c)
	db := rCtx.DBTxn

	prt, err := data.GetPasswordResetTokenByToken(db, token)
	if err != nil {
		return nil, err
	}

	user, err := data.GetIdentity(db, data.ByID(prt.IdentityID))
	if err != nil {
		return nil, err
	}

	if !user.Verified {
		user.Verified = true
		if err = data.SaveIdentity(db, user); err != nil {
			return nil, err
		}
	}

	if err := updateCredential(c, user, newPassword, true); err != nil {
		return nil, err
	}

	if err := data.DeletePasswordResetToken(db, prt); err != nil {
		logging.Errorf("deleting password reset token: %s", err)
	}

	return user, nil
}

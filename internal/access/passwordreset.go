package access

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func PasswordResetRequest(c *gin.Context, email string) (token string, err error) {
	// no auth required
	db := getDB(c)

	users, err := data.ListIdentities(db, &models.Pagination{Limit: 1}, data.ByName(email))
	if err != nil {
		return "", err
	}

	if len(users) != 1 {
		return "", internal.ErrNotFound
	}

	prt, err := data.CreatePasswordResetToken(db, &users[0])
	if err != nil {
		return "", err
	}

	return prt.Token, nil
}

func VerifiedPasswordReset(c *gin.Context, token, newPassword string) (*models.Identity, error) {
	// no auth required
	db := getDB(c)

	prt, err := data.GetPasswordResetTokenByToken(db, token)
	if err != nil {
		return nil, err
	}

	user, err := data.GetIdentity(db, data.ByID(prt.IdentityID))
	if err != nil {
		return nil, err
	}

	if err := updateCredential(c, user, newPassword, true); err != nil {
		return nil, err
	}

	return user, nil
}

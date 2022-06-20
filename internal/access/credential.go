package access

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func CreateCredential(c *gin.Context, user models.Identity) (string, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return "", err
	}

	tmpPassword, err := generate.CryptoRandom(12, generate.CharsetPassword)
	if err != nil {
		return "", fmt.Errorf("generate: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(tmpPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash: %w", err)
	}

	userCredential := &models.Credential{
		IdentityID:          user.ID,
		PasswordHash:        hash,
		OneTimePassword:     true,
		OneTimePasswordUsed: false,
	}

	if err := data.CreateCredential(db, userCredential); err != nil {
		return "", err
	}

	return tmpPassword, nil
}

func UpdateCredential(c *gin.Context, user *models.Identity, newPassword string) error {
	db, err := hasAuthorization(c, user.ID, isIdentitySelf, models.InfraAdminRole)
	if err != nil {
		return err
	}

	isSelf, err := isIdentitySelf(c, user.ID)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash: %w", err)
	}

	userCredential, err := data.GetCredential(db, data.ByIdentityID(user.ID))
	if err != nil {
		if errors.Is(err, internal.ErrNotFound) && !isSelf {
			if err := data.CreateCredential(db, &models.Credential{
				IdentityID:          user.ID,
				PasswordHash:        hash,
				OneTimePassword:     true,
				OneTimePasswordUsed: false,
			}); err != nil {
				return fmt.Errorf("creating credentials: %w", err)
			}
			return nil
		}
		return fmt.Errorf("existing credential: %w", err)
	}

	userCredential.PasswordHash = hash
	userCredential.OneTimePassword = !isSelf
	userCredential.OneTimePasswordUsed = false

	if err := data.SaveCredential(db, userCredential); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	return nil
}

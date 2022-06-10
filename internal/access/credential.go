package access

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func CreateCredential(c *gin.Context, user models.Identity) (string, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return "", err
	}

	oneTimePassword, err := generate.CryptoRandom(10)
	if err != nil {
		return "", fmt.Errorf("generate: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(oneTimePassword), bcrypt.DefaultCost)
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

	return oneTimePassword, nil
}

func UpdateCredential(c *gin.Context, user *models.Identity, newPassword string) error {
	db, err := hasAuthorization(c, user.ID, isIdentitySelf, models.InfraAdminRole)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash: %w", err)
	}

	userCredential, err := data.GetCredential(db, data.ByIdentityID(user.ID))
	if err != nil {
		return fmt.Errorf("existing credential: %w", err)
	}

	userCredential.PasswordHash = hash
	userCredential.OneTimePassword = false
	userCredential.OneTimePasswordUsed = false

	if user.ID != AuthenticatedIdentity(c).ID {
		// an admin can only set a one time password for a user
		userCredential.OneTimePassword = true
		userCredential.OneTimePasswordUsed = false
	}

	if err := data.SaveCredential(db, userCredential); err != nil {
		return err
	}

	return nil
}

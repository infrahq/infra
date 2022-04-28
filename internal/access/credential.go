package access

import (
	"fmt"
	"time"

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

func LoginWithPasswordCredential(c *gin.Context, name, password string, expiry time.Time) (string, *models.Identity, bool, error) {
	db := getDB(c)

	identity, err := data.GetIdentity(db, data.ByName(name))
	if err != nil {
		return "", nil, false, fmt.Errorf("%w: credentials name: %v", internal.ErrUnauthorized, err)
	}

	requiresUpdate, err := data.ValidateCredential(db, identity, password)
	if err != nil {
		return "", nil, false, fmt.Errorf("%w: validate password: %v", internal.ErrUnauthorized, err)
	}

	// the password is valid
	issuedAccessKey := &models.AccessKey{
		IssuedFor:  identity.ID,
		ProviderID: InfraProvider(c).ID,
		ExpiresAt:  expiry,
	}

	secret, err := data.CreateAccessKey(db, issuedAccessKey)
	if err != nil {
		return "", nil, false, fmt.Errorf("create token for creds: %w", err)
	}

	return secret, identity, requiresUpdate, nil
}

package access

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"golang.org/x/crypto/bcrypt"
)

func CreateCredential(c *gin.Context, user models.User) (string, error) {
	db, err := requireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return "", err
	}

	oneTimePassword, err := generate.CryptoRandom(10)
	if err != nil {
		return "", fmt.Errorf("generate: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(oneTimePassword), bcrypt.MinCost)
	if err != nil {
		return "", fmt.Errorf("hash: %w", err)
	}

	userCredential := &models.Credential{
		Identity:            user.PolymorphicIdentifier(),
		PasswordHash:        hash,
		OneTimePassword:     true,
		OneTimePasswordUsed: false,
	}

	if err := data.CreateCredential(db, userCredential); err != nil {
		return "", err
	}

	return oneTimePassword, nil
}

func UpdateCredential(c *gin.Context, user *models.User, newPassword string) error {
	db, err := hasAuthorization(c, user.ID, isUserSelf, models.InfraAdminRole)
	if err != nil {
		return err
	}

	provider, err := data.GetProvider(db, data.ByID(user.ProviderID))
	if err != nil {
		return fmt.Errorf("creds user provider: %w", err)
	}

	// only internal Infra users can have username/password authentication
	if provider.Name != models.InternalInfraProviderName {
		return fmt.Errorf("%w: cannot set user passwords in this provider", internal.ErrBadRequest)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash: %w", err)
	}

	userCredential, err := data.GetCredential(db, data.ByIdentity(user.PolymorphicIdentifier()))
	if err != nil {
		return fmt.Errorf("existing credential: %w", err)
	}

	userCredential.PasswordHash = hash
	userCredential.OneTimePassword = false
	userCredential.OneTimePasswordUsed = false

	if user.PolymorphicIdentifier() != getCurrentIdentity(c) {
		// an admin can only set a one time password for a user
		userCredential.OneTimePassword = true
		userCredential.OneTimePasswordUsed = false
	}

	if err := data.SaveCredential(db, userCredential); err != nil {
		return err
	}

	return nil
}

func LoginWithUserCredential(c *gin.Context, email, password string, expiry time.Time) (string, *models.User, bool, error) {
	db := getDB(c)

	infraProvider, err := data.GetProvider(db, data.ByName(models.InternalInfraProviderName))
	if err != nil {
		return "", nil, false, fmt.Errorf("%w: internal provider: %v", internal.ErrUnauthorized, err)
	}

	user, err := data.GetUser(db, data.ByEmail(email), data.ByProviderID(infraProvider.ID))
	if err != nil {
		return "", nil, false, fmt.Errorf("%w: credentials email: %v", internal.ErrUnauthorized, err)
	}

	requiresUpdate, err := data.ValidateCredential(db, user, password)
	if err != nil {
		return "", nil, false, fmt.Errorf("%w: validate password: %v", internal.ErrUnauthorized, err)
	}

	// the password is valid
	issuedAccessKey := &models.AccessKey{
		IssuedFor: user.PolymorphicIdentifier(),
		ExpiresAt: expiry,
	}

	secret, err := data.CreateAccessKey(db, issuedAccessKey)
	if err != nil {
		return "", nil, false, fmt.Errorf("create token for creds: %w", err)
	}

	return secret, user, requiresUpdate, nil
}

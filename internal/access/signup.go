package access

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type SignupDetails struct {
	Name      string
	Password  string
	Org       *models.Organization
	SubDomain string
}

// Signup creates a user identity using the supplied name and password and
// grants the identity "admin" access to Infra.
func Signup(c *gin.Context, keyExpiresAt time.Time, baseDomain string, details SignupDetails) (*models.Identity, string, error) {
	// no authorization is setup yet
	db := getDB(c)

	details.Org.Domain = SanitizedDomain(details.SubDomain, baseDomain)

	if err := data.CreateOrganizationAndSetContext(db, details.Org); err != nil {
		return nil, "", fmt.Errorf("create org on sign-up: %w", err)
	}

	db = data.NewTransaction(db.GormDB(), details.Org.ID)

	// check the admin user's password requirements against our basic password requirements
	err := checkPasswordRequirements(db, details.Password)
	if err != nil {
		return nil, "", err
	}

	identity := &models.Identity{
		Name: details.Name,
	}

	if err := data.CreateIdentity(db, identity); err != nil {
		return nil, "", fmt.Errorf("create identity on sign-up: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(details.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("hash password on sign-up: %w", err)
	}

	_, err = CreateProviderUser(c, InfraProvider(c), identity)
	if err != nil {
		return nil, "", fmt.Errorf("create provider user on sign-up: %w", err)
	}

	credential := &models.Credential{
		IdentityID:   identity.ID,
		PasswordHash: hash,
	}

	if err := data.CreateCredential(db, credential); err != nil {
		return nil, "", fmt.Errorf("create credential on sign-up: %w", err)
	}

	err = data.CreateGrant(db, &models.Grant{
		Subject:   uid.NewIdentityPolymorphicID(identity.ID),
		Privilege: models.InfraAdminRole,
		Resource:  ResourceInfraAPI,
		CreatedBy: identity.ID,
	})
	if err != nil {
		return nil, "", fmt.Errorf("create grant on sign-up: %w", err)
	}

	// grant the user a session on initial sign-up
	accessKey := &models.AccessKey{
		IssuedFor:         identity.ID,
		IssuedForIdentity: identity,
		ProviderID:        data.InfraProvider(db).ID,
		ExpiresAt:         keyExpiresAt,
	}

	bearer, err := data.CreateAccessKey(db, accessKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create access key after sign-up: %w", err)
	}

	return identity, bearer, nil
}

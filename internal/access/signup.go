package access

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// Signup creates a user identity using the supplied name and password and
// grants the identity "admin" access to Infra.
func Signup(c *gin.Context, keyExpiresAt time.Time, name, password string, org *models.Organization) (*models.Identity, string, error) {
	// no authorization is setup yet
	db := getDB(c)

	err := checkPasswordRequirements(db, password)
	if err != nil {
		return nil, "", err
	}

	org.SetDefaultDomain()
	// lower-case domains look better
	org.Domain = strings.ToLower(org.Domain)
	if err := data.CreateOrganization(db, org); err != nil {
		return nil, "", fmt.Errorf("create org on sign-up: %w", err)
	}
	db.Statement.Context = data.WithOrg(db.Statement.Context, org)

	identity := &models.Identity{
		Name: name,
	}

	if err := data.CreateIdentity(db, identity); err != nil {
		return nil, "", fmt.Errorf("create identity on sign-up: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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

	grants := []*models.Grant{
		{
			Subject:   uid.NewIdentityPolymorphicID(identity.ID),
			Privilege: models.InfraAdminRole,
			Resource:  ResourceInfraAPI,
			CreatedBy: identity.ID,
		},
		{
			Subject:   uid.NewIdentityPolymorphicID(identity.ID),
			Privilege: models.InfraSupportAdminRole,
			Resource:  ResourceInfraAPI,
			CreatedBy: identity.ID,
		},
	}

	for _, grant := range grants {
		if err := data.CreateGrant(db, grant); err != nil {
			return nil, "", fmt.Errorf("create grant on sign-up: %w", err)
		}
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

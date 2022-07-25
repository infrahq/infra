package access

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// SignupEnabled queries the current state of the service and returns whether
// or not to allow unauthenticated signup. Signup is enabled if and only if
// the configuration flag 'enableSignup' is set and no identities, providers,
// or grants have been configured, both currently and previously.
func SignupEnabled(c *gin.Context) (bool, error) {
	// no authorization is setup yet
	db := getDB(c)

	// use Unscoped because deleting identities, providers or grants should not re-enable signup
	identities, err := data.Count[models.Identity](db.Unscoped(), data.NotName(models.InternalInfraConnectorIdentityName))
	if err != nil {
		return false, err
	}

	providers, err := data.Count[models.Provider](db.Unscoped(), data.NotProviderKind(models.ProviderKindInfra))
	if err != nil {
		return false, err
	}

	grants, err := data.Count[models.Grant](db.Unscoped(), data.NotPrivilege(models.InfraConnectorRole))
	if err != nil {
		return false, err
	}

	accessKeys, err := data.Count[models.AccessKey](db.Unscoped())
	if err != nil {
		return false, err
	}

	return identities+providers+grants+accessKeys == 0, nil
}

// Signup creates a user identity using the supplied name and password and
// grants the identity "admin" access to Infra.
func Signup(c *gin.Context, orgName, name, password string) (*models.Identity, error) {
	// no authorization is setup yet
	db := getDB(c)

	organization := &models.Organization{Name: orgName}
	if err := data.CreateOrganization(db, organization); err != nil {
		return nil, err
	}
	c.Set("organization", organization)
	logging.Infof("Org ID -> %s", organization.ID)

	// TODO: add connector user and grant permission

	identity, err := createInitialUser(c, db, name, models.InfraAdminRole, organization.ID, 0)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	credential := &models.Credential{
		IdentityID:   identity.ID,
		PasswordHash: hash,
	}
	credential.OrganizationID = organization.ID

	if err = data.CreateCredential(db, credential); err != nil {
		return nil, err
	}

	_, err = createInitialUser(c, db, "connector", models.InfraConnectorRole, organization.ID, identity.ID)
	if err != nil {
		return nil, err
	}

	return identity, nil
}

func createInitialUser(c *gin.Context, db *gorm.DB, name, grantType string, orgID, createdBy uid.ID) (*models.Identity, error) {
	identity := &models.Identity{Name: name}
	identity.OrganizationID = orgID

	if err := data.CreateIdentity(db, identity); err != nil {
		return nil, err
	}

	_, err := CreateProviderUser(c, InfraProvider(c), identity)
	if err != nil {
		return nil, fmt.Errorf("couldn't create provider user")
	}

	grant := &models.Grant{
		Subject:   uid.NewIdentityPolymorphicID(identity.ID),
		Privilege: models.InfraAdminRole,
		Resource:  "infra",
		CreatedBy: identity.ID,
	}
	if createdBy > 0 {
		grant.CreatedBy = createdBy
	}
	grant.OrganizationID = orgID

	if err := data.CreateGrant(db, grant); err != nil {
		return nil, err
	}

	return identity, nil
}

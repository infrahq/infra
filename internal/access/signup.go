package access

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

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

	providers, err := data.Count[models.Provider](db.Unscoped(), data.NotName(models.InternalInfraProviderName))
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
func Signup(c *gin.Context, name, password string) (*models.Identity, error) {
	// no authorization is setup yet
	db := getDB(c)

	identity := &models.Identity{Name: name}

	if err := data.CreateIdentity(db, identity); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	_, err = CreateProviderUser(c, InfraProvider(c), identity)
	if err != nil {
		return nil, fmt.Errorf("create provider user")
	}

	credential := &models.Credential{
		IdentityID:   identity.ID,
		PasswordHash: hash,
	}

	if err := data.CreateCredential(db, credential); err != nil {
		return nil, err
	}

	grant := &models.Grant{
		Subject:   uid.NewIdentityPolymorphicID(identity.ID),
		Privilege: models.InfraAdminRole,
		Resource:  "infra",
		CreatedBy: identity.ID,
	}

	if err := data.CreateGrant(db, grant); err != nil {
		return nil, err
	}

	return identity, nil
}

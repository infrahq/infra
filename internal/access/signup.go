package access

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// Signup creates a user identity using the supplied name and password and
// grants the identity "admin" access to Infra.
func Signup(c *gin.Context, name, password string) (*models.Identity, error) {
	// no authorization is setup yet
	db := getDB(c)

	err := checkPasswordRequirements(db, password)
	if err != nil {
		return nil, err
	}

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
			return nil, err
		}
	}

	return identity, nil
}

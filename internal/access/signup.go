package access

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// Signup creates an organization and user identity using the supplied name and password and
// grants the identity "admin" access to Infra.
func Signup(c *gin.Context, orgName, domain, name, password string) (*models.Identity, *models.Organization, error) {
	// no authorization is setup yet
	db := getDB(c)

	org := &models.Organization{Name: orgName}
	org.SetDefaultDomain()
	if err := data.CreateOrganization(db, org); err != nil {
		return nil, nil, err
	}
	c.Set("org", org)
	db.Statement.Context = context.WithValue(db.Statement.Context, data.OrgCtxKey{}, org)

	err := checkPasswordRequirements(db, org, password)
	if err != nil {
		return nil, nil, fmt.Errorf("password requirements: %w", err)
	}

	identity := &models.Identity{
		Model: models.Model{OrganizationID: org.ID},
		Name:  name,
	}

	if err := data.CreateIdentity(db, identity); err != nil {
		return nil, nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	_, err = CreateProviderUser(c, InfraProvider(c), identity)
	if err != nil {
		return nil, nil, fmt.Errorf("create provider user: %w", err)
	}

	credential := &models.Credential{
		Model: models.Model{
			OrganizationID: org.ID,
		},
		IdentityID:   identity.ID,
		PasswordHash: hash,
	}

	if err := data.CreateCredential(db, credential); err != nil {
		return nil, nil, err
	}

	grants := []*models.Grant{
		{
			Model: models.Model{
				OrganizationID: org.ID,
			},
			Subject:   uid.NewIdentityPolymorphicID(identity.ID),
			Privilege: models.InfraAdminRole,
			Resource:  ResourceInfraAPI,
			CreatedBy: identity.ID,
		},
		{
			Model: models.Model{
				OrganizationID: org.ID,
			},
			Subject:   uid.NewIdentityPolymorphicID(identity.ID),
			Privilege: models.InfraSupportAdminRole,
			Resource:  ResourceInfraAPI,
			CreatedBy: identity.ID,
		},
	}

	for _, grant := range grants {
		if err := data.CreateGrant(db, grant); err != nil {
			return nil, nil, err
		}
	}

	return identity, org, nil
}

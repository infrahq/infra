package access

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/patch"
	"github.com/infrahq/infra/uid"
)

func setupDB(t *testing.T) *gorm.DB {
	driver, err := data.NewSQLiteDriver("file::memory:")
	assert.NilError(t, err)

	patch.ModelsSymmetricKey(t)
	db, err := data.NewDB(driver, nil)
	assert.NilError(t, err)

	return db
}

func setupAccessTestContext(t *testing.T) (*gin.Context, *gorm.DB, *models.Provider) {
	// setup db and context
	db := setupDB(t)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("db", db)

	admin := &models.Identity{Name: "admin@example.com"}
	err := data.CreateIdentity(db, admin)
	assert.NilError(t, err)

	c.Set("identity", admin)

	adminGrant := &models.Grant{
		Subject:   admin.PolyID(),
		Privilege: models.InfraAdminRole,
		Resource:  ResourceInfraAPI,
	}
	err = data.CreateGrant(db, adminGrant)
	assert.NilError(t, err)

	provider := &models.Provider{Name: models.InternalInfraProviderName}
	err = data.CreateProvider(db, provider)
	assert.NilError(t, err)

	identity := &models.Identity{Name: models.InternalInfraConnectorIdentityName}
	err = data.CreateIdentity(db, identity)
	assert.NilError(t, err)

	return c, db, provider
}

var (
	tom       = &models.Identity{Name: "tom@infrahq.com"}
	tomsGroup = &models.Group{Name: "tom's group"}
)

func TestBasicGrant(t *testing.T) {
	db := setupDB(t)
	err := data.CreateIdentity(db, tom)
	assert.NilError(t, err)

	grant(t, db, tom, "i:steven", "read", "infra.groups.1")
	can(t, db, "i:steven", "read", "infra.groups.1")
	cant(t, db, "i:steven", "read", "infra.groups")
	cant(t, db, "i:steven", "read", "infra.groups.2")
	cant(t, db, "i:steven", "write", "infra.groups.1")

	grant(t, db, tom, "i:bob", "read", "infra.groups")
	can(t, db, "i:bob", "read", "infra.groups")
	cant(t, db, "i:bob", "read", "infra.groups.1") // currently we check for exact grant match, this may change as grants evolve
	cant(t, db, "i:bob", "write", "infra.groups")

	grant(t, db, tom, "i:alice", "read", "infra.machines")
	can(t, db, "i:alice", "read", "infra.machines")
	cant(t, db, "i:alice", "read", "infra")
	cant(t, db, "i:alice", "read", "infra.machines.1")
	cant(t, db, "i:alice", "write", "infra.machines")
}

func TestUsersGroupGrant(t *testing.T) {
	db := setupDB(t)

	tom = &models.Identity{Name: "tom@infrahq.com"}
	tomsGroup = &models.Group{Name: "tom's group"}
	provider := &models.Provider{Name: models.InternalInfraProviderName}

	err := data.CreateProvider(db, provider)
	assert.NilError(t, err)

	err = data.CreateIdentity(db, tom)
	assert.NilError(t, err)

	_, err = data.CreateProviderUser(db, provider, tom)
	assert.NilError(t, err)

	err = data.CreateGroup(db, tomsGroup)
	assert.NilError(t, err)

	err = data.AssignIdentityToGroups(db, tom, provider, []string{tomsGroup.Name})
	assert.NilError(t, err)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("db", db)
	c.Set("identity", tom)

	authDB, err := RequireInfraRole(c, models.InfraAdminRole)
	assert.ErrorIs(t, err, internal.ErrForbidden)
	assert.Assert(t, authDB == nil)

	grant(t, db, tom, tomsGroup.PolyID(), models.InfraAdminRole, "infra")

	authDB, err = RequireInfraRole(c, models.InfraAdminRole)
	assert.NilError(t, err)
	assert.Assert(t, authDB != nil)
}

func TestInfraRequireInfraRole(t *testing.T) {
	db := setupDB(t)

	setup := func(t *testing.T, infraRole string) *gin.Context {
		testIdentity := &models.Identity{Name: fmt.Sprintf("infra-%s-%s", infraRole, time.Now())}

		err := data.CreateIdentity(db, testIdentity)
		assert.NilError(t, err)

		err = data.CreateGrant(db, &models.Grant{Subject: testIdentity.PolyID(), Privilege: infraRole, Resource: ResourceInfraAPI})
		assert.NilError(t, err)

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("db", db)
		c.Set("identity", testIdentity)

		return c
	}

	t.Run("has specific required role", func(t *testing.T) {
		c := setup(t, models.InfraAdminRole)

		authDB, err := RequireInfraRole(c, models.InfraAdminRole)
		assert.NilError(t, err)
		assert.Assert(t, authDB != nil)
	})

	t.Run("does not have specific required role", func(t *testing.T) {
		c := setup(t, models.InfraViewRole)

		authDB, err := RequireInfraRole(c, models.InfraAdminRole)
		assert.Error(t, err, "forbidden: requestor does not have required grant")
		assert.Assert(t, authDB == nil)
	})

	t.Run("has required role in list", func(t *testing.T) {
		c := setup(t, models.InfraViewRole)

		authDB, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraViewRole)
		assert.NilError(t, err)
		assert.Assert(t, authDB != nil)
	})

	t.Run("does not have required role in list", func(t *testing.T) {
		c := setup(t, models.InfraViewRole)

		authDB, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraConnectorRole)
		assert.Error(t, err, "forbidden: requestor does not have required grant")
		assert.Assert(t, authDB == nil)
	})
}

func grant(t *testing.T, db *gorm.DB, currentUser *models.Identity, subject uid.PolymorphicID, privilege, resource string) {
	err := data.CreateGrant(db, &models.Grant{
		Subject:   subject,
		Privilege: privilege,
		Resource:  resource,
		CreatedBy: currentUser.ID,
	})
	assert.NilError(t, err)
}

func can(t *testing.T, db *gorm.DB, subject uid.PolymorphicID, privilege, resource string) {
	canAccess, err := Can(db, subject, privilege, resource)
	assert.NilError(t, err)
	assert.Assert(t, canAccess)
}

func cant(t *testing.T, db *gorm.DB, subject uid.PolymorphicID, privilege, resource string) {
	canAccess, err := Can(db, subject, privilege, resource)
	assert.NilError(t, err)
	assert.Assert(t, !canAccess)
}

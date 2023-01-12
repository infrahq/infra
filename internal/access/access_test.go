package access

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/database"
	"github.com/infrahq/infra/internal/testing/patch"
	"github.com/infrahq/infra/uid"
)

func setupDB(t *testing.T) *data.DB {
	t.Helper()
	patch.ModelsSymmetricKey(t)
	db, err := data.NewDB(data.NewDBOptions{DSN: database.PostgresDriver(t, "_access").DSN})
	assert.NilError(t, err)
	return db
}

func setupAccessTestContext(t *testing.T) (*gin.Context, *data.Transaction, *models.Provider) {
	// setup db and context
	db := setupDB(t)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	tx := txnForTestCase(t, db)

	admin := &models.Identity{Name: "admin@example.com"}
	err := data.CreateIdentity(tx, admin)
	assert.NilError(t, err)

	c.Set(RequestContextKey, RequestContext{
		DBTxn:         tx,
		Authenticated: Authenticated{User: admin},
	})

	adminGrant := &models.Grant{
		Subject:   uid.NewIdentityPolymorphicID(admin.ID),
		Privilege: models.InfraAdminRole,
		Resource:  ResourceInfraAPI,
	}
	err = data.CreateGrant(tx, adminGrant)
	assert.NilError(t, err)

	provider := data.InfraProvider(tx)

	return c, tx, provider
}

func txnForTestCase(t *testing.T, db *data.DB) *data.Transaction {
	t.Helper()
	tx, err := db.Begin(context.Background(), nil)
	assert.NilError(t, err)
	t.Cleanup(func() {
		assert.NilError(t, tx.Rollback())
	})
	return tx.WithOrgID(db.DefaultOrg.ID)
}

func TestAuthorize(t *testing.T) {
	db := setupDB(t)

	admin := &models.Identity{Name: "admin@infrahq.com"}
	err := data.CreateIdentity(db, admin)
	assert.NilError(t, err)

	grant(t, db, admin, "i:steven", "read", ResourceInfraAPI)
	can(t, db, "steven", "read")
	cant(t, db, "steven", "write")

	grant(t, db, admin, "i:a11ce", "write", ResourceInfraAPI)
	cant(t, db, "a11ce", "read")
	can(t, db, "a11ce", "write")
}

func TestRequireInfraRole_GrantsFromGroupMembership(t *testing.T) {
	db := setupDB(t)

	tom := &models.Identity{Name: "tom@infrahq.com"}
	tomsGroup := &models.Group{Name: "tom's group"}
	provider := data.InfraProvider(db)

	err := data.CreateIdentity(db, tom)
	assert.NilError(t, err)

	user, err := data.CreateProviderUser(db, provider, tom)
	assert.NilError(t, err)

	err = data.CreateGroup(db, tomsGroup)
	assert.NilError(t, err)

	_, err = data.AssignIdentityToGroups(db, user, []string{tomsGroup.Name})
	assert.NilError(t, err)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	tx := txnForTestCase(t, db)
	c.Set(RequestContextKey, RequestContext{
		DBTxn:         tx,
		Authenticated: Authenticated{User: tom},
	})
	authDB, err := RequireInfraRole(c, models.InfraAdminRole)
	assert.ErrorIs(t, err, ErrNotAuthorized)
	assert.Assert(t, authDB == nil)

	admin := &models.Identity{Model: models.Model{ID: uid.ID(512)}}
	grant(t, tx, admin, uid.NewGroupPolymorphicID(tomsGroup.ID), models.InfraAdminRole, "infra")

	authDB, err = RequireInfraRole(c, models.InfraAdminRole)
	assert.NilError(t, err)
	assert.Assert(t, authDB != nil)
}

func TestRequireInfraRole(t *testing.T) {
	db := setupDB(t)

	setup := func(t *testing.T, infraRole string) *gin.Context {
		testIdentity := &models.Identity{Name: fmt.Sprintf("infra-%s-%s", infraRole, time.Now())}

		err := data.CreateIdentity(db, testIdentity)
		assert.NilError(t, err)

		err = data.CreateGrant(db, &models.Grant{Subject: uid.NewIdentityPolymorphicID(testIdentity.ID), Privilege: infraRole, Resource: ResourceInfraAPI})
		assert.NilError(t, err)

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		tx := txnForTestCase(t, db)
		c.Set(RequestContextKey, RequestContext{
			DBTxn:         tx,
			Authenticated: Authenticated{User: testIdentity},
		})
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
		assert.ErrorIs(t, err, ErrNotAuthorized)
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
		assert.ErrorIs(t, err, ErrNotAuthorized)
		assert.Assert(t, authDB == nil)
	})
}

func grant(t *testing.T, db data.WriteTxn, createdBy *models.Identity, subject uid.PolymorphicID, privilege, resource string) {
	err := data.CreateGrant(db, &models.Grant{
		Subject:   subject,
		Privilege: privilege,
		Resource:  resource,
		CreatedBy: createdBy.ID,
	})
	assert.NilError(t, err)
}

func can(t *testing.T, db *data.DB, subject string, privilege string) {
	t.Helper()
	id, err := uid.Parse([]byte(subject))
	assert.NilError(t, err)
	rCtx := RequestContext{DBTxn: txnForTestCase(t, db)}
	rCtx.Authenticated.User = &models.Identity{Model: models.Model{ID: id}}
	err = IsAuthorized(rCtx, privilege)
	assert.NilError(t, err)
}

func cant(t *testing.T, db *data.DB, subject string, privilege string) {
	id, err := uid.Parse([]byte(subject))
	assert.NilError(t, err)
	rCtx := RequestContext{DBTxn: txnForTestCase(t, db)}
	rCtx.Authenticated.User = &models.Identity{Model: models.Model{ID: id}}
	err = IsAuthorized(rCtx, privilege)
	assert.ErrorIs(t, err, ErrNotAuthorized)
}

func TestAuthorizationError(t *testing.T) {
	t.Run("one role", func(t *testing.T) {
		err := AuthorizationError{
			Operation:     "create",
			Resource:      "access key",
			RequiredRoles: []string{"admin"},
		}
		expected := "you do not have permission to create access key, requires role admin"
		assert.Equal(t, err.Error(), expected)
	})
	t.Run("two roles", func(t *testing.T) {
		err := AuthorizationError{
			Operation:     "list",
			Resource:      "users",
			RequiredRoles: []string{"admin", "view"},
		}
		expected := "you do not have permission to list users, requires role admin, or view"
		assert.Equal(t, err.Error(), expected)
	})
	t.Run("three roles", func(t *testing.T) {
		err := AuthorizationError{
			Operation:     "add",
			Resource:      "destination",
			RequiredRoles: []string{"admin", "view", "connector"},
		}
		expected := "you do not have permission to add destination, requires role admin, view, or connector"
		assert.Equal(t, err.Error(), expected)
	})
	t.Run("is ErrNotAuthorized", func(t *testing.T) {
		err := AuthorizationError{}
		assert.Assert(t, errors.Is(err, ErrNotAuthorized))
	})
}

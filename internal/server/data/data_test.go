package data

import (
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/database"
	"github.com/infrahq/infra/internal/testing/patch"
	"github.com/infrahq/infra/uid"
)

func setupDB(t *testing.T, driver gorm.Dialector) *gorm.DB {
	t.Helper()
	patch.ModelsSymmetricKey(t)

	db, err := NewDB(driver, nil)
	assert.NilError(t, err)

	logging.PatchLogger(t, zerolog.NewTestWriter(t))
	t.Cleanup(InvalidateCache)

	return db.DB
}

var isEnvironmentCI = os.Getenv("CI") != ""

// postgresDriver requires postgres to be available in a CI environment, and
// marks the test as skipped when not in CI environment.
func postgresDriver(t *testing.T) gorm.Dialector {
	driver := database.PostgresDriver(t, "")
	switch {
	case driver == nil && isEnvironmentCI:
		t.Fatal("CI must test all drivers, set POSTGRESQL_CONNECTION")
	case driver == nil:
		t.Skip("Set POSTGRESQL_CONNECTION to test against postgresql")
	}
	return driver.Dialector
}

// runDBTests against all supported databases. Defaults to only sqlite locally,
// and all supported DBs in CI.
// Set POSTGRESQL_CONNECTION to a postgresql connection string to run tests
// against postgresql.
func runDBTests(t *testing.T, run func(t *testing.T, db *gorm.DB)) {
	t.Run("postgres", func(t *testing.T) {
		pgsql := postgresDriver(t)
		db := setupDB(t, pgsql)
		run(t, db)
		db.Rollback()
	})
}

func TestSnowflakeIDSerialization(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		id := uid.New()
		g := &models.Group{Model: models.Model{ID: id}, Name: "Foo"}
		err := db.Create(g).Error
		assert.NilError(t, err)

		var group models.Group
		err = db.First(&group, &models.Group{Name: "Foo"}).Error
		assert.NilError(t, err)
		assert.Assert(t, 0 != group.ID)

		var intID int64
		err = db.Select("id").Table("groups").Scan(&intID).Error
		assert.NilError(t, err)

		assert.Equal(t, int64(id), intID)
	})
}

func TestDatabaseSelectors(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {

		// assert.NilError(t, initializeSchema(db))

		org := OrgFromContext(db.Statement.Context)
		assert.Assert(t, org != nil)
		// mimic server.DatabaseMiddleware
		withCtx := db.WithContext(context.Background())

		// normally we don't want the default org to go through to the withCtx that the database middleware makes, but our tests won't work without it.
		withCtx.Statement.Context = WithOrg(withCtx.Statement.Context, org)

		org = OrgFromContext(withCtx.Statement.Context)
		assert.Assert(t, org != nil)

		assert.Assert(t, db != withCtx, "db=%p withCtx=%p", db, withCtx)

		err := withCtx.Transaction(func(tx *gorm.DB) error {
			assert.Assert(t, withCtx != tx, "db=%p tx=%p", withCtx, tx)

			org = OrgFromContext(tx.Statement.Context)
			assert.Assert(t, org != nil)

			// query using one of our helpers and selectors
			_, err := ListGrants(tx, nil, ByID(534))
			assert.NilError(t, err)

			// query with Model and Where
			var groups []models.Group
			qDB := tx.Model(&models.Group{}).Where("id = ?", 42).Find(&groups)
			assert.NilError(t, qDB.Error)
			assert.Assert(t, tx != qDB, "tx=%p queryDB=%p", tx, qDB)

			// Show that queries have not modified the original gorm.DB references
			assert.Equal(t, len(db.Statement.Clauses), 0)
			assert.Equal(t, len(withCtx.Statement.Clauses), 0)
			assert.Equal(t, len(tx.Statement.Clauses), 0)
			return nil
		})
		assert.NilError(t, err)

		// query using one of our helpers and selectors
		_, err = ListGrants(db, nil, ByID(534))
		assert.NilError(t, err)

		// query with Model and Where
		var groups []models.Group
		qDB := db.Model(&models.Group{}).Where("id = ?", 42).Find(&groups)
		assert.NilError(t, qDB.Error)
		assert.Assert(t, db != qDB, "db=%p queryDB=%p", db, qDB)
		t.Logf("DB pointer: %p", qDB)

		// Show that queries have not modified the original gorm.DB references
		assert.Equal(t, len(db.Statement.Clauses), 0)
		assert.Equal(t, len(withCtx.Statement.Clauses), 0)
	})
}

func TestPaginationSelector(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		letters := make([]string, 0, 26)
		for r := 'a'; r < 'a'+26; r++ {
			letters = append(letters, string(r))
			g := &models.Identity{Name: string(r)}
			assert.NilError(t, CreateIdentity(db, g))
		}

		p := models.Pagination{Page: 1, Limit: 10}

		actual, err := ListIdentities(db, &p)
		assert.NilError(t, err)
		assert.Equal(t, len(actual), 10)
		for i := 0; i < p.Limit; i++ {
			assert.Equal(t, letters[i+(p.Page-1)*p.Limit], actual[i].Name)
		}

		p.Page = 2
		actual, err = ListIdentities(db, &p)
		assert.NilError(t, err)
		assert.Equal(t, len(actual), 10)
		for i := 0; i < p.Limit; i++ {
			assert.Equal(t, letters[i+(p.Page-1)*p.Limit], actual[i].Name)
		}

		p.Page = 3
		actual, err = ListIdentities(db, &p)
		assert.NilError(t, err)
		assert.Equal(t, len(actual), 6)

		for i := 0; i < 6; i++ {
			assert.Equal(t, letters[i+(p.Page-1)*p.Limit], actual[i].Name)
		}

		p.Page, p.Limit = 1, 26
		actual, err = ListIdentities(db, &p)
		assert.NilError(t, err)
		for i, user := range actual {
			assert.Equal(t, user.Name, letters[i])
		}
	})
}

func TestDefaultSortFromType(t *testing.T) {
	assert.Equal(t, getDefaultSortFromType(new(models.AccessKey)), "name ASC")
	assert.Equal(t, getDefaultSortFromType(new(models.Destination)), "name ASC")
	assert.Equal(t, getDefaultSortFromType(new(models.Grant)), "id ASC")
	assert.Equal(t, getDefaultSortFromType(new(models.Group)), "name ASC")
	assert.Equal(t, getDefaultSortFromType(new(models.Provider)), "name ASC")
	assert.Equal(t, getDefaultSortFromType(new(models.Identity)), "name ASC")
}

func TestCreateTransactionError(t *testing.T) {
	// on creation error (such as conflict) the database transaction should still be usable
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		err := db.Transaction(func(tx *gorm.DB) error {
			g := &models.Grant{}
			err := add(tx, g)
			if err != nil {
				return err
			}

			// attempt to re-create, which results in a conflict
			err = add(tx, g)
			assert.ErrorContains(t, err, "already exists")

			// the same transaction should still be usable
			_, err = get[models.Grant](tx, ByID(g.ID))
			return err
		})

		assert.NilError(t, err)
	})
}

func TestSetOrg(t *testing.T) {
	model := &models.AccessKey{}
	org := &models.Organization{}
	org.ID = 123456

	db := &gorm.DB{}
	db.Statement = &gorm.Statement{
		Context: WithOrg(context.Background(), org),
	}
	setOrg(db, model)
	assert.Equal(t, model.OrganizationID, uid.ID(123456))
}

package data

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/patch"
	"github.com/infrahq/infra/uid"
)

func setupDB(t *testing.T, driver gorm.Dialector) *gorm.DB {
	t.Helper()
	patch.ModelsSymmetricKey(t)

	db, err := NewDB(driver, nil)
	assert.NilError(t, err)

	org := &models.Organization{
		Name:   "Database Co",
		Domain: "database.local",
	}
	err = CreateOrganization(db, org)
	assert.NilError(t, err)

	db.Statement.Context = context.WithValue(db.Statement.Context, "org", org)

	InfraProvider(db)

	logging.PatchLogger(t, zerolog.NewTestWriter(t))
	t.Cleanup(InvalidateCache)

	return db
}

// dbDrivers returns the list of database drivers to test.
// Set POSTGRESQL_CONNECTION to a postgresql connection string to run tests
// against postgresql.
func dbDrivers(t *testing.T) []gorm.Dialector {
	t.Helper()
	var drivers []gorm.Dialector

	tmp := t.TempDir()
	sqlite, err := NewSQLiteDriver(filepath.Join(tmp, t.Name()))
	assert.NilError(t, err, "sqlite driver")
	drivers = append(drivers, sqlite)

	pgConn, ok := os.LookupEnv("POSTGRESQL_CONNECTION")
	switch {
	case ok:
		pgsql, err := NewPostgresDriver(pgConn)
		assert.NilError(t, err, "postgresql driver")

		db, err := gorm.Open(pgsql)
		assert.NilError(t, err, "connect to postgresql")
		t.Cleanup(func() {
			assert.NilError(t, db.Exec("DROP SCHEMA testing CASCADE").Error)
		})
		assert.NilError(t, db.Exec("CREATE SCHEMA testing").Error)

		pgsql, err = NewPostgresDriver(pgConn + " search_path=testing")
		assert.NilError(t, err, "postgresql driver")

		drivers = append(drivers, pgsql)
	case os.Getenv("CI") != "":
		t.Fatalf("CI must test all drivers, set POSTGRESQL_CONNECTION")
	}

	return drivers
}

// runDBTests against all supported databases. Defaults to only sqlite locally,
// and all supported DBs in CI. See dbDrivers to test other databases locally.
func runDBTests(t *testing.T, run func(t *testing.T, db *gorm.DB)) {
	for _, driver := range dbDrivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			run(t, setupDB(t, driver))
		})
	}
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

func TestPaginationSelector(t *testing.T) {
	letters := make([]string, 0, 26)
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		for r := 'a'; r < 'a'+26; r++ {
			letters = append(letters, string(r))
			g := &models.Identity{Name: string(r)}
			err := add(db, g)
			assert.NilError(t, err)
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

package data

import (
	"context"
	"os"
	"testing"

	"github.com/infrahq/secrets"
	"go.uber.org/zap/zaptest"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func setup(t *testing.T) *gorm.DB {
	driver, err := NewSQLiteDriver("file::memory:")
	assert.NilError(t, err)

	db, err := NewDB(driver)
	assert.NilError(t, err)

	fp := secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{
		Path: os.TempDir(),
	})

	kp := secrets.NewNativeKeyProvider(fp)

	key, err := kp.GenerateDataKey("")
	assert.NilError(t, err)

	models.SymmetricKey = key

	err = db.Create(&models.Provider{Name: models.InternalInfraProviderName}).Error
	assert.NilError(t, err)

	setupLogging(t)
	t.Cleanup(InvalidateCache)

	return db
}

func setupLogging(t *testing.T) {
	origL := logging.L
	logging.L = zaptest.NewLogger(t)
	logging.S = logging.L.Sugar()
	t.Cleanup(func() {
		logging.L = origL
		logging.S = logging.L.Sugar()
	})
}

func TestSnowflakeIDSerialization(t *testing.T) {
	db := setup(t)

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
}

func TestDatabaseSelectors(t *testing.T) {
	driver, err := NewSQLiteDriver("file::memory:")
	assert.NilError(t, err)

	db, err := NewRawDB(driver)
	assert.NilError(t, err)
	t.Logf("DB pointer: %p", db)

	assert.NilError(t, initializeSchema(db))

	// mimic server.DatabaseMiddleware
	withCtx := db.WithContext(context.Background())
	assert.Assert(t, db != withCtx, "db=%p withCtx=%p", db, withCtx)
	t.Logf("DB pointer: %p", withCtx)

	err = withCtx.Transaction(func(tx *gorm.DB) error {
		assert.Assert(t, withCtx != tx, "db=%p tx=%p", withCtx, tx)
		t.Logf("DB pointer: %p", tx)

		// query using one of our helpers and selectors
		_, err := ListGrants(tx, ByID(534))
		assert.NilError(t, err)

		// query with Model and Where
		var groups []models.Group
		qDB := tx.Model(&models.Group{}).Where("id = ?", 42).Find(&groups)
		assert.NilError(t, qDB.Error)
		assert.Assert(t, tx != qDB, "tx=%p queryDB=%p", tx, qDB)
		t.Logf("DB pointer: %p", qDB)

		// Show that queries have not modified the original gorm.DB references
		assert.Equal(t, len(db.Statement.Clauses), 0)
		assert.Equal(t, len(withCtx.Statement.Clauses), 0)
		assert.Equal(t, len(tx.Statement.Clauses), 0)
		return nil
	})
	assert.NilError(t, err)

	// query using one of our helpers and selectors
	_, err = ListGrants(db, ByID(534))
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
}

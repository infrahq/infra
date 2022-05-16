package data

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/infrahq/secrets"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

// see loadSQL for setting up your own migration test
func Test202204111503(t *testing.T) {
	db := setupWithNoMigrations(t, func(db *gorm.DB) {
		loadSQL(t, db, "202204111503")
	})

	err := migrate(db)
	assert.NilError(t, err)

	ids, err := ListIdentities(db, ByName("steven@example.com"))
	assert.NilError(t, err)

	assert.Assert(t, len(ids) == 1)
}

func Test202204211705(t *testing.T) {
	db := setupWithNoMigrations(t, func(db *gorm.DB) {
		loadSQL(t, db, "202204211705")
	})

	key, err := tmpSymmetricKey()
	assert.NilError(t, err)

	models.SymmetricKey = key
	t.Cleanup(func() {
		models.SymmetricKey = nil
	})

	err = migrate(db)
	assert.NilError(t, err)

	// check it still works
	settings, err := GetSettings(db)
	assert.NilError(t, err)

	assert.Assert(t, settings != nil)
	assert.Assert(t, settings.PrivateJWK[0] == '{') // unencrypted type is json string.

	// check the storage data
	type Settings struct {
		models.Model
		PrivateJWK []byte
	}
	rawSettings := Settings{}
	err = db.Model(rawSettings).Where("id = ?", settings.ID).First(&rawSettings).Error
	assert.NilError(t, err)

	assert.Assert(t, rawSettings.PrivateJWK[0] != '{')
}

func tmpSymmetricKey() (*secrets.SymmetricKey, error) {
	sp := secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{
		Path: os.TempDir(),
	})

	rootKey := "db_at_rest"
	symmetricKeyProvider := secrets.NewNativeKeyProvider(sp)
	return symmetricKeyProvider.GenerateDataKey(rootKey)
}

// loadSQL loads a sql file from disk by a file name matching the migration it's meant to test.
// to create a new file for testing a migration:
//
// 1. do whatever you need to get the db in the state you want to test. it might be helpful to capture the db state before writing your migration. Make sure there are some relevant records in the affected tables.
//
// 2. connect up to the db to dump out the data. if you're running sqlite in kubernetes, this will look like this:
//   kubectl exec -it deployment/infra-server -- apk add sqlite
//   kubectl exec -it deployment/infra-server -- /usr/bin/sqlite3 /var/lib/infrahq/server/sqlite3.db
// at the prompt, do:
//   .dump
// and copy the results. Copy them to a file with the same name of the migration and a .sql extension in the migrationdata/ folder.
//
// 3. write the migration and test that it does what you expect. It can be helpful to put any necessary guards in place to make sure the database is in the state you expect. sometimes failed migrations leave it in a broken state, and might run when you don't expect, so defensive programming is helpful here.
//
// 4. go back to the sql file:
//   - remove any SQL records that aren't relevant to the test
//   - blank out provider client ids and name
//   - remove any email addresses and replace with @example.com
//   - remove any provider_users redirect urls
// any other sensitive fields are encrypted and the key isn't included in the database.
// Make sure the test still passes.
func loadSQL(t *testing.T, db *gorm.DB, filename string) {
	f, err := os.Open("migrationdata/" + filename + ".sql")
	assert.NilError(t, err)

	b, err := ioutil.ReadAll(f)
	assert.NilError(t, err)

	err = db.Exec(string(b)).Error
	assert.NilError(t, err)
}

func setupWithNoMigrations(t *testing.T, f func(db *gorm.DB)) *gorm.DB {
	driver, err := NewSQLiteDriver("file::memory:")
	assert.NilError(t, err)

	db, err := newRawDB(driver)
	assert.NilError(t, err)

	f(db)

	models.SkipSymmetricKey = true
	t.Cleanup(func() {
		models.SkipSymmetricKey = false
	})

	setupLogging(t)

	return db
}

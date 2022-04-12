package data

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/infrahq/infra/internal/server/models"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
)

// see loadSQL for setting up your own migration test
func Test202204111503(t *testing.T) {
	db := setupWithNoMigrations(t, func(db *gorm.DB) {
		loadSQL(t, db, "202204111503")
	})

	err := migrate(db)
	assert.NilError(t, err)

	ids, err := ListIdentities(db, ByName("steven.soroka@infrahq.com"))
	assert.NilError(t, err)

	assert.Assert(t, len(ids) == 1)
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
// and copy the results. Copy them to a file with the same name of the migration and a .sql extention in the migrationdata/ folder.
//
// 3. write the migration and test that it does what you expect. It can be helpful to put any necessary guards in place to make sure the database is in the state you expect. sometimes failed migrations leave it in a broken state, and might run when you don't expect, so defensive programming is helpful here.
//
// 4. ideally, remove any SQL records that aren't relevant to the test and make sure everything still works.
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
	// keyRec, err := GetEncryptionKey(db, ByName("dbkey"))
	// assert.NilError(t, err)

	// fp := secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{})

	// kp := secrets.NewNativeSecretProvider(fp)
	// key, err := kp.DecryptDataKey("migrationdata/202204111503.key", keyRec.Encrypted)
	// assert.NilError(t, err)

	// models.SymmetricKey = key

	setupLogging(t)

	return db
}

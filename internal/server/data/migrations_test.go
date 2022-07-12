package data

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	"k8s.io/utils/strings/slices"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/patch"
)

// see loadSQL for setting up your own migration test
func TestMigration_202204111503(t *testing.T) {
	driver := setupWithNoMigrations(t, func(db *gorm.DB) {
		loadSQL(t, db, "202204111503")
	})

	// migration runs as part of NewDB
	db, err := NewDB(driver, nil)
	assert.NilError(t, err)

	ids, err := ListIdentities(db, &models.Pagination{}, ByName("steven@example.com"))
	assert.NilError(t, err)
	assert.Assert(t, len(ids) == 1)
	// check that merged identity has unique grants
	grants, err := ListGrants(db, &models.Pagination{}, BySubject(ids[0].PolyID()))
	assert.NilError(t, err)
	assert.Assert(t, len(grants) == 1)

}

func TestMigration_202204211705(t *testing.T) {
	driver := setupWithNoMigrations(t, func(db *gorm.DB) {
		loadSQL(t, db, "202204211705")
	})

	// migration runs as part of NewDB
	db, err := NewDB(driver, nil)
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

var cmpProviderShallow = gocmp.Comparer(func(x, y models.Provider) bool {
	return x.Name == y.Name && x.Kind == y.Kind && x.URL == y.URL
})

// loadSQL loads a sql file from disk by a file name matching the migration it's meant to test.
// To create a new file for testing a migration:
//
// 1. Start an infra server and perform operations with the CLI or API to
//    get the db in the state you want to test. You should capture the db state
//    before writing your migration. Make sure there are some relevant records
//    in the affected tables. Do not use a production server! Any sensitive data
//    should be throw away development credentials because they will be checked
//    into git.
//
// 2. Connect up to the db to dump the data. If you're running sqlite in
//    kubernetes:
//
//   kubectl exec -it deployment/infra-server -- apk add sqlite
//   kubectl exec -it deployment/infra-server -- /usr/bin/sqlite3 /var/lib/infrahq/server/sqlite3.db
//
//   at the prompt, do:
//     .dump
//   Copy to output to a file with the same name of the migration and a .sql
//   extension in the migrationdata/ folder.
//
//   Or from a local db:
//
//   echo -e ".output dump.sql\n.dump" | sqlite3 sqlite3.db
//
// 3. Write the migration and test that it does what you expect. It can be helpful
//    to put any necessary guards in place to make sure the database is in the state
//    you expect. Sometimes failed migrations leave it in a broken state, and might
//    run when you don't expect, so defensive programming is helpful here.
//
func loadSQL(t *testing.T, db *gorm.DB, filename string) {
	f, err := os.Open("migrationdata/" + filename + ".sql")
	assert.NilError(t, err)

	b, err := ioutil.ReadAll(f)
	assert.NilError(t, err)

	err = db.Exec(string(b)).Error
	assert.NilError(t, err)
}

func setupWithNoMigrations(t *testing.T, f func(db *gorm.DB)) gorm.Dialector {
	dir := t.TempDir()
	driver, err := NewSQLiteDriver(filepath.Join(dir, "sqlite3.db"))
	assert.NilError(t, err)

	db, err := newRawDB(driver)
	assert.NilError(t, err)

	f(db)

	patch.ModelsSymmetricKey(t)
	logging.PatchLogger(t, zerolog.NewTestWriter(t))

	return driver
}

func TestMigration_DropCertificateTables(t *testing.T) {
	driver := setupWithNoMigrations(t, func(db *gorm.DB) {
		loadSQL(t, db, "202206161733")
	})

	db, err := NewDB(driver, nil)
	assert.NilError(t, err)

	assert.Assert(t, !tableExists(t, db, "trusted_certificates"))
	assert.Assert(t, !tableExists(t, db, "root_certificates"))
}

func tableExists(t *testing.T, db *gorm.DB, name string) bool {
	t.Helper()
	var count int
	err := db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", name).Row().Scan(&count)
	assert.NilError(t, err)
	return count > 0
}

func TestMigration_AddKindToProvider(t *testing.T) {
	for _, driver := range dbDrivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			db, err := newRawDB(driver)
			assert.NilError(t, err)

			loadSQL(t, db, "202206151027-"+driver.Name())

			db, err = NewDB(driver, nil)
			assert.NilError(t, err)

			var providers []models.Provider
			err = db.Omit("client_secret").Find(&providers).Error
			assert.NilError(t, err)
			expected := []models.Provider{
				{Name: "infra", Kind: models.ProviderKindInfra},
				{Name: "okta", Kind: models.ProviderKindOkta, URL: "dev.okta.com"},
			}
			assert.DeepEqual(t, providers, expected, cmpProviderShallow)
		})
	}
}

// this test does an external call to example.okta.com, if it fails check your network connection
func TestMigration_AddAuthURLAndScopesToProvider(t *testing.T) {
	for _, driver := range dbDrivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			db, err := newRawDB(driver)
			assert.NilError(t, err)

			loadSQL(t, db, "202206281027-"+driver.Name())

			db, err = NewDB(driver, nil)
			assert.NilError(t, err)

			var providers []models.Provider
			err = db.Omit("client_secret").Find(&providers).Error
			assert.NilError(t, err)

			assert.Equal(t, len(providers), 2)
			authUrls := make(map[string]string)
			scopes := make(map[string][]string)
			for _, p := range providers {
				authUrls[p.Name] = p.AuthURL
				scopes[p.Name] = p.Scopes
			}
			assert.Equal(t, authUrls["infra"], "")
			assert.Equal(t, len(scopes["infra"]), 0)
			assert.Equal(t, authUrls["okta"], "https://example.okta.com/oauth2/v1/authorize")
			assert.Assert(t, slices.Equal(scopes["okta"], []string{"openid", "email", "offline_access", "groups"}))
		})
	}
}

func TestMigration_SetDestinationLastSeenAt(t *testing.T) {
	for _, driver := range dbDrivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			db, err := newRawDB(driver)
			assert.NilError(t, err)

			loadSQL(t, db, "202207041724-"+driver.Name())

			db, err = NewDB(driver, nil)
			assert.NilError(t, err)

			var destinations []models.Destination
			err = db.Find(&destinations).Error
			assert.NilError(t, err)

			for _, destination := range destinations {
				assert.Equal(t, destination.LastSeenAt, destination.UpdatedAt)
			}
		})
	}
}

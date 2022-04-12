package data

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"go.uber.org/zap/zaptest"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
)

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

	logging.L = zaptest.NewLogger(t)
	logging.S = logging.L.Sugar()

	return db
}

package models_test

import (
	"os"
	"testing"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/secrets"
	"github.com/infrahq/infra/uid"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

type StructForTesting struct {
	ID      uid.ID `gorm:"primaryKey"`
	ASecret models.EncryptedAtRest
}

func TestEncryptedAtRest(t *testing.T) {
	var err error
	// secret provider setup
	sp := secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{
		Path: os.TempDir(),
	})

	rootKey := "db_at_rest"
	symmetricKeyProvider := secrets.NewNativeSecretProvider(sp)
	symmetricKey, err := symmetricKeyProvider.GenerateDataKey(rootKey)
	assert.NilError(t, err)

	models.SymmetricKey = symmetricKey

	// test
	driver, err := data.NewSQLiteDriver("file::memory:")
	assert.NilError(t, err)

	db, err := data.NewDB(driver)
	assert.NilError(t, err)

	err = db.AutoMigrate(&StructForTesting{})
	assert.NilError(t, err)

	id := uid.New()

	m := &StructForTesting{
		ID:      id,
		ASecret: "don't tell",
	}

	err = db.Save(m).Error
	assert.NilError(t, err)

	var result string
	err = db.Raw("select a_secret from struct_for_testings where id = ?", id).Scan(&result).Error
	assert.NilError(t, err)

	assert.Assert(t, "don't tell" != result)
	assert.Assert(t, "" != result)
	assert.Assert(t, is.Len(result, 88)) // encrypts to this many bytes

	m2 := &StructForTesting{}

	err = db.Find(m2, db.Where("id = ?", id)).Error
	assert.NilError(t, err)

	assert.Equal(t, "don't tell", string(m2.ASecret))
}

package models_test

import (
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/patch"
	"github.com/infrahq/infra/uid"
)

type StructForTesting struct {
	ID      uid.ID `gorm:"primaryKey"`
	ASecret models.EncryptedAtRest
}

func TestEncryptedAtRest(t *testing.T) {
	patch.ModelsSymmetricKey(t)

	driver, err := data.NewSQLiteDriver("file::memory:")
	assert.NilError(t, err)

	db, err := data.NewDB(driver, nil)
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

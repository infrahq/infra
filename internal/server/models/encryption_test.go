package models_test

import (
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/database"
	"github.com/infrahq/infra/internal/testing/patch"
	"github.com/infrahq/infra/uid"
)

type StructForTesting struct {
	ID      uid.ID `gorm:"primaryKey"`
	ASecret models.EncryptedAtRest
}

func (s StructForTesting) Schema() string {
	return `
CREATE TABLE struct_for_testings (
	id bigint PRIMARY KEY,
	a_secret text
);`
}

func TestEncryptedAtRest(t *testing.T) {
	patch.ModelsSymmetricKey(t)

	pg := database.PostgresDriver(t, "_models")
	db, err := data.NewDB(pg.Dialector, nil)
	assert.NilError(t, err)

	_, err = db.Exec(StructForTesting{}.Schema())
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

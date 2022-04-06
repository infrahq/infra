package data

import (
	"os"
	"testing"

	"go.uber.org/zap/zaptest"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/secrets"
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

	kp := secrets.NewNativeSecretProvider(fp)

	key, err := kp.GenerateDataKey("")
	assert.NilError(t, err)

	models.SymmetricKey = key

	logging.L = zaptest.NewLogger(t)
	logging.S = logging.L.Sugar()

	return db
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

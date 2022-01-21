package data

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/secrets"
	"github.com/infrahq/infra/uid"
)

func setup(t *testing.T) *gorm.DB {
	driver, err := NewSQLiteDriver("file::memory:")
	require.NoError(t, err)

	db, err := NewDB(driver)
	require.NoError(t, err)

	fp := secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{
		Path: os.TempDir(),
	})

	kp := secrets.NewNativeSecretProvider(fp)

	key, err := kp.GenerateDataKey("")
	require.NoError(t, err)

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
	require.NoError(t, err)

	var group models.Group
	err = db.First(&group, &models.Group{Name: "Foo"}).Error
	require.NoError(t, err)
	require.NotEqual(t, 0, group.ID)

	var intID int64
	err = db.Select("id").Table("groups").Scan(&intID).Error
	require.NoError(t, err)

	require.Equal(t, int64(id), intID)
}

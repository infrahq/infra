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

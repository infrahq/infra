package models_test

import (
	"os"
	"testing"

	"github.com/infrahq/infra/uid"
	"github.com/stretchr/testify/require"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/secrets"
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
	require.NoError(t, err)

	models.SymmetricKey = symmetricKey

	// test
	driver, err := data.NewSQLiteDriver("file::memory:")
	require.NoError(t, err)

	db, err := data.NewDB(driver)
	require.NoError(t, err)

	err = db.AutoMigrate(&StructForTesting{})
	require.NoError(t, err)

	id := uid.New()

	m := &StructForTesting{
		ID:      id,
		ASecret: "don't tell",
	}

	err = db.Save(m).Error
	require.NoError(t, err)

	var result string
	err = db.Raw("select a_secret from struct_for_testings where id = ?", id).Scan(&result).Error
	require.NoError(t, err)

	require.NotEqual(t, "don't tell", result)
	require.NotEqual(t, "", result)
	require.Len(t, result, 88) // encrypts to this many bytes

	m2 := &StructForTesting{}

	err = db.Find(m2, db.Where("id = ?", id)).Error
	require.NoError(t, err)

	require.EqualValues(t, "don't tell", m2.ASecret)
}

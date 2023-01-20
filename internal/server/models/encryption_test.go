package models_test

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/database"
	"github.com/infrahq/infra/internal/testing/patch"
	"github.com/infrahq/infra/uid"
)

type StructForTesting struct {
	ID      uid.ID
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

	db, err := data.NewDB(data.NewDBOptions{DSN: database.PostgresDriver(t, "models").DSN})
	assert.NilError(t, err)

	_, err = db.Exec(StructForTesting{}.Schema())
	assert.NilError(t, err)

	id := uid.New()

	m := &StructForTesting{
		ID:      id,
		ASecret: "don't tell",
	}

	_, err = db.Exec(`INSERT into struct_for_testings VALUES(?, ?)`, m.ID, m.ASecret)
	assert.NilError(t, err)

	var result string
	err = db.QueryRow("select a_secret from struct_for_testings where id = ?", id).Scan(&result)
	assert.NilError(t, err)

	assert.Assert(t, "don't tell" != result)
	assert.Assert(t, "" != result)

	m2 := &StructForTesting{}
	err = db.QueryRow(`SELECT a_secret FROM struct_for_testings where id = ?`, id).Scan(&m2.ASecret)
	assert.NilError(t, err)

	assert.Equal(t, "don't tell", string(m2.ASecret))
}

func TestEncryptedAtRest_WithBytes(t *testing.T) {
	patch.ModelsSymmetricKey(t)

	db, err := data.NewDB(data.NewDBOptions{DSN: database.PostgresDriver(t, "models").DSN})
	assert.NilError(t, err)

	settings, err := data.GetSettings(db)
	assert.NilError(t, err)

	t.Run("Scan", func(t *testing.T) {
		var newEncrypted models.EncryptedAtRest
		err := db.QueryRow(`SELECT private_jwk FROM settings WHERE id = ?`, settings.ID).Scan(&newEncrypted)
		assert.NilError(t, err)

		assert.Equal(t, string(settings.PrivateJWK), string(newEncrypted))
	})
	t.Run("Value", func(t *testing.T) {
		newEncrypted := settings.PrivateJWK

		_, err := db.Exec(`UPDATE settings SET private_jwk = ? WHERE id = ?`, newEncrypted, settings.ID)
		assert.NilError(t, err)

		updated, err := data.GetSettings(db)
		assert.NilError(t, err)

		assert.Equal(t, string(updated.PrivateJWK), string(settings.PrivateJWK))
	})
}

package data

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestInitializeSettings(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var settings *models.Settings
		runStep(t, "first call creates new settings", func(t *testing.T) {
			var err error
			settings, err = initializeSettings(db, db.DefaultOrg.ID)
			assert.NilError(t, err)

			assert.Assert(t, settings.ID != 0)
			assert.Assert(t, len(settings.PrivateJWK) != 0)
			assert.Assert(t, len(settings.PublicJWK) != 0)
		})

		runStep(t, "next call returns existing settings", func(t *testing.T) {
			nextSettings, err := initializeSettings(db, db.DefaultOrg.ID)
			assert.NilError(t, err)
			assert.DeepEqual(t, settings, nextSettings, cmpModel)
		})
	})
}

func runStep(t *testing.T, name string, fn func(t *testing.T)) {
	if !t.Run(name, fn) {
		t.FailNow()
	}
}

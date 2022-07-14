package data

import (
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"

	"github.com/infrahq/infra/internal/server/models"
)

func TestInitializeSettings(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		var settings *models.Settings
		runStep(t, "first call creates new settings", func(t *testing.T) {
			var err error
			settings, err = InitializeSettings(db)
			assert.NilError(t, err)

			assert.Assert(t, settings.ID != 0)
			assert.Assert(t, len(settings.PrivateJWK) != 0)
			assert.Assert(t, len(settings.PublicJWK) != 0)
		})

		runStep(t, "next call returns existing settings", func(t *testing.T) {
			nextSettings, err := InitializeSettings(db)
			assert.NilError(t, err)
			assert.DeepEqual(t, settings, nextSettings, cmpModel)
		})
	})
}

var cmpModel = gocmp.Options{
	gocmp.FilterPath(opt.PathField(models.Model{}, "CreatedAt"),
		opt.TimeWithThreshold(2*time.Second)),
	gocmp.FilterPath(opt.PathField(models.Model{}, "UpdatedAt"),
		opt.TimeWithThreshold(2*time.Second)),
}

func runStep(t *testing.T, name string, fn func(t *testing.T)) {
	if !t.Run(name, fn) {
		t.FailNow()
	}
}

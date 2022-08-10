package data

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
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

var cmpModel = cmp.Options{
	cmp.FilterPath(opt.PathField(models.Model{}, "ID"), anyValidUID),
	cmp.FilterPath(opt.PathField(models.Model{}, "CreatedAt"), opt.TimeWithThreshold(2*time.Second)),
	cmp.FilterPath(opt.PathField(models.Model{}, "UpdatedAt"), opt.TimeWithThreshold(2*time.Second)),
}

var anyValidUID = cmp.Comparer(func(x, y uid.ID) bool {
	return x > 0 && y > 0
})

func runStep(t *testing.T, name string, fn func(t *testing.T)) {
	if !t.Run(name, fn) {
		t.FailNow()
	}
}

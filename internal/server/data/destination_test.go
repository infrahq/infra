package data

import (
	"testing"
	"time"

	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestDestinationSaveCreatedPersists(t *testing.T) {
	driver, err := NewSQLiteDriver("file::memory:")
	assert.NilError(t, err)

	db, err := NewDB(driver, nil)
	assert.NilError(t, err)

	destination := &models.Destination{
		Name: "example-cluster-1",
	}

	err = CreateDestination(db, destination)
	assert.NilError(t, err)
	assert.Assert(t, !destination.CreatedAt.IsZero())

	destination.Name = "example-cluster-2"
	destination.CreatedAt = time.Time{}

	err = SaveDestination(db, destination)
	assert.NilError(t, err)

	destination, err = GetDestination(db, ByID(destination.ID))
	assert.NilError(t, err)
	assert.Assert(t, !destination.CreatedAt.IsZero())
}

func TestCountDestinationsByConnectedVersion(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		assert.NilError(t, CreateDestination(db, &models.Destination{Name: "1", UniqueID: "1", LastSeenAt: time.Now()}))
		assert.NilError(t, CreateDestination(db, &models.Destination{Name: "2", UniqueID: "2", Version: "", LastSeenAt: time.Now().Add(-10 * time.Minute)}))
		assert.NilError(t, CreateDestination(db, &models.Destination{Name: "3", UniqueID: "3", Version: "0.1.0", LastSeenAt: time.Now()}))
		assert.NilError(t, CreateDestination(db, &models.Destination{Name: "4", UniqueID: "4", Version: "0.1.0"}))
		assert.NilError(t, CreateDestination(db, &models.Destination{Name: "5", UniqueID: "5", Version: "0.1.0"}))

		actual, err := CountDestinationsByConnectedVersion(db)
		assert.NilError(t, err)

		expected := []destinationsCount{
			{Connected: false, Version: "", Count: 1},
			{Connected: true, Version: "", Count: 1},
			{Connected: false, Version: "0.1.0", Count: 2},
			{Connected: true, Version: "0.1.0", Count: 1},
		}

		assert.DeepEqual(t, actual, expected)
	})
}

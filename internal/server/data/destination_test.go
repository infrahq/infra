package data

import (
	"testing"
	"time"

	"github.com/infrahq/infra/internal/server/models"
	"gotest.tools/v3/assert"
)

func TestDestinationSaveCreatedPersists(t *testing.T) {
	driver, err := NewSQLiteDriver("file::memory:")
	assert.NilError(t, err)

	db, err := NewDB(driver)
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

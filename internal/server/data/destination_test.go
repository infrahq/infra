package data

import (
	"errors"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestCreateDestination(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("success", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			destination := &models.Destination{
				Name:          "kubernetes",
				UniqueID:      "unique-id",
				ConnectionURL: "10.0.0.1:1001",
				ConnectionCA:  "the-pem-encoded-cert",
				LastSeenAt:    time.Date(2022, 1, 2, 3, 4, 5, 600, time.UTC),
				Version:       "0.100.1",
				Resources:     []string{"res1", "res2"},
				Roles:         []string{"role1", "role2"},
			}

			err := CreateDestination(tx, destination)
			assert.NilError(t, err)
			assert.Assert(t, destination.ID != 0)

			expected := &models.Destination{
				Model: models.Model{
					ID:        destination.ID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				OrganizationMember: models.OrganizationMember{OrganizationID: defaultOrganizationID},
				Name:               "kubernetes",
				UniqueID:           "unique-id",
				ConnectionURL:      "10.0.0.1:1001",
				ConnectionCA:       "the-pem-encoded-cert",
				LastSeenAt:         time.Date(2022, 1, 2, 3, 4, 5, 600, time.UTC),
				Version:            "0.100.1",
				Resources:          []string{"res1", "res2"},
				Roles:              []string{"role1", "role2"},
			}
			assert.DeepEqual(t, destination, expected, cmpModel)
		})
		t.Run("conflict on uniqueID", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			destination := &models.Destination{
				Name:     "kubernetes",
				UniqueID: "unique-id",
			}
			err := CreateDestination(tx, destination)
			assert.NilError(t, err)
			assert.Assert(t, destination.ID != 0)

			next := &models.Destination{
				Name:     "other",
				UniqueID: "unique-id",
			}
			err = CreateDestination(tx, next)
			var ucErr UniqueConstraintError
			assert.Assert(t, errors.As(err, &ucErr))
			expected := UniqueConstraintError{Table: "destinations", Column: "uniqueID"}
			assert.DeepEqual(t, ucErr, expected)
		})
	})
}

func TestDestinationSaveCreatedPersists(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		destination := &models.Destination{
			Name:     "example-cluster-1",
			UniqueID: "11111",
		}

		err := CreateDestination(db, destination)
		assert.NilError(t, err)
		assert.Assert(t, !destination.CreatedAt.IsZero())

		destination.Name = "example-cluster-2"
		destination.CreatedAt = time.Time{}

		err = SaveDestination(db, destination)
		assert.NilError(t, err)

		destination, err = GetDestination(db, ByID(destination.ID))
		assert.NilError(t, err)
		assert.Assert(t, !destination.CreatedAt.IsZero())
	})
}

func TestCountDestinationsByConnectedVersion(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		createDestinations(t, db,
			&models.Destination{Name: "1", UniqueID: "1", LastSeenAt: time.Now()},
			&models.Destination{Name: "2", UniqueID: "2", Version: "", LastSeenAt: time.Now().Add(-10 * time.Minute)},
			&models.Destination{Name: "3", UniqueID: "3", Version: "0.1.0", LastSeenAt: time.Now()},
			&models.Destination{Name: "4", UniqueID: "4", Version: "0.1.0"},
			&models.Destination{Name: "5", UniqueID: "5", Version: "0.1.0"},
		)
		actual, err := CountDestinationsByConnectedVersion(db)
		assert.NilError(t, err)

		expected := []DestinationsCount{
			{Connected: false, Version: "", Count: 1},
			{Connected: false, Version: "0.1.0", Count: 2},
			{Connected: true, Version: "", Count: 1},
			{Connected: true, Version: "0.1.0", Count: 1},
		}
		assert.DeepEqual(t, actual, expected)
	})
}

func createDestinations(t *testing.T, tx GormTxn, destinations ...*models.Destination) {
	t.Helper()
	for i := range destinations {
		err := CreateDestination(tx, destinations[i])
		assert.NilError(t, err, destinations[i].Name)
	}
}

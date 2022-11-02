package data

import (
	"errors"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
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

func TestUpdateDestination(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("success", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			created := time.Date(2022, 1, 2, 3, 4, 5, 600, time.UTC)
			orig := &models.Destination{
				Model:    models.Model{CreatedAt: created, UpdatedAt: created},
				Name:     "example-cluster-1",
				UniqueID: "11111",
			}
			createDestinations(t, tx, orig)

			// Unlike other update operations, the passed in destination
			// may be constructed entirely by the caller and may not have the
			// created, or updated time set.
			destination := &models.Destination{
				Model:         models.Model{ID: orig.ID},
				Name:          "example-cluster-2",
				UniqueID:      "22222",
				ConnectionURL: "dest.internal:10001",
				ConnectionCA:  "the-pem-encoded-cert",
				Resources:     []string{"res1", "res3"},
				Roles:         []string{"role1"},
				Version:       "0.100.2",
			}
			err := UpdateDestination(tx, destination)
			assert.NilError(t, err)

			actual, err := GetDestination(tx, GetDestinationOptions{ByID: destination.ID})
			assert.NilError(t, err)

			expected := &models.Destination{
				Model: models.Model{
					ID:        destination.ID,
					CreatedAt: created,
					UpdatedAt: time.Now(),
				},
				OrganizationMember: models.OrganizationMember{OrganizationID: defaultOrganizationID},
				Name:               "example-cluster-2",
				UniqueID:           "22222",
				ConnectionURL:      "dest.internal:10001",
				ConnectionCA:       "the-pem-encoded-cert",
				Resources:          []string{"res1", "res3"},
				Roles:              []string{"role1"},
				Version:            "0.100.2",
			}
			assert.DeepEqual(t, actual, expected, cmpModel)
		})
	})
}

func TestGetDestination(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
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
		other := &models.Destination{
			Name:     "other",
			UniqueID: "other-unique-id",
		}
		otherOrgDest := &models.Destination{
			Name:               "kubernetes",
			UniqueID:           "unique-id",
			OrganizationMember: models.OrganizationMember{OrganizationID: 200},
		}
		deleted := &models.Destination{
			Name:     "deleted",
			UniqueID: "unique-id",
		}
		deleted.DeletedAt.Time = time.Now()
		deleted.DeletedAt.Valid = true
		createDestinations(t, db, other, destination, otherOrgDest, deleted)

		t.Run("default opts", func(t *testing.T) {
			_, err := GetDestination(db, GetDestinationOptions{})
			assert.ErrorContains(t, err, "an ID is required")
		})
		t.Run("by ID", func(t *testing.T) {
			actual, err := GetDestination(db, GetDestinationOptions{ByID: destination.ID})
			assert.NilError(t, err)
			assert.DeepEqual(t, actual, destination, cmpTimeWithDBPrecision)
		})
		t.Run("by uniqueID", func(t *testing.T) {
			actual, err := GetDestination(db, GetDestinationOptions{ByUniqueID: "unique-id"})
			assert.NilError(t, err)
			assert.DeepEqual(t, actual, destination, cmpTimeWithDBPrecision)
		})
		t.Run("by name", func(t *testing.T) {
			actual, err := GetDestination(db, GetDestinationOptions{ByName: "kubernetes"})
			assert.NilError(t, err)
			assert.DeepEqual(t, actual, destination, cmpTimeWithDBPrecision)
		})
		t.Run("not found", func(t *testing.T) {
			_, err := GetDestination(db, GetDestinationOptions{ByID: 12345})
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
		t.Run("not found soft deleted", func(t *testing.T) {
			_, err := GetDestination(db, GetDestinationOptions{ByID: deleted.ID})
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
	})
}

func TestListDestinations(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
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
		other := &models.Destination{
			Name:     "other",
			UniqueID: "other-unique-id",
		}
		otherOrgDest := &models.Destination{
			Name:               "kubernetes",
			UniqueID:           "unique-id",
			OrganizationMember: models.OrganizationMember{OrganizationID: 200},
		}
		deleted := &models.Destination{
			Name:     "deleted",
			UniqueID: "unique-id",
		}
		deleted.DeletedAt.Time = time.Now()
		deleted.DeletedAt.Valid = true
		createDestinations(t, db, destination, other, otherOrgDest, deleted)

		t.Run("default options", func(t *testing.T) {
			actual, err := ListDestinations(db, ListDestinationsOptions{})
			assert.NilError(t, err)

			expected := []models.Destination{
				{Model: models.Model{ID: destination.ID}},
				{Model: models.Model{ID: other.ID}},
			}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
		})
		t.Run("by uniqueID", func(t *testing.T) {
			actual, err := ListDestinations(db, ListDestinationsOptions{
				ByUniqueID: "unique-id",
			})
			assert.NilError(t, err)

			expected := []models.Destination{
				{Model: models.Model{ID: destination.ID}},
			}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
		})
		t.Run("by name", func(t *testing.T) {
			actual, err := ListDestinations(db, ListDestinationsOptions{ByName: "kubernetes"})
			assert.NilError(t, err)

			expected := []models.Destination{
				{Model: models.Model{ID: destination.ID}},
			}
			assert.DeepEqual(t, actual, expected, cmpModelByID)

		})
		t.Run("with pagination", func(t *testing.T) {
			page := &Pagination{Page: 2, Limit: 1}
			actual, err := ListDestinations(db, ListDestinationsOptions{Pagination: page})
			assert.NilError(t, err)

			expected := []models.Destination{
				{Model: models.Model{ID: other.ID}},
			}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
			assert.Equal(t, page.TotalCount, 2)
		})
	})
}

func TestDeleteDestination(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)

		dest := &models.Destination{Name: "kube", UniqueID: "1111"}
		createDestinations(t, tx, dest)

		err := DeleteDestination(tx, dest.ID)
		assert.NilError(t, err)

		_, err = GetDestination(tx, GetDestinationOptions{ByID: dest.ID})
		assert.ErrorIs(t, err, internal.ErrNotFound)
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

func createDestinations(t *testing.T, tx WriteTxn, destinations ...*models.Destination) {
	t.Helper()
	for i := range destinations {
		err := CreateDestination(tx, destinations[i])
		assert.NilError(t, err, destinations[i].Name)
	}
}

func TestCountAllDestinations(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		createDestinations(t, db,
			&models.Destination{Name: "1", UniqueID: "1"},
			&models.Destination{Name: "2", UniqueID: "2"},
			&models.Destination{Name: "3", UniqueID: "3"},
			&models.Destination{Name: "4", UniqueID: "4"},
			&models.Destination{Name: "5", UniqueID: "5"},
		)
		actual, err := CountAllDestinations(db)
		assert.NilError(t, err)

		assert.Equal(t, actual, int64(5))
	})
}

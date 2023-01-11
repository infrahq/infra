package data

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
)

func TestCreateDestination(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("success", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			destination := &models.Destination{
				Name:          "thehost",
				Kind:          "ssh",
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
				Name:               "thehost",
				Kind:               "ssh",
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
				Kind:     "kubernetes",
			}
			err := CreateDestination(tx, destination)
			assert.NilError(t, err)
			assert.Assert(t, destination.ID != 0)

			next := &models.Destination{
				Name:     "other",
				UniqueID: "unique-id",
				Kind:     "ssh",
			}
			err = CreateDestination(tx, next)
			var ucErr UniqueConstraintError
			assert.Assert(t, errors.As(err, &ucErr))
			expected := UniqueConstraintError{Table: "destinations", Column: "uniqueID"}
			assert.DeepEqual(t, ucErr, expected)
		})
		t.Run("multiple missing uniqueID", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			destination := &models.Destination{
				Name: "kubernetes",
				Kind: "kubernetes",
			}
			err := CreateDestination(tx, destination)
			assert.NilError(t, err)

			second := &models.Destination{
				Name: "dev",
				Kind: "kubernetes",
			}
			err = CreateDestination(tx, second)
			assert.NilError(t, err)
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
				Kind:     "kubernetes",
				UniqueID: "11111",
			}
			createDestinations(t, tx, orig)

			destination := &models.Destination{
				Model:         orig.Model,
				Name:          "example-cluster-2",
				UniqueID:      "22222",
				Kind:          "kubernetes",
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
				Kind:               "kubernetes",
				ConnectionURL:      "dest.internal:10001",
				ConnectionCA:       "the-pem-encoded-cert",
				Resources:          []string{"res1", "res3"},
				Roles:              []string{"role1"},
				Version:            "0.100.2",
			}
			assert.DeepEqual(t, actual, expected, cmpModel)
		})
		t.Run("multiple missing uniqueID", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			destination := &models.Destination{
				Name: "kubernetes",
				Kind: "kubernetes",
			}
			err := CreateDestination(tx, destination)
			assert.NilError(t, err)

			second := &models.Destination{
				Name:     "dev",
				Kind:     "kubernetes",
				UniqueID: "something",
			}
			err = CreateDestination(tx, second)
			assert.NilError(t, err)

			second.UniqueID = ""
			err = UpdateDestination(tx, second)
			assert.NilError(t, err)
		})
	})
}

func TestGetDestination(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		destination := &models.Destination{
			Name:          "kubernetes",
			UniqueID:      "unique-id",
			Kind:          "kubernetes",
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
			Kind:     "kubernetes",
		}
		otherOrgDest := &models.Destination{
			Name:               "kubernetes",
			UniqueID:           "unique-id",
			Kind:               "ssh",
			OrganizationMember: models.OrganizationMember{OrganizationID: 200},
		}
		deleted := &models.Destination{
			Name:     "deleted",
			UniqueID: "unique-id",
			Kind:     "ssh",
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
		t.Run("from other organization", func(t *testing.T) {
			_, err := GetDestination(db, GetDestinationOptions{
				ByID:             destination.ID,
				FromOrganization: 723,
			})
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
	})
}

func TestListDestinations(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		destination := &models.Destination{
			Name:          "kubernetes",
			Kind:          "kubernetes",
			UniqueID:      "unique-id",
			ConnectionURL: "10.0.0.1:1001",
			ConnectionCA:  "the-pem-encoded-cert",
			LastSeenAt:    time.Date(2022, 1, 2, 3, 4, 5, 600, time.UTC),
			Version:       "0.100.1",
			Resources:     []string{"res1", "res2"},
			Roles:         []string{"role1", "role2"},
		}
		second := &models.Destination{
			Name:          "bastion",
			Kind:          "ssh",
			UniqueID:      "notused",
			ConnectionURL: "10.0.0.2:22",
			ConnectionCA:  "fingerprint",
		}
		other := &models.Destination{
			Name:     "other",
			UniqueID: "other-unique-id",
			Kind:     "kubernetes",
		}
		otherOrgDest := &models.Destination{
			Name:               "kubernetes",
			UniqueID:           "unique-id",
			OrganizationMember: models.OrganizationMember{OrganizationID: 200},
			Kind:               "ssh",
		}
		deleted := &models.Destination{
			Name:     "deleted",
			UniqueID: "unique-id",
			Kind:     "ssh",
		}
		deleted.DeletedAt.Time = time.Now()
		deleted.DeletedAt.Valid = true
		createDestinations(t, db, destination, second, other, otherOrgDest, deleted)

		var cmpDestination = cmp.Options{
			cmpTimeWithDBPrecision,
			cmpopts.EquateEmpty(),
		}

		t.Run("default options", func(t *testing.T) {
			actual, err := ListDestinations(db, ListDestinationsOptions{})
			assert.NilError(t, err)

			expected := []models.Destination{*second, *destination, *other}
			assert.DeepEqual(t, actual, expected, cmpDestination)
		})
		t.Run("by uniqueID", func(t *testing.T) {
			actual, err := ListDestinations(db, ListDestinationsOptions{
				ByUniqueID: "unique-id",
			})
			assert.NilError(t, err)

			expected := []models.Destination{*destination}
			assert.DeepEqual(t, actual, expected, cmpDestination)
		})
		t.Run("by name", func(t *testing.T) {
			actual, err := ListDestinations(db, ListDestinationsOptions{ByName: "kubernetes"})
			assert.NilError(t, err)

			expected := []models.Destination{*destination}
			assert.DeepEqual(t, actual, expected, cmpDestination)
		})
		t.Run("with pagination", func(t *testing.T) {
			page := &Pagination{Page: 2, Limit: 2}
			actual, err := ListDestinations(db, ListDestinationsOptions{Pagination: page})
			assert.NilError(t, err)

			expected := []models.Destination{*other}
			assert.DeepEqual(t, actual, expected, cmpDestination)
			assert.Equal(t, page.TotalCount, 3)
		})
		t.Run("by kind", func(t *testing.T) {
			actual, err := ListDestinations(db, ListDestinationsOptions{ByKind: "kubernetes"})
			assert.NilError(t, err)

			expected := []models.Destination{*destination, *other}
			assert.DeepEqual(t, actual, expected, cmpDestination)

		})
	})
}

func TestDeleteDestination(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)

		dest := &models.Destination{Name: "kube", UniqueID: "1111", Kind: "kubernetes"}
		createDestinations(t, tx, dest)

		pileOGrants := []*models.Grant{
			{
				Subject:   "i:1234567",
				Privilege: "view",
				Resource:  "kube",
			},
			{
				Subject:   "i:1234567",
				Privilege: "view",
				Resource:  "kube.namespace",
			},
			{
				Subject:   "i:1234567",
				Privilege: "view",
				Resource:  "somethingelse",
			},
		}

		for _, g := range pileOGrants {
			err := CreateGrant(tx, g)
			assert.NilError(t, err)
		}

		actual, err := ListGrants(tx, ListGrantsOptions{BySubject: "i:1234567"})
		assert.NilError(t, err)
		assert.Equal(t, len(actual), 3)

		err = DeleteDestination(tx, dest.ID)
		assert.NilError(t, err)

		_, err = GetDestination(tx, GetDestinationOptions{ByID: dest.ID})
		assert.ErrorIs(t, err, internal.ErrNotFound)

		actual, err = ListGrants(tx, ListGrantsOptions{BySubject: "i:1234567"})
		assert.NilError(t, err)
		assert.Equal(t, len(actual), 1)
		assert.Equal(t, actual[0].Resource, "somethingelse")
	})
}

func TestCountDestinationsByConnectedVersion(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		createDestinations(t, db,
			&models.Destination{Name: "1", UniqueID: "1", Kind: "ssh", LastSeenAt: time.Now()},
			&models.Destination{Name: "2", UniqueID: "2", Kind: "ssh", Version: "", LastSeenAt: time.Now().Add(-10 * time.Minute)},
			&models.Destination{Name: "3", UniqueID: "3", Kind: "ssh", Version: "0.1.0", LastSeenAt: time.Now()},
			&models.Destination{Name: "4", UniqueID: "4", Kind: "ssh", Version: "0.1.0"},
			&models.Destination{Name: "5", UniqueID: "5", Kind: "ssh", Version: "0.1.0"},
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
			&models.Destination{Name: "1", UniqueID: "1", Kind: "ssh"},
			&models.Destination{Name: "2", UniqueID: "2", Kind: "ssh"},
			&models.Destination{Name: "3", UniqueID: "3", Kind: "ssh"},
			&models.Destination{Name: "4", UniqueID: "4", Kind: "ssh"},
			&models.Destination{Name: "5", UniqueID: "5", Kind: "ssh"},
		)
		actual, err := CountAllDestinations(db)
		assert.NilError(t, err)

		assert.Equal(t, actual, int64(5))
	})
}

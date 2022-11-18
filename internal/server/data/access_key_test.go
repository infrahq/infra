package data

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestCreateAccessKey(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		org := &models.Organization{Name: "something", Domain: "example.com"}
		assert.NilError(t, CreateOrganization(db, org))

		tx := txnForTestCase(t, db, org.ID)

		jerry := &models.Identity{Name: "jseinfeld@infrahq.com"}
		err := CreateIdentity(tx, jerry)
		assert.NilError(t, err)

		infraProviderID := InfraProvider(tx).ID

		t.Run("all default values", func(t *testing.T) {
			key := &models.AccessKey{
				IssuedFor:  jerry.ID,
				ProviderID: infraProviderID,
			}
			pair, err := CreateAccessKey(tx, key)
			assert.NilError(t, err)

			expected := &models.AccessKey{
				OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
				Model: models.Model{
					ID:        uid.ID(12345),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				IssuedFor:         jerry.ID,
				ProviderID:        infraProviderID,
				KeyID:             "<any-string>",
				Secret:            "<any-string>",
				ExpiresAt:         time.Now().Add(12 * time.Hour),
				ExtensionDeadline: time.Now().Add(12 * time.Hour),
				Name:              fmt.Sprintf("%s-%s", jerry.Name, key.ID.String()),
				SecretChecksum:    secretChecksum(key.Secret),
			}
			assert.DeepEqual(t, key, expected, cmpAccessKey)
			assert.Equal(t, pair, key.Token())

			// check that we can fetch the same value from the db
			fromDB, err := GetAccessKeyByKeyID(tx, key.KeyID)
			assert.NilError(t, err)

			// fromDB should not have the secret value
			key.Secret = ""
			assert.DeepEqual(t, fromDB, key, cmpopts.EquateEmpty(), cmpTimeWithDBPrecision)
		})

		t.Run("all values", func(t *testing.T) {
			key := &models.AccessKey{
				Model: models.Model{
					ID:        uid.ID(512512),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name:              "the-key",
				IssuedFor:         jerry.ID,
				ProviderID:        infraProviderID,
				ExpiresAt:         time.Now().Add(time.Hour),
				Extension:         3 * time.Hour,
				ExtensionDeadline: time.Now().Add(time.Minute),
				KeyID:             "0123456789",
				Secret:            "012345678901234567890123",
				Scopes:            []string{"first", "third"},
			}
			pair, err := CreateAccessKey(tx, key)
			assert.NilError(t, err)
			assert.Equal(t, pair, key.Token())

			// check that we can fetch the same value from the db
			fromDB, err := GetAccessKeyByKeyID(tx, key.KeyID)
			assert.NilError(t, err)
			// fromDB should not have the secret value
			key.Secret = ""
			assert.DeepEqual(t, fromDB, key, cmpTimeWithDBPrecision)
		})

		t.Run("invalid specified key id length", func(t *testing.T) {
			key := &models.AccessKey{
				KeyID:      "too-short",
				IssuedFor:  jerry.ID,
				ProviderID: InfraProvider(tx).ID,
			}
			_, err := CreateAccessKey(tx, key)
			assert.Error(t, err, "invalid key length")
		})

		t.Run("invalid specified key secret length", func(t *testing.T) {
			key := &models.AccessKey{
				Secret:     "too-short",
				IssuedFor:  jerry.ID,
				ProviderID: InfraProvider(tx).ID,
			}
			_, err := CreateAccessKey(tx, key)
			assert.Error(t, err, "invalid secret length")
		})

		t.Run("access key for provider IssuedFor matches ProviderID", func(t *testing.T) {
			provider := &models.Provider{
				Name: "okta",
				Kind: models.ProviderKindOkta,
			}
			err := CreateProvider(tx, provider)
			assert.NilError(t, err)

			key := &models.AccessKey{
				Name:      "okta-scim",
				Secret:    "012345678901234567890123",
				IssuedFor: provider.ID,
			}
			_, err = CreateAccessKey(tx, key)
			assert.NilError(t, err)
			assert.Equal(t, key.ProviderID, key.IssuedFor)
		})
	})
}

var cmpModel = cmp.Options{
	cmp.FilterPath(opt.PathField(models.Model{}, "ID"), anyValidUID),
	cmp.FilterPath(opt.PathField(models.Model{}, "CreatedAt"), opt.TimeWithThreshold(2*time.Second)),
	cmp.FilterPath(opt.PathField(models.Model{}, "UpdatedAt"), opt.TimeWithThreshold(2*time.Second)),
}

var cmpAccessKey = cmp.Options{
	cmpModel,
	cmp.FilterPath(opt.PathField(models.AccessKey{}, "KeyID"), nonZeroString),
	cmp.FilterPath(opt.PathField(models.AccessKey{}, "Secret"), nonZeroString),
	cmp.FilterPath(opt.PathField(models.AccessKey{}, "ExpiresAt"), opt.TimeWithThreshold(time.Second)),
	cmp.FilterPath(opt.PathField(models.AccessKey{}, "ExtensionDeadline"), opt.TimeWithThreshold(time.Second)),
}

var nonZeroString = cmp.Comparer(func(x, y string) bool {
	if x == "" || y == "" {
		return false
	}
	if x == "<any-string>" || y == "<any-string>" {
		return true
	}
	return false
})

var anyValidUID = cmp.Comparer(func(x, y uid.ID) bool {
	return x > 0 && y > 0
})

// PostgreSQL only has microsecond precision
var cmpTimeWithDBPrecision = cmpopts.EquateApproxTime(time.Microsecond)

func createAccessKeyWithExtensionDeadline(t *testing.T, db WriteTxn, ttl, extensionDeadline time.Duration) (string, *models.AccessKey) {
	identity := &models.Identity{Name: "Wall-E"}
	err := CreateIdentity(db, identity)
	assert.NilError(t, err)

	token := &models.AccessKey{
		IssuedFor:         identity.ID,
		ProviderID:        InfraProvider(db).ID,
		ExpiresAt:         time.Now().Add(ttl),
		ExtensionDeadline: time.Now().Add(extensionDeadline).UTC(),
	}

	body, err := CreateAccessKey(db, token)
	assert.NilError(t, err)

	return body, token
}

func TestValidateRequestAccessKey(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)
		body, _ := createTestAccessKey(t, tx, time.Hour*5)

		_, err := ValidateRequestAccessKey(tx, body)
		assert.NilError(t, err)

		random := generate.MathRandom(models.AccessKeySecretLength, generate.CharsetAlphaNumeric)
		authorization := fmt.Sprintf("%s.%s", strings.Split(body, ".")[0], random)

		_, err = ValidateRequestAccessKey(tx, authorization)
		assert.Error(t, err, "access key invalid secret")
	})
}

func TestDeleteAccessKeys(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		provider := &models.Provider{Name: "azure", Kind: models.ProviderKindAzure}
		otherProvider := &models.Provider{Name: "other", Kind: models.ProviderKindGoogle}
		createProviders(t, db, provider, otherProvider)

		user := &models.Identity{Name: "main@example.com"}
		otherUser := &models.Identity{Name: "other@example.com"}
		createIdentities(t, db, user, otherUser)

		t.Run("empty options", func(t *testing.T) {
			err := DeleteAccessKeys(db, DeleteAccessKeysOptions{})
			assert.ErrorContains(t, err, "requires an ID to delete")
		})

		t.Run("by user id", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)
			key1 := &models.AccessKey{IssuedFor: user.ID, ProviderID: provider.ID}
			key2 := &models.AccessKey{IssuedFor: user.ID, ProviderID: otherProvider.ID}
			toKeep := &models.AccessKey{IssuedFor: otherUser.ID, ProviderID: otherProvider.ID}
			createAccessKeys(t, tx, key1, key2, toKeep)

			err := DeleteAccessKeys(tx, DeleteAccessKeysOptions{ByIssuedForID: user.ID})
			assert.NilError(t, err)

			remaining, err := ListAccessKeys(tx, ListAccessKeyOptions{})
			assert.NilError(t, err)
			expected := []models.AccessKey{
				{Model: models.Model{ID: toKeep.ID}},
			}
			assert.DeepEqual(t, remaining, expected, cmpModelByID)
		})

		t.Run("by provider id", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)
			key1 := &models.AccessKey{IssuedFor: user.ID, ProviderID: provider.ID}
			key2 := &models.AccessKey{IssuedFor: otherUser.ID, ProviderID: provider.ID}
			toKeep := &models.AccessKey{IssuedFor: user.ID, ProviderID: otherProvider.ID}
			createAccessKeys(t, tx, key1, key2, toKeep)

			err := DeleteAccessKeys(tx, DeleteAccessKeysOptions{ByProviderID: provider.ID})
			assert.NilError(t, err)

			remaining, err := ListAccessKeys(tx, ListAccessKeyOptions{})
			assert.NilError(t, err)
			expected := []models.AccessKey{
				{Model: models.Model{ID: toKeep.ID}},
			}
			assert.DeepEqual(t, remaining, expected, cmpModelByID)
		})

		t.Run("by id", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)
			key1 := &models.AccessKey{IssuedFor: otherUser.ID, ProviderID: provider.ID}
			toKeep := &models.AccessKey{IssuedFor: user.ID, ProviderID: otherProvider.ID}
			createAccessKeys(t, tx, key1, toKeep)

			err := DeleteAccessKeys(tx, DeleteAccessKeysOptions{ByID: key1.ID})
			assert.NilError(t, err)

			remaining, err := ListAccessKeys(tx, ListAccessKeyOptions{})
			assert.NilError(t, err)
			expected := []models.AccessKey{
				{Model: models.Model{ID: toKeep.ID}},
			}
			assert.DeepEqual(t, remaining, expected, cmpModelByID)
		})
	})
}

func createAccessKeys(t *testing.T, db WriteTxn, keys ...*models.AccessKey) {
	t.Helper()
	for i := range keys {
		_, err := CreateAccessKey(db, keys[i])
		assert.NilError(t, err)
	}
}

type primaryKeyable interface {
	Primary() uid.ID
}

var cmpModelByID = cmp.Comparer(func(x, y primaryKeyable) bool {
	return x.Primary() == y.Primary()
})

func TestCheckAccessKeyExpired(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)
		body, _ := createTestAccessKey(t, tx, -1*time.Hour)

		_, err := ValidateRequestAccessKey(tx, body)
		assert.ErrorIs(t, err, ErrAccessKeyExpired)
	})
}

func TestCheckAccessKeyPastExtensionDeadline(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)
		body, _ := createAccessKeyWithExtensionDeadline(t, tx, 1*time.Hour, -1*time.Hour)

		_, err := ValidateRequestAccessKey(tx, body)
		assert.ErrorIs(t, err, ErrAccessKeyDeadlineExceeded)
	})
}

func TestListAccessKeys(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		user := &models.Identity{Name: "tmp@infrahq.com"}
		otherUser := &models.Identity{Name: "admin@infrahq.com"}
		createIdentities(t, db, user, otherUser)

		first := &models.AccessKey{
			Name:       "alpha",
			Model:      models.Model{ID: 5},
			IssuedFor:  user.ID,
			ProviderID: InfraProvider(db).ID,
			ExpiresAt:  time.Now().Add(time.Hour).UTC(),
			KeyID:      "1234567890",
		}
		second := &models.AccessKey{
			Name:       "beta",
			Model:      models.Model{ID: 6},
			IssuedFor:  user.ID,
			ProviderID: InfraProvider(db).ID,
			ExpiresAt:  time.Now().Add(-time.Hour).UTC(),
			KeyID:      "1234567891",
		}
		third := &models.AccessKey{
			Name:              "charlie",
			Model:             models.Model{ID: 7},
			IssuedFor:         user.ID,
			ProviderID:        InfraProvider(db).ID,
			ExpiresAt:         time.Now().Add(time.Hour).UTC(),
			ExtensionDeadline: time.Now().Add(-time.Hour).UTC(),
			KeyID:             "1234567892",
		}
		deleted := &models.AccessKey{
			Name:              "delta",
			Model:             models.Model{ID: 8},
			IssuedFor:         user.ID,
			ProviderID:        InfraProvider(db).ID,
			ExpiresAt:         time.Now().Add(time.Hour).UTC(),
			ExtensionDeadline: time.Now().Add(time.Hour).UTC(),
			KeyID:             "1234567893",
		}
		deleted.DeletedAt.Time = time.Now()
		deleted.DeletedAt.Valid = true

		forth := &models.AccessKey{
			Name:       "delta",
			Model:      models.Model{ID: 9},
			IssuedFor:  otherUser.ID,
			ProviderID: InfraProvider(db).ID,
			ExpiresAt:  time.Now().Add(time.Hour).UTC(),
			KeyID:      "1234567894",
		}

		createAccessKeys(t, db, forth, third, second, first, deleted)

		otherOrg := &models.Organization{Name: "other", Domain: "ok.example.com"}
		assert.NilError(t, CreateOrganization(db, otherOrg))

		t.Run("setup other org", func(t *testing.T) {
			tx := txnForTestCase(t, db, otherOrg.ID)

			otherOrgUser := &models.Identity{
				Name:               "gamma@other.org",
				OrganizationMember: models.OrganizationMember{OrganizationID: otherOrg.ID},
			}
			assert.NilError(t, CreateIdentity(tx, otherOrgUser))

			otherOrgKey := &models.AccessKey{
				Model:              models.Model{ID: 17},
				OrganizationMember: models.OrganizationMember{OrganizationID: otherOrg.ID},
				Name:               "epsilon",
				IssuedFor:          otherOrgUser.ID,
				ProviderID:         InfraProvider(tx).ID,
				ExpiresAt:          time.Now().Add(time.Hour).UTC(),
				KeyID:              "2234567800",
			}
			createAccessKeys(t, tx, otherOrgKey)
			assert.NilError(t, tx.Commit())
		})

		cmpAccessKeyShallow := cmp.Comparer(func(x, y models.AccessKey) bool {
			return x.ID == y.ID && x.IssuedForName == y.IssuedForName
		})

		t.Run("default other org ID", func(t *testing.T) {
			tx := txnForTestCase(t, db, otherOrg.ID)

			actual, err := ListAccessKeys(tx, ListAccessKeyOptions{})
			assert.NilError(t, err)

			expected := []models.AccessKey{
				{Model: models.Model{ID: 17}, IssuedForName: "gamma@other.org"},
			}
			assert.DeepEqual(t, actual, expected, cmpAccessKeyShallow)
		})

		t.Run("default", func(t *testing.T) {
			actual, err := ListAccessKeys(db, ListAccessKeyOptions{})
			assert.NilError(t, err)

			expected := []models.AccessKey{
				{Model: models.Model{ID: 5}, IssuedForName: "tmp@infrahq.com"},
				{Model: models.Model{ID: 9}, IssuedForName: "admin@infrahq.com"},
			}
			assert.DeepEqual(t, actual, expected, cmpAccessKeyShallow)
		})

		t.Run("include expired", func(t *testing.T) {
			actual, err := ListAccessKeys(db, ListAccessKeyOptions{IncludeExpired: true})
			assert.NilError(t, err)

			expected := []models.AccessKey{
				{Model: models.Model{ID: 5}, IssuedForName: "tmp@infrahq.com"},
				{Model: models.Model{ID: 6}, IssuedForName: "tmp@infrahq.com"},
				{Model: models.Model{ID: 7}, IssuedForName: "tmp@infrahq.com"},
				{Model: models.Model{ID: 9}, IssuedForName: "admin@infrahq.com"},
			}
			assert.DeepEqual(t, actual, expected, cmpAccessKeyShallow)
		})

		t.Run("by name", func(t *testing.T) {
			actual, err := ListAccessKeys(db, ListAccessKeyOptions{ByName: "alpha"})
			assert.NilError(t, err)

			expected := []models.AccessKey{
				{Model: models.Model{ID: 5}, IssuedForName: "tmp@infrahq.com"},
			}
			assert.DeepEqual(t, actual, expected, cmpAccessKeyShallow)
		})

		t.Run("by issued for user", func(t *testing.T) {
			actual, err := ListAccessKeys(db, ListAccessKeyOptions{ByIssuedForID: user.ID})
			assert.NilError(t, err)

			expected := []models.AccessKey{
				{Model: models.Model{ID: 5}, IssuedForName: "tmp@infrahq.com"},
			}
			assert.DeepEqual(t, actual, expected, cmpAccessKeyShallow)
		})

		t.Run("by name and expired", func(t *testing.T) {
			actual, err := ListAccessKeys(db, ListAccessKeyOptions{
				ByName:         "beta",
				IncludeExpired: true,
			})
			assert.NilError(t, err)

			expected := []models.AccessKey{
				{Model: models.Model{ID: 6}, IssuedForName: "tmp@infrahq.com"},
			}
			assert.DeepEqual(t, actual, expected, cmpAccessKeyShallow)
		})

		t.Run("by issued for an expired", func(t *testing.T) {
			actual, err := ListAccessKeys(db, ListAccessKeyOptions{
				ByIssuedForID:  user.ID,
				IncludeExpired: true,
			})
			assert.NilError(t, err)

			expected := []models.AccessKey{
				{Model: models.Model{ID: 5}, IssuedForName: "tmp@infrahq.com"},
				{Model: models.Model{ID: 6}, IssuedForName: "tmp@infrahq.com"},
				{Model: models.Model{ID: 7}, IssuedForName: "tmp@infrahq.com"},
			}
			assert.DeepEqual(t, actual, expected, cmpAccessKeyShallow)
		})

		t.Run("include expired with pagination", func(t *testing.T) {
			page := &Pagination{Page: 2, Limit: 2}
			actual, err := ListAccessKeys(db, ListAccessKeyOptions{
				IncludeExpired: true,
				Pagination:     page,
			})
			assert.NilError(t, err)

			expected := []models.AccessKey{
				{Model: models.Model{ID: 7}, IssuedForName: "tmp@infrahq.com"},
				{Model: models.Model{ID: 9}, IssuedForName: "admin@infrahq.com"},
			}
			assert.DeepEqual(t, actual, expected, cmpAccessKeyShallow)
			expectedPage := &Pagination{
				Page:       2,
				Limit:      2,
				TotalCount: 4,
			}
			assert.DeepEqual(t, page, expectedPage)
		})

		t.Run("by issued for with pagination", func(t *testing.T) {
			page := &Pagination{Page: 1, Limit: 2}
			actual, err := ListAccessKeys(db, ListAccessKeyOptions{
				ByIssuedForID: user.ID,
				Pagination:    page,
			})
			assert.NilError(t, err)

			expected := []models.AccessKey{
				{Model: models.Model{ID: 5}, IssuedForName: "tmp@infrahq.com"},
			}
			assert.DeepEqual(t, actual, expected, cmpAccessKeyShallow)
			expectedPage := &Pagination{
				Page:       1,
				Limit:      2,
				TotalCount: 1,
			}
			assert.DeepEqual(t, page, expectedPage)
		})
	})
}

func createTestAccessKey(t *testing.T, db WriteTxn, sessionDuration time.Duration) (string, *models.AccessKey) {
	user := &models.Identity{Name: "tmp@infrahq.com"}
	err := CreateIdentity(db, user)
	assert.NilError(t, err)

	token := &models.AccessKey{
		IssuedFor:  user.ID,
		ProviderID: InfraProvider(db).ID,
		ExpiresAt:  time.Now().Add(sessionDuration),
	}

	body, err := CreateAccessKey(db, token)
	assert.NilError(t, err)

	return body, token
}

func TestUpdateAccessKey(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)

		provider := InfraProvider(tx)

		key := func() *models.AccessKey {
			created := time.Date(2022, 1, 2, 3, 4, 5, 0, time.UTC)
			return &models.AccessKey{
				Model: models.Model{
					ID:        123456,
					CreatedAt: created,
					UpdatedAt: created,
				},
				Name:              "this-is-my-key",
				IssuedFor:         212121,
				ProviderID:        provider.ID,
				ExpiresAt:         time.Date(2022, 2, 1, 2, 3, 4, 0, time.UTC),
				ExtensionDeadline: time.Date(2022, 2, 1, 4, 3, 4, 0, time.UTC),
				Extension:         2 * time.Hour,
				KeyID:             "the-key-id",
				Secret:            "the-key-secret-is-thatok",
				Scopes:            models.CommaSeparatedStrings{"one", "two"},
			}
		}

		orig := key()
		_, err := CreateAccessKey(tx, orig)
		assert.NilError(t, err)

		newSecret := "the-key-secret-is-123456"
		orig.Secret = newSecret
		orig.Scopes = nil

		err = UpdateAccessKey(tx, orig)
		assert.NilError(t, err)

		actual, err := GetAccessKeyByKeyID(tx, orig.KeyID)
		assert.NilError(t, err)

		expected := key()
		expected.UpdatedAt = time.Now()
		expected.Secret = ""
		expected.SecretChecksum = secretChecksum(newSecret)
		expected.Scopes = nil
		expected.OrganizationID = db.DefaultOrg.ID

		keyCmp := cmp.Options{
			cmp.FilterPath(opt.PathField(models.Model{}, "UpdatedAt"), opt.TimeWithThreshold(2*time.Second)),
			cmpopts.EquateEmpty(),
		}
		assert.DeepEqual(t, actual, expected, keyCmp)
	})
}

func TestGetAccessKey(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		user := &models.Identity{Name: "su@example.com"}
		createIdentities(t, db, user)

		ak := &models.AccessKey{
			Name:       "the-key",
			IssuedFor:  user.ID,
			ProviderID: 700700,
		}

		other := &models.AccessKey{
			Name:       "the-other-key",
			IssuedFor:  user.ID,
			ProviderID: 700700,
		}

		createAccessKeys(t, db, ak, other)

		t.Run("default options", func(t *testing.T) {
			_, err := GetAccessKey(db, GetAccessKeysOptions{})
			assert.ErrorContains(t, err, "either an ID, or name")
		})
		t.Run("found by name", func(t *testing.T) {
			actual, err := GetAccessKey(db, GetAccessKeysOptions{
				ByName:    "the-key",
				IssuedFor: user.ID,
			})
			assert.NilError(t, err)
			expected := *ak
			expected.Secret = ""
			expected.IssuedForName = "su@example.com"
			assert.DeepEqual(t, actual, &expected, cmpTimeWithDBPrecision, cmpopts.EquateEmpty())
		})

		t.Run("found by id", func(t *testing.T) {
			actual, err := GetAccessKey(db, GetAccessKeysOptions{ByID: ak.ID})
			assert.NilError(t, err)
			expected := *ak
			expected.Secret = ""
			expected.IssuedForName = "su@example.com"
			assert.DeepEqual(t, actual, &expected, cmpTimeWithDBPrecision, cmpopts.EquateEmpty())
		})

		t.Run("found by name and id", func(t *testing.T) {
			actual, err := GetAccessKey(db, GetAccessKeysOptions{ByName: "the-key", ByID: ak.ID})
			assert.NilError(t, err)
			expected := *ak
			expected.Secret = ""
			expected.IssuedForName = "su@example.com"
			assert.DeepEqual(t, actual, &expected, cmpTimeWithDBPrecision, cmpopts.EquateEmpty())
		})

		t.Run("not found", func(t *testing.T) {
			_, err := GetAccessKey(db, GetAccessKeysOptions{
				ByName:    "not-this-key-name",
				IssuedFor: 600600,
			})
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})

		t.Run("not found soft deleted", func(t *testing.T) {
			err := DeleteAccessKeys(db, DeleteAccessKeysOptions{ByID: ak.ID})
			assert.NilError(t, err)

			_, err = GetAccessKey(db, GetAccessKeysOptions{ByID: ak.ID})
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
	})
}

func TestGetAccessKeyByID(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		ak := &models.AccessKey{
			Name:       "the-key",
			IssuedFor:  600600,
			ProviderID: 700700,
		}

		other := &models.AccessKey{
			Name:       "the-other-key",
			IssuedFor:  600600,
			ProviderID: 700700,
		}

		createAccessKeys(t, db, ak, other)

		t.Run("found", func(t *testing.T) {
			actual, err := GetAccessKeyByKeyID(db, ak.KeyID)
			assert.NilError(t, err)
			expected := *ak
			expected.Secret = ""
			assert.DeepEqual(t, actual, &expected, cmpTimeWithDBPrecision, cmpopts.EquateEmpty())
		})

		t.Run("not found", func(t *testing.T) {
			_, err := GetAccessKeyByKeyID(db, "not-this-key-id")
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})

		t.Run("not found soft deleted", func(t *testing.T) {
			err := DeleteAccessKeys(db, DeleteAccessKeysOptions{ByID: ak.ID})
			assert.NilError(t, err)

			_, err = GetAccessKeyByKeyID(db, ak.KeyID)
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
	})
}

func TestRemoveExpiredAccessKeys(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)
		user := &models.Identity{Name: "user@example.com"}
		createIdentities(t, tx, user)

		ak := &models.AccessKey{
			Name:      "foo expiry",
			IssuedFor: user.ID,
			ExpiresAt: time.Now().Add(-2 * time.Hour),
		}
		_, err := CreateAccessKey(tx, ak)
		assert.NilError(t, err)

		ak2 := &models.AccessKey{
			Name:      "foo expiry2",
			IssuedFor: user.ID,
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		_, err = CreateAccessKey(tx, ak2)
		assert.NilError(t, err)

		err = RemoveExpiredAccessKeys(tx)
		assert.NilError(t, err)

		_, err = GetAccessKey(tx, GetAccessKeysOptions{ByID: ak.ID})
		assert.ErrorContains(t, err, "not found")

		_, err = GetAccessKey(tx, GetAccessKeysOptions{ByID: ak2.ID})
		assert.NilError(t, err)
	})

}

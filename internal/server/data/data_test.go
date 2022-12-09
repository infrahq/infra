package data

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/database"
	"github.com/infrahq/infra/internal/testing/patch"
	"github.com/infrahq/infra/uid"
)

func setupDB(t *testing.T) *DB {
	t.Helper()
	patch.ModelsSymmetricKey(t)

	db, err := NewDB(NewDBOptions{DSN: database.PostgresDriver(t, "_data").DSN})
	assert.NilError(t, err)

	logging.PatchLogger(t, zerolog.NewTestWriter(t))

	return db
}

func txnForTestCase(t *testing.T, db *DB, orgID uid.ID) *Transaction {
	t.Helper()
	tx, err := db.Begin(context.Background(), nil)
	assert.NilError(t, err)
	t.Cleanup(func() {
		_ = tx.Rollback()
	})
	return tx.WithOrgID(orgID)
}

// runDBTests against all supported databases.
// Set POSTGRESQL_CONNECTION to a postgresql connection string to run tests
// against postgresql.
func runDBTests(t *testing.T, run func(t *testing.T, db *DB)) {
	db := setupDB(t)
	run(t, db)
}

func TestSnowflakeIDSerialization(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		id := uid.New()
		g := &models.Group{Model: models.Model{ID: id}, Name: "Foo"}

		err := CreateGroup(db, g)
		assert.NilError(t, err)

		actual, err := GetGroup(db, GetGroupOptions{ByID: g.ID})
		assert.NilError(t, err)
		assert.Assert(t, actual.ID != 0)
		assert.Equal(t, actual.ID, g.ID)
	})
}

func TestPaginationSelector(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		alphabeticalIdentities := []string{}
		for r := 'a'; r < 'a'+26; r++ {
			alphabeticalIdentities = append(alphabeticalIdentities, string(r))
			g := &models.Identity{Name: string(r)}
			assert.NilError(t, CreateIdentity(db, g))
		}

		p := &Pagination{Page: 1, Limit: 10}
		opts := ListIdentityOptions{
			Pagination: p,
			ByNotName:  models.InternalInfraConnectorIdentityName,
		}
		actual, err := ListIdentities(context.Background(), db, opts)
		assert.NilError(t, err)
		assert.Equal(t, len(actual), 10)
		for i := 0; i < p.Limit; i++ {
			if actual[i].Name == models.InternalInfraConnectorIdentityName {
				continue
			}
			assert.Equal(t, alphabeticalIdentities[i+(p.Page-1)*p.Limit], actual[i].Name)
		}

		p.Page = 2
		actual, err = ListIdentities(context.Background(), db, opts)
		assert.NilError(t, err)
		assert.Equal(t, len(actual), 10)
		for i := 0; i < p.Limit; i++ {
			assert.Equal(t, alphabeticalIdentities[i+(p.Page-1)*p.Limit], actual[i].Name)
		}

		p.Page = 3
		actual, err = ListIdentities(context.Background(), db, opts)
		assert.NilError(t, err)
		assert.Equal(t, len(actual), 6)

		for i := 0; i < 6; i++ {
			assert.Equal(t, alphabeticalIdentities[i+(p.Page-1)*p.Limit], actual[i].Name)
		}

		p.Page, p.Limit = 1, 26
		actual, err = ListIdentities(context.Background(), db, opts)
		assert.NilError(t, err)
		for i, user := range actual {
			assert.Equal(t, user.Name, alphabeticalIdentities[i])
		}
	})
}

func TestSetOrg(t *testing.T) {
	model := &models.AccessKey{}

	tx := &Transaction{orgID: 123456}
	setOrg(tx, model)
	assert.Equal(t, model.OrganizationID, uid.ID(123456))
}

func TestNewDB(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		assert.Equal(t, db.DefaultOrg.ID, uid.ID(defaultOrganizationID))

		org, err := GetOrganization(db, GetOrganizationOptions{ByID: defaultOrganizationID})
		assert.NilError(t, err)
		assert.DeepEqual(t, org, db.DefaultOrg, cmpTimeWithDBPrecision)
	})
}

func TestDB_Begin(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("rollback", func(t *testing.T) {
			ctx := context.Background()
			tx, err := db.Begin(ctx, nil)
			assert.NilError(t, err)
			tx = tx.WithOrgID(db.DefaultOrg.ID)

			user := &models.Identity{Name: "something@example.com"}
			err = CreateIdentity(tx, user)
			assert.NilError(t, err)

			assert.NilError(t, tx.Rollback())

			// using the tx fails
			_, err = GetIdentity(tx, GetIdentityOptions{ByID: user.ID})
			assert.ErrorContains(t, err, "transaction has already been committed or rolled back")

			// using the db shows to show the rollback worked
			_, err = GetIdentity(db, GetIdentityOptions{ByID: user.ID})
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
		t.Run("commit", func(t *testing.T) {
			ctx := context.Background()
			tx, err := db.Begin(ctx, nil)
			assert.NilError(t, err)
			tx = tx.WithOrgID(db.DefaultOrg.ID)

			user := &models.Identity{Name: "something@example.com"}
			err = CreateIdentity(tx, user)
			assert.NilError(t, err)

			assert.NilError(t, tx.Commit())

			// using the tx fails
			_, err = GetIdentity(tx, GetIdentityOptions{ByID: user.ID})
			assert.ErrorContains(t, err, "transaction has already been committed or rolled back")

			// using the db shows the commit worked
			_, err = GetIdentity(db, GetIdentityOptions{ByID: user.ID})
			assert.NilError(t, err)
		})
	})
}

func TestLongRunningQueriesAreCancelled(t *testing.T) {
	if testing.Short() {
		t.Skip("too slow for short run")
	}

	runDBTests(t, func(t *testing.T, db *DB) {
		started := time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		tx, err := db.Begin(ctx, nil)
		assert.NilError(t, err)

		_, err = tx.Exec("select pg_sleep(2);")
		assert.Error(t, err, "timeout: context deadline exceeded")

		elapsed := time.Since(started)
		assert.Assert(t, elapsed < 1500*time.Millisecond, "query should have timed out and been cancelled")
	})
}

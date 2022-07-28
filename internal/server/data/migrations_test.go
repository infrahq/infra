package data

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	"k8s.io/utils/strings/slices"

	"github.com/infrahq/infra/internal/server/data/migrator"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/patch"
)

func TestMigrations(t *testing.T) {
	patch.ModelsSymmetricKey(t)
	allMigrations := migrations()

	type testCase struct {
		label    testCaseLabel
		setup    func(t *testing.T, db *gorm.DB)
		expected func(t *testing.T, db *gorm.DB)
		cleanup  func(t *testing.T, db *gorm.DB)
	}

	run := func(t *testing.T, index int, tc testCase, db *gorm.DB) {
		logging.PatchLogger(t, zerolog.NewTestWriter(t))
		if index >= len(allMigrations) {
			t.Fatalf("there are more test cases than migrations")
		}
		mgs := allMigrations[:index+1]

		if mID := mgs[len(mgs)-1].ID; mID != tc.label.Name {
			t.Error("the list of test cases is not in the same order as the list of migrations")
			t.Fatalf("test case %v was run with migration ID %v", tc.label.Name, mID)
		}

		if index == 0 {
			filename := fmt.Sprintf("testdata/migrations/%v-%v.sql", tc.label.Name, db.Dialector.Name())
			raw, err := ioutil.ReadFile(filename)
			assert.NilError(t, err)

			assert.NilError(t, db.Exec(string(raw)).Error)
		}

		if tc.setup != nil {
			tc.setup(t, db)
		}
		if tc.cleanup != nil {
			defer tc.cleanup(t, db)
		}

		opts := migrator.Options{
			InitSchema: func(db *gorm.DB) error {
				return fmt.Errorf("unexpected call to init schema")
			},
		}

		m := migrator.New(db, opts, mgs)
		err := m.Migrate()
		assert.NilError(t, err)

		// TODO: make expected required, not optional
		if tc.expected != nil {
			tc.expected(t, db)
		}
	}

	testCases := []testCase{
		{
			label: testCaseLine("202204281130"),
			expected: func(t *testing.T, tx *gorm.DB) {
				hasCol := tx.Migrator().HasColumn("settings", "signup_enabled")
				assert.Assert(t, !hasCol)
			},
		},
		{
			label: testCaseLine("202204291613"),
			expected: func(t *testing.T, db *gorm.DB) {
			},
		},
		{
			label: testCaseLine("202206081027"),
		},
		{
			label: testCaseLine("202206151027"),
			setup: func(t *testing.T, db *gorm.DB) {
				sql := `INSERT INTO providers(name) VALUES ('infra'), ('okta');`
				err := db.Exec(sql).Error
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, db *gorm.DB) {
				sql := `DELETE FROM providers`
				err := db.Exec(sql).Error
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, db *gorm.DB) {
				type provider struct {
					Name string
					Kind models.ProviderKind
				}

				query := `SELECT name, kind FROM providers where deleted_at is null`
				var actual []provider
				rows, err := db.Raw(query).Rows()
				assert.NilError(t, err)

				for rows.Next() {
					var p provider
					err := rows.Scan(&p.Name, &p.Kind)
					assert.NilError(t, err)
					actual = append(actual, p)
				}

				expected := []provider{
					{Name: "infra", Kind: models.ProviderKindInfra},
					{Name: "okta", Kind: models.ProviderKindOkta},
				}
				assert.DeepEqual(t, actual, expected)
			},
		},
		{
			label: testCaseLine("202206161733"),
			setup: func(t *testing.T, db *gorm.DB) {
				// integrity check
				assert.Assert(t, tableExists(t, db, "trusted_certificates"))
				assert.Assert(t, tableExists(t, db, "root_certificates"))
			},
			expected: func(t *testing.T, db *gorm.DB) {
				assert.Assert(t, !tableExists(t, db, "trusted_certificates"))
				assert.Assert(t, !tableExists(t, db, "root_certificates"))
			},
		},
		// TODO:
		{label: testCaseLine("202206281027")},
		{label: testCaseLine("202207041724")},
		{label: testCaseLine("202207081217")},
		{label: testCaseLine("202207270000")},
	}

	ids := make(map[string]struct{}, len(testCases))
	for _, tc := range testCases {
		ids[tc.label.Name] = struct{}{}
	}
	// all migrations should be covered by a test
	for _, m := range allMigrations {
		if _, exists := ids[m.ID]; !exists {
			t.Fatalf("migration ID %v is missing test coverage! Add a test case to this test.", m.ID)
		}
	}

	for _, driver := range dbDrivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			db, err := newRawDB(driver)
			assert.NilError(t, err)

			for i, tc := range testCases {
				runStep(t, tc.label.Name, func(t *testing.T) {
					fmt.Printf("    %v: test case %v\n", tc.label.Line, tc.label.Name)
					run(t, i, tc, db)
				})
			}

			// TODO: compare final migrated schema to static schema
		})
	}
}

// testCaseLine is motivated by this Go proposal https://github.com/golang/go/issues/52751.
// That issue has additional context about the problem this solves.
func testCaseLine(name string) testCaseLabel {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return testCaseLabel{Name: name, Line: "unknown"}
	}
	return testCaseLabel{
		Name: name,
		Line: fmt.Sprintf("%v:%v", filepath.Base(file), line),
	}
}

type testCaseLabel struct {
	Name string
	Line string
}

// loadSQL loads a sql file from disk by a file name matching the migration it's meant to test.
// To create a new file for testing a migration:
//
// 1. Start an infra server and perform operations with the CLI or API to
//    get the db in the state you want to test. You should capture the db state
//    before writing your migration. Make sure there are some relevant records
//    in the affected tables. Do not use a production server! Any sensitive data
//    should be throw away development credentials because they will be checked
//    into git.
//
// 2. Connect up to the db to dump the data. If you're running sqlite in
//    kubernetes:
//
//   kubectl exec -it deployment/infra-server -- apk add sqlite
//   kubectl exec -it deployment/infra-server -- /usr/bin/sqlite3 /var/lib/infrahq/server/sqlite3.db
//
//   at the prompt, do:
//     .dump
//   Copy to output to a file with the same name of the migration and a .sql
//   extension in the migrationdata/ folder.
//
//   Or from a local db:
//
//   echo -e ".output dump.sql\n.dump" | sqlite3 sqlite3.db
//
// 3. Write the migration and test that it does what you expect. It can be helpful
//    to put any necessary guards in place to make sure the database is in the state
//    you expect. Sometimes failed migrations leave it in a broken state, and might
//    run when you don't expect, so defensive programming is helpful here.
//
func loadSQL(t *testing.T, db *gorm.DB, filename string) {
	f, err := os.Open("migrationdata/" + filename + ".sql")
	assert.NilError(t, err)

	b, err := ioutil.ReadAll(f)
	assert.NilError(t, err)

	err = db.Exec(string(b)).Error
	assert.NilError(t, err)
}

func tableExists(t *testing.T, db *gorm.DB, name string) bool {
	t.Helper()
	var count int
	err := db.Raw("SELECT count(id) FROM " + name).Row().Scan(&count)
	if err != nil {
		t.Logf("table exists error: %v", err)
	}
	return err == nil
}

// this test does an external call to example.okta.com, if it fails check your network connection
func TestMigration_AddAuthURLAndScopesToProvider(t *testing.T) {
	for _, driver := range dbDrivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			db, err := newRawDB(driver)
			assert.NilError(t, err)

			loadSQL(t, db, "202206281027-"+driver.Name())

			db, err = NewDB(driver, nil)
			assert.NilError(t, err)

			var providers []models.Provider
			err = db.Omit("client_secret").Find(&providers).Error
			assert.NilError(t, err)

			assert.Equal(t, len(providers), 2)
			authUrls := make(map[string]string)
			scopes := make(map[string][]string)
			for _, p := range providers {
				authUrls[p.Name] = p.AuthURL
				scopes[p.Name] = p.Scopes
			}
			assert.Equal(t, authUrls["infra"], "")
			assert.Equal(t, len(scopes["infra"]), 0)
			assert.Equal(t, authUrls["okta"], "https://example.okta.com/oauth2/v1/authorize")
			assert.Assert(t, slices.Equal(scopes["okta"], []string{"openid", "email", "offline_access", "groups"}))
		})
	}
}

func TestMigration_SetDestinationLastSeenAt(t *testing.T) {
	for _, driver := range dbDrivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			db, err := newRawDB(driver)
			assert.NilError(t, err)

			loadSQL(t, db, "202207041724-"+driver.Name())

			db, err = NewDB(driver, nil)
			assert.NilError(t, err)

			var destinations []models.Destination
			err = db.Find(&destinations).Error
			assert.NilError(t, err)

			for _, destination := range destinations {
				assert.Equal(t, destination.LastSeenAt, destination.UpdatedAt)
			}
		})
	}
}

func TestMigration_RemoveDeletedProviderUsers(t *testing.T) {
	for _, driver := range dbDrivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			db, err := newRawDB(driver)
			assert.NilError(t, err)

			loadSQL(t, db, "202207270000-"+driver.Name())

			db, err = NewDB(driver, nil)
			assert.NilError(t, err)

			// there should only be one provider user from the infra provider
			// the other user has a deleted_at time and was cleared
			type providerUserDetails struct {
				Email      string
				ProviderID string
			}

			var puDetails []providerUserDetails
			err = db.Raw("SELECT email, provider_id FROM provider_users").Scan(&puDetails).Error
			assert.NilError(t, err)

			assert.Equal(t, len(puDetails), 1)
			assert.Equal(t, puDetails[0].Email, "example@infrahq.com")
			assert.Equal(t, puDetails[0].ProviderID, "75225930151567361")
		})
	}
}

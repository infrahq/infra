package data

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data/migrator"
	"github.com/infrahq/infra/internal/server/data/schema"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/database"
	"github.com/infrahq/infra/internal/testing/patch"
	"github.com/infrahq/infra/uid"
)

func TestMigrations(t *testing.T) {
	if testing.Short() {
		t.Skip("too slow for -short run")
	}
	patch.ModelsSymmetricKey(t)
	allMigrations := migrations()

	type testCase struct {
		label    testCaseLabel
		setup    func(t *testing.T, tx WriteTxn)
		expected func(t *testing.T, tx WriteTxn)
		cleanup  func(t *testing.T, tx WriteTxn)
	}

	run := func(t *testing.T, index int, tc testCase, db *DB) {
		logging.PatchLogger(t, zerolog.NewTestWriter(t))
		if index >= len(allMigrations) {
			t.Fatalf("there are more test cases than migrations")
		}
		mgs := allMigrations[:index+1]
		currentMigration := mgs[len(mgs)-1]

		if mID := currentMigration.ID; mID != tc.label.Name {
			t.Error("the list of test cases is not in the same order as the list of migrations")
			t.Fatalf("test case %v was run with migration ID %v", tc.label.Name, mID)
		}

		if index == 0 {
			filename := fmt.Sprintf("testdata/migrations/%v-postgres.sql", tc.label.Name)
			raw, err := ioutil.ReadFile(filename)
			assert.NilError(t, err)

			_, err = db.Exec(string(raw))
			assert.NilError(t, err)
		}

		if tc.setup != nil {
			tc.setup(t, db)
		}
		if tc.cleanup != nil {
			defer tc.cleanup(t, db)
		}

		opts := migrator.Options{
			InitSchema: func(db migrator.DB) error {
				return fmt.Errorf("unexpected call to init schema")
			},
		}

		tx, err := db.Begin(context.Background(), nil)
		assert.NilError(t, err)
		defer tx.Rollback()

		m := migrator.New(tx, opts, mgs)
		err = m.Migrate()
		assert.NilError(t, err)
		assert.NilError(t, tx.Commit())

		t.Run("run again to check idempotency", func(t *testing.T) {
			tx, err := db.Begin(context.Background(), nil)
			assert.NilError(t, err)
			defer tx.Rollback()

			err = currentMigration.Migrate(tx)
			assert.NilError(t, err)
			assert.NilError(t, tx.Commit())
		})

		tx, err = db.Begin(context.Background(), nil)
		assert.NilError(t, err)
		defer tx.Rollback()
		tc.expected(t, tx)
	}

	testCases := []testCase{
		{
			label: testCaseLine("202204281130"),
			expected: func(t *testing.T, tx WriteTxn) {
				// dropped columns are tested by schema comparison
			},
		},
		{
			label: testCaseLine("202204291613"),
			expected: func(t *testing.T, db WriteTxn) {
				// dropped columns are tested by schema comparison
			},
		},
		{
			label: testCaseLine("2022-06-08T10:27-fixed"),
			expected: func(t *testing.T, db WriteTxn) {
				// dropped constraints are tested by schema comparison
			},
		},
		{
			label: testCaseLine("202206151027"),
			setup: func(t *testing.T, db WriteTxn) {
				stmt := `INSERT INTO providers(name) VALUES ('infra'), ('okta');`
				_, err := db.Exec(stmt)
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, db WriteTxn) {
				stmt := `DELETE FROM providers`
				_, err := db.Exec(stmt)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, db WriteTxn) {
				type provider struct {
					Name string
					Kind models.ProviderKind
				}

				query := `SELECT name, kind FROM providers where deleted_at is null`
				rows, err := db.Query(query)
				assert.NilError(t, err)

				actual, err := scanRows(rows, func(p *provider) []any {
					return []any{&p.Name, &p.Kind}
				})
				assert.NilError(t, err)

				expected := []provider{
					{Name: "infra", Kind: models.ProviderKindInfra},
					{Name: "okta", Kind: models.ProviderKindOkta},
				}
				assert.DeepEqual(t, actual, expected)
			},
		},
		{
			label: testCaseLine("202206161733"),
			setup: func(t *testing.T, db WriteTxn) {
				// integrity check
				assert.Assert(t, migrator.HasTable(db, "trusted_certificates"))
				assert.Assert(t, migrator.HasTable(db, "root_certificates"))
			},
			expected: func(t *testing.T, db WriteTxn) {
				assert.Assert(t, !migrator.HasTable(db, "trusted_certificates"))
				assert.Assert(t, !migrator.HasTable(db, "root_certificates"))
			},
		},
		{
			label: testCaseLine("202206281027"),
			setup: func(t *testing.T, db WriteTxn) {
				t.Skip("this migration no longer works with transactions")
				stmt := `
INSERT INTO providers (id, created_at, updated_at, deleted_at, name, url, client_id, client_secret, kind, created_by) VALUES (67301777540980736, '2022-07-05 17:13:14.172568+00', '2022-07-05 17:13:14.172568+00', NULL, 'infra', '', '', 'AAAAEIRG2/PYF2erJG6cYHTybucGYWVzZ2NtBDjJTEEbL3Jvb3QvLmluZnJhL3NxbGl0ZTMuZGIua2V5DGt4MdtlZuxOUhZQTw', 'infra', 1);
INSERT INTO providers (id, created_at, updated_at, deleted_at, name, url, client_id, client_secret, kind, created_by) VALUES (67301777540980737, '2022-07-05 17:13:14.172568+00', '2022-07-05 17:13:14.172568+00', NULL, 'okta', 'example.okta.com', 'client-id', 'AAAAEIRG2/PYF2erJG6cYHTybucGYWVzZ2NtBDjJTEEbL3Jvb3QvLmluZnJhL3NxbGl0ZTMuZGIua2V5DGt4MdtlZuxOUhZQTw', 'okta', 1);
`
				_, err := db.Exec(stmt)
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, db WriteTxn) {
				_, err := db.Exec(`DELETE FROM providers;`)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, db WriteTxn) {
				rows, err := db.Query(`SELECT name, auth_url, scopes FROM providers ORDER BY name`)
				assert.NilError(t, err)

				var actual []models.Provider
				for rows.Next() {
					var p models.Provider
					var authURL sql.NullString
					err := rows.Scan(&p.Name, &authURL, &p.Scopes)
					assert.NilError(t, err)
					p.AuthURL = authURL.String
					actual = append(actual, p)
				}

				expected := []models.Provider{
					{
						Name:    "infra",
						AuthURL: "",
						Scopes:  nil,
					},
					{
						Name:    "okta",
						AuthURL: "https://example.okta.com/oauth2/v1/authorize", // set from external endpoint
						Scopes:  models.CommaSeparatedStrings{"openid", "email", "offline_access", "groups"},
					},
				}
				assert.DeepEqual(t, actual, expected)
			},
		},
		{
			label: testCaseLine("202207041724"),
			setup: func(t *testing.T, db WriteTxn) {
				stmt := `
INSERT INTO destinations (id, created_at, updated_at, name, unique_id)
VALUES (12345, '2022-07-05 00:41:49.143574', '2022-07-05 01:41:49.143574Z', 'the-destination', 'unique-id');`
				_, err := db.Exec(stmt)
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, db WriteTxn) {
				_, err := db.Exec(`DELETE FROM destinations`)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, db WriteTxn) {
				stmt := `SELECT id, name, updated_at, last_seen_at from destinations`
				rows, err := db.Query(stmt)
				assert.NilError(t, err)
				defer rows.Close()

				var actual []models.Destination
				for rows.Next() {
					var d models.Destination
					err := rows.Scan(&d.ID, &d.Name, &d.UpdatedAt, &d.LastSeenAt)
					assert.NilError(t, err)
					actual = append(actual, d)
				}

				updated := parseTime(t, "2022-07-05T01:41:49.143574Z")
				expected := []models.Destination{
					{
						Model: models.Model{
							ID:        12345,
							UpdatedAt: updated,
						},
						Name:       "the-destination",
						LastSeenAt: updated,
					},
				}
				assert.DeepEqual(t, actual, expected)
			},
		},
		{
			label: testCaseLine("202207081217"),
			setup: func(t *testing.T, db WriteTxn) {
				stmt := `
					INSERT INTO grants(id, subject, resource, privilege)
					VALUES (10100, 'i:aaa', 'infra', 'admin'),
					       (10101, 'i:aaa', 'infra', 'admin'),
					       (10102, 'i:aaa', 'other', 'admin'),
					       (10103, 'i:aaa', 'infra', 'view'),
						   (10104, 'i:aab', 'infra', 'admin');
				`
				_, err := db.Exec(stmt)
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, db WriteTxn) {
				_, err := db.Exec(`DELETE FROM grants`)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, db WriteTxn) {
				stmt := `SELECT id, subject, resource, privilege FROM grants`
				rows, err := db.Query(stmt)
				assert.NilError(t, err)
				defer rows.Close()

				var actual []models.Grant
				for rows.Next() {
					var g models.Grant
					err := rows.Scan(&g.ID, &g.Subject, &g.Resource, &g.Privilege)
					assert.NilError(t, err)
					actual = append(actual, g)
				}

				expected := []models.Grant{
					{
						Model:     models.Model{ID: 10100},
						Subject:   "i:aaa",
						Resource:  "infra",
						Privilege: "admin",
					},
					{
						Model:     models.Model{ID: 10102},
						Subject:   "i:aaa",
						Resource:  "other",
						Privilege: "admin",
					},
					{
						Model:     models.Model{ID: 10103},
						Subject:   "i:aaa",
						Resource:  "infra",
						Privilege: "view",
					},
					{
						Model:     models.Model{ID: 10104},
						Subject:   "i:aab",
						Resource:  "infra",
						Privilege: "admin",
					},
				}
				assert.DeepEqual(t, actual, expected)
			},
		},
		{
			label: testCaseLine("202207270000"),
			setup: func(t *testing.T, db WriteTxn) {
				stmt := `
INSERT INTO provider_users (identity_id, provider_id, id, created_at, updated_at, deleted_at, email, groups, last_update, redirect_url, access_token, refresh_token, expires_at) VALUES(75225930155761664,75225930151567361,75226263837810687,'2022-07-27 14:02:18.934641547+00:00','2022-07-27 14:02:19.547474589+00:00',NULL,'example@infrahq.com','','2022-07-27 14:02:19.54741888+00:00','http://localhost:8301','aaa','bbb','2022-07-27 15:02:18.420551838+00:00');
INSERT INTO provider_users (identity_id, provider_id, id, created_at, updated_at, deleted_at, email, groups, last_update, redirect_url, access_token, refresh_token, expires_at) VALUES(75225930155761664,75225930151567360,75226263837810688,'2022-07-27 14:02:18.934641547+00:00','2022-07-27 14:02:19.547474589+00:00','2022-07-27 14:00:59.448457344+00:00','example@infrahq.com','','2022-07-27 14:02:19.54741888+00:00','http://localhost:8301','aaa','bbb','2022-07-27 15:02:18.420551838+00:00');
`
				_, err := db.Exec(stmt)
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, db WriteTxn) {
				_, err := db.Exec(`DELETE FROM provider_users;`)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, db WriteTxn) {
				// there should only be one provider user from the infra provider
				// the other user has a deleted_at time and was cleared
				type providerUserDetails struct {
					Email      string
					ProviderID string
				}

				var puDetails []providerUserDetails
				rows, err := db.Query("SELECT email, provider_id FROM provider_users")
				assert.NilError(t, err)

				for rows.Next() {
					var u providerUserDetails
					assert.NilError(t, rows.Scan(&u.Email, &u.ProviderID))
					puDetails = append(puDetails, u)
				}
				assert.NilError(t, rows.Close())

				assert.Equal(t, len(puDetails), 1)
				assert.Equal(t, puDetails[0].Email, "example@infrahq.com")
				assert.Equal(t, puDetails[0].ProviderID, "75225930151567361")
			},
		},
		{
			label: testCaseLine("2022-07-28T12:46"),
			setup: func(t *testing.T, db WriteTxn) {
				stmt := `
				INSERT INTO identities (id, name, deleted_at) VALUES (100, 'deleted@test.com', '2022-07-27 14:02:18.934641547+00:00'), (101, 'user@test.com', NULL);
				INSERT INTO groups (id, name) VALUES (102, 'Test');
				INSERT INTO identities_groups (identity_id, group_id) VALUES (100, 102), (101, 102);`

				_, err := db.Exec(stmt)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, db WriteTxn) {
				type IdentityGroup struct {
					IdentityID uid.ID
					GroupID    uid.ID
				}
				var relations []IdentityGroup
				rows, err := db.Query("SELECT identity_id, group_id FROM identities_groups")
				assert.NilError(t, err)
				defer rows.Close()

				for rows.Next() {
					var relation IdentityGroup
					err := rows.Scan(&relation.IdentityID, &relation.GroupID)
					assert.NilError(t, err)
					relations = append(relations, relation)
				}

				assert.Equal(t, len(relations), 1)
				assert.DeepEqual(t, relations[0], IdentityGroup{IdentityID: 101, GroupID: 102})
			},
			cleanup: func(t *testing.T, db WriteTxn) {
				_, err := db.Exec(`DELETE FROM identities_groups;`)
				assert.NilError(t, err)
				_, err = db.Exec(`DELETE FROM identities;`)
				assert.NilError(t, err)
				_, err = db.Exec(`DELETE FROM groups;`)
				assert.NilError(t, err)
			},
		},
		{
			label: testCaseLine("2022-07-21T18:28"),
			setup: func(t *testing.T, db WriteTxn) {
				_, err := db.Exec(`INSERT INTO settings(id, created_at) VALUES(1, ?);`, time.Now())
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, db WriteTxn) {
				_, err := db.Exec(`DELETE FROM settings WHERE id=1;`)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, db WriteTxn) {
				row := db.QueryRow(`
					SELECT lowercase_min, uppercase_min, number_min, symbol_min, length_min
					FROM settings
					LIMIT 1
				`)

				var settings models.Settings
				err := row.Scan(
					&settings.LowercaseMin,
					&settings.UppercaseMin,
					&settings.NumberMin,
					&settings.SymbolMin,
					&settings.LengthMin,
				)
				assert.NilError(t, err)
				expected := models.Settings{LengthMin: 8}
				assert.DeepEqual(t, settings, expected)
			},
		},
		{
			label: testCaseLine("2022-07-27T15:54"),
			expected: func(t *testing.T, db WriteTxn) {
				// column changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine("2022-08-04T17:72"),
			expected: func(t *testing.T, db WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine("2022-08-10T13:35"),
			setup: func(t *testing.T, db WriteTxn) {
				stmt := `
INSERT INTO providers(id, name) VALUES (12345, 'okta');
`
				_, err := db.Exec(stmt)
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, db WriteTxn) {
				stmt := `DELETE FROM providers WHERE id=12345;`
				_, err := db.Exec(stmt)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, db WriteTxn) {
				stmt := `SELECT id, name, created_at, updated_at FROM organizations`
				rows, err := db.Query(stmt)
				assert.NilError(t, err)

				orgs, err := scanRows(rows, func(org *models.Organization) []any {
					return []any{&org.ID, &org.Name, &org.CreatedAt, &org.UpdatedAt}
				})
				assert.NilError(t, err)

				now := time.Now()
				expected := []models.Organization{
					{
						Model: models.Model{
							ID:        99,
							CreatedAt: now,
							UpdatedAt: now,
						},
						Name: "Default",
					},
				}
				assert.DeepEqual(t, orgs, expected, cmpModel)
				org := orgs[0]

				stmt = `SELECT id, organization_id FROM providers;`
				p := &models.Provider{}
				err = db.QueryRow(stmt).Scan(&p.ID, &p.OrganizationID)
				assert.NilError(t, err)

				expectedProvider := &models.Provider{
					Model:              models.Model{ID: 12345},
					OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
				}
				assert.DeepEqual(t, p, expectedProvider)

				stmt = `SELECT id, organization_id FROM settings;`
				s := &models.Settings{}
				err = db.QueryRow(stmt).Scan(&s.ID, &s.OrganizationID)
				assert.NilError(t, err)

				expectedSettings := &models.Settings{
					Model:              models.Model{ID: 555111},
					OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
				}
				assert.DeepEqual(t, s, expectedSettings)
			},
		},
		{
			label: testCaseLine("2022-08-11T11:52"),
			expected: func(t *testing.T, db WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine("2022-08-12T11:05"),
			expected: func(t *testing.T, tx WriteTxn) {
				// dropped indexes are tested by schema comparison
			},
		},
		{
			label: testCaseLine("2022-08-22T14:58:00Z"),
			expected: func(t *testing.T, tx WriteTxn) {
				// tested elsewhere
			},
		},
		{
			label: testCaseLine("2022-08-30T11:45"),
			setup: func(t *testing.T, db WriteTxn) {
				var originalOrgID uid.ID
				err := db.QueryRow(`SELECT id from organizations where name='Default'`).Scan(&originalOrgID)
				assert.NilError(t, err)

				stmt := ` INSERT INTO providers(id, name, organization_id) VALUES (12345, 'okta', ?)`
				_, err = db.Exec(stmt, originalOrgID)
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, db WriteTxn) {
				stmt := `DELETE FROM providers WHERE id=12345;`
				_, err := db.Exec(stmt)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, tx WriteTxn) {
				stmt := `SELECT id, name, domain FROM organizations`
				org := &models.Organization{}
				err := tx.QueryRow(stmt).Scan(&org.ID, &org.Name, (*optionalString)(&org.Domain))
				assert.NilError(t, err)

				expected := &models.Organization{
					Model:  models.Model{ID: defaultOrganizationID},
					Domain: "",
					Name:   "Default",
				}
				assert.DeepEqual(t, org, expected)

				stmt = `SELECT id, organization_id FROM providers;`
				p := &models.Provider{}
				err = tx.QueryRow(stmt).Scan(&p.ID, &p.OrganizationID)
				assert.NilError(t, err)

				expectedProvider := &models.Provider{
					Model:              models.Model{ID: 12345},
					OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
				}
				assert.DeepEqual(t, p, expectedProvider)

				stmt = `SELECT id, organization_id FROM settings;`
				s := &models.Settings{}
				err = tx.QueryRow(stmt).Scan(&s.ID, &s.OrganizationID)
				assert.NilError(t, err)

				expectedSettings := &models.Settings{
					Model:              models.Model{ID: 555111},
					OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
				}
				assert.DeepEqual(t, s, expectedSettings)
			},
		},
		{
			label: testCaseLine("2022-09-01T15:00"),
			setup: func(t *testing.T, db WriteTxn) {
				var originalOrgID uid.ID
				err := db.QueryRow(`SELECT id from organizations where name='Default'`).Scan(&originalOrgID)
				assert.NilError(t, err)

				stmt := ` INSERT INTO identities(id, name, organization_id) VALUES (12345, 'migration1@example.com', ?)`
				_, err = db.Exec(stmt, originalOrgID)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, db WriteTxn) {
				stmt := `SELECT verification_token, verified FROM identities where id = ?`
				user := &models.Identity{}
				err := db.QueryRow(stmt, 12345).Scan(&user.VerificationToken, &user.Verified)
				assert.NilError(t, err)

				assert.Assert(t, !user.Verified)
				assert.Assert(t, len(user.VerificationToken) == 10)
			},
			cleanup: func(t *testing.T, db WriteTxn) {
				stmt := `DELETE FROM identities WHERE id=12345;`
				_, err := db.Exec(stmt)
				assert.NilError(t, err)
			},
		},
		{
			label: testCaseLine("2022-09-22T11:00"),
			setup: func(t *testing.T, tx WriteTxn) {
				orgA := uid.ID(1000)
				orgB := uid.ID(2000)

				groupA := models.Group{
					Model: models.Model{
						ID: 1001,
					},
					OrganizationMember: models.OrganizationMember{
						OrganizationID: orgA,
					},
					Name: "group A",
				}

				groupB := models.Group{
					Model: models.Model{
						ID: 1002,
					},
					OrganizationMember: models.OrganizationMember{
						OrganizationID: orgB,
					},
					Name: "group B",
				}

				identityA := models.Identity{
					Model: models.Model{
						ID: 1003,
					},
					OrganizationMember: models.OrganizationMember{
						OrganizationID: orgA,
					},
					Name: "identity A",
				}

				identityB := models.Identity{
					Model: models.Model{
						ID: 1004,
					},
					OrganizationMember: models.OrganizationMember{
						OrganizationID: orgB,
					},
					Name: "identity B",
				}

				stmt := `INSERT INTO groups(id, name, organization_id) VALUES (?, ?, ?)`
				_, err := tx.Exec(stmt, groupA.ID, groupA.Name, groupA.OrganizationID)
				assert.NilError(t, err)
				_, err = tx.Exec(stmt, groupB.ID, groupB.Name, groupB.OrganizationID)
				assert.NilError(t, err)

				stmt = `INSERT INTO identities(id, name, organization_id) VALUES (?, ?, ?)`
				_, err = tx.Exec(stmt, identityA.ID, identityA.Name, identityA.OrganizationID)
				assert.NilError(t, err)
				_, err = tx.Exec(stmt, identityB.ID, identityB.Name, identityB.OrganizationID)
				assert.NilError(t, err)

				stmt = `INSERT INTO identities_groups(identity_id, group_id) VALUES (?, ?)`
				_, err = tx.Exec(stmt, identityA.ID, groupA.ID)
				assert.NilError(t, err)
				_, err = tx.Exec(stmt, identityB.ID, groupB.ID)
				assert.NilError(t, err)
				_, err = tx.Exec(stmt, identityB.ID, groupA.ID)
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, tx WriteTxn) {
				stmt := `
					DELETE FROM groups;
					DELETE FROM identities;
					DELETE from identities_groups;
				`
				_, err := tx.Exec(stmt)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, tx WriteTxn) {
				type identityGroup struct {
					IdentityID uid.ID
					GroupID    uid.ID
				}
				var results []identityGroup

				stmt := `SELECT identity_id, group_id FROM identities_groups`
				rows, err := tx.Query(stmt)
				assert.NilError(t, err)
				for rows.Next() {
					var item identityGroup
					err := rows.Scan(&item.IdentityID, &item.GroupID)
					assert.NilError(t, err)

					results = append(results, item)
				}
				assert.NilError(t, rows.Close())

				for _, item := range results {
					var identityOrgID uid.ID
					err = tx.QueryRow(`SELECT organization_id FROM identities WHERE id = ?`, item.IdentityID).Scan(&identityOrgID)
					assert.NilError(t, err)

					var groupOrgID uid.ID
					err = tx.QueryRow(`SELECT organization_id FROM groups WHERE id = ?`, item.GroupID).Scan(&groupOrgID)
					assert.NilError(t, err)

					assert.Equal(t, identityOrgID, groupOrgID)
				}
			},
		},
		{
			label: testCaseLine("2022-09-22T13:00:00"),
			setup: func(t *testing.T, db WriteTxn) {
				_, err := db.Exec("INSERT INTO provider_users(provider_id, identity_id) VALUES(1001, 1002)")
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, db WriteTxn) {
				_, err := db.Exec("DELETE FROM provider_users")
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, db WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine("2022-10-04T11:44"),
			setup: func(t *testing.T, db WriteTxn) {
				_, err := db.Exec(`
					INSERT INTO destinations(id, name) VALUES
					(10009, 'with.dot.no.more'),
					(10010, 'no-dots')`)
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, db WriteTxn) {
				_, err := db.Exec("DELETE FROM destinations")
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, db WriteTxn) {
				row := db.QueryRow("SELECT name from destinations where id=?", 10009)
				var name string
				assert.NilError(t, row.Scan(&name))
				assert.Equal(t, name, "with_dot_no_more")

				row = db.QueryRow("SELECT name from destinations where id=?", 10010)
				assert.NilError(t, row.Scan(&name))
				assert.Equal(t, name, "no-dots")
			},
		},
		{
			label: testCaseLine("2022-10-05T11:12"),
			expected: func(t *testing.T, db WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine("2022-10-05T18:00:00"),
			setup: func(t *testing.T, db WriteTxn) {
				_, err := db.Exec("INSERT INTO identities(id, name, deleted_at) VALUES(1111, 'hello@example.com', ?)", time.Now())
				assert.NilError(t, err)
				_, err = db.Exec("INSERT INTO provider_users(provider_id, identity_id, email) VALUES(9999, 1111, 'hello@example.com')")
				assert.NilError(t, err)
				_, err = db.Exec("INSERT INTO provider_users(provider_id, identity_id, email) VALUES(9999, 2222, 'hello@example.com')")
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, db WriteTxn) {
				_, err := db.Exec("DELETE FROM identities")
				assert.NilError(t, err)
				_, err = db.Exec("DELETE FROM provider_users")
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, db WriteTxn) {
				var count int
				err := db.QueryRow(`SELECT count(*) from provider_users where identity_id = 1111`).Scan(&count)
				assert.NilError(t, err)
				assert.Equal(t, count, 0)
			},
		},
		{
			label: testCaseLine("2022-09-28T13:00"),
			expected: func(t *testing.T, db WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine("2022-10-17T18:00"),
			expected: func(t *testing.T, db WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine("2022-10-17T12:40"),
			setup: func(t *testing.T, tx WriteTxn) {
				stmt := `
					INSERT into grants(id, subject, resource, organization_id)
					VALUES (1001, 'i:abcd', 'any', ?)`
				_, err := tx.Exec(stmt, defaultOrganizationID)
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, tx WriteTxn) {
				_, err := tx.Exec(`DELETE FROM grants`)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, tx WriteTxn) {
				var updateIndex int64
				err := tx.QueryRow(`SELECT update_index FROM grants`).Scan(&updateIndex)
				assert.NilError(t, err)
				assert.Equal(t, updateIndex, int64(2))
			},
		},
		{
			label: testCaseLine(addDeviceFlowAuthRequestTable().ID),
			expected: func(t *testing.T, db WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine(modifyDeviceFlowAuthRequestDropApproved().ID),
			expected: func(t *testing.T, db WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine(addExpiresAtIndices().ID),
			expected: func(t *testing.T, db WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine("2022-11-07T14:00"),
			setup: func(t *testing.T, tx WriteTxn) {
				stmt := `INSERT INTO destinations(id, name) VALUES(12345, 'some')`
				_, err := tx.Exec(stmt)
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, tx WriteTxn) {
				_, err := tx.Exec(`DELETE FROM destinations`)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, tx WriteTxn) {
				var dest destinationsTable

				err := tx.QueryRow(`SELECT id, kind FROM destinations`).Scan(&dest.ID, &dest.Kind)
				assert.NilError(t, err)
				expected := destinationsTable{
					Model: models.Model{ID: 12345},
					Kind:  models.DestinationKind("kubernetes"),
				}
				assert.DeepEqual(t, expected, dest)
			},
		},
		{
			label: testCaseLine("2022-11-03T13:00"),
			expected: func(t *testing.T, db WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine("2022-11-10T17:30"),
			expected: func(t *testing.T, tx WriteTxn) {
				var orgID uid.ID
				err := tx.QueryRow(`SELECT ID FROM organizations WHERE id = ?`, defaultOrganizationID).Scan(&orgID)
				assert.NilError(t, err)
			},
		},
		{
			label: testCaseLine("2022-11-15T10:00"),
			expected: func(t *testing.T, tx WriteTxn) {
				_, err := tx.Exec(`INSERT INTO access_keys(id, issued_for, name) VALUES(10000, 12345, 'foo')`)
				assert.NilError(t, err)
				_, err = tx.Exec(`INSERT INTO access_keys(id, issued_for, name) VALUES(10001, 12346, 'foo')`)
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, tx WriteTxn) {
				_, err := tx.Exec(`DELETE FROM access_keys WHERE id IN (10000, 10001)`)
				assert.NilError(t, err)
			},
		},
		{
			label: testCaseLine("2022-11-21T11:00"),
			expected: func(t *testing.T, db WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine(updateAccessKeysTimeoutColumn().ID),
			expected: func(t *testing.T, db WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine("2022-10-26T18:00"),
			expected: func(t *testing.T, tx WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine("2022-11-17T14:00"),
			setup: func(t *testing.T, tx WriteTxn) {
				stmt := `
					INSERT INTO identities(id, created_at, updated_at, created_by, last_seen_at, name, organization_id)
					VALUES (?, ?, ?, ?, ?, ?, 22)`
				_, err := tx.Exec(stmt, 10222, time.Now(), time.Now(), 77, time.Now(), "susu@example.com")
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, tx WriteTxn) {
				_, err := tx.Exec(`DELETE from identities`)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, tx WriteTxn) {
				txn, ok := tx.(*Transaction)
				assert.Assert(t, ok, "wrong type %T", tx)

				user, err := GetIdentity(txn.WithOrgID(22), GetIdentityOptions{ByID: 10222})
				assert.NilError(t, err)
				assert.Equal(t, user.SSHLoginName, "susu")
				assert.Equal(t, user.OrganizationID, uid.ID(22))
			},
		},
		{
			label: testCaseLine(makeIdxEmailsProvidersUnique().ID),
			expected: func(t *testing.T, db WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine(deviceFlowAuthRequestsAddUserIDProviderID().ID),
			expected: func(t *testing.T, db WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
		{
			label: testCaseLine(setGoogleSocialLoginDefaultID().ID),
			setup: func(t *testing.T, tx WriteTxn) {
				user := models.Identity{
					Model: models.Model{
						ID: 1,
					},
					OrganizationMember: models.OrganizationMember{
						OrganizationID: 2,
					},
					Name: "lucy@example.com",
				}
				stmt := `
					INSERT INTO identities(id, name, organization_id)
					VALUES (?, ?, ?)
				`
				_, err := tx.Exec(stmt, user.ID, user.Name, user.OrganizationID)
				assert.NilError(t, err)
				// add the zeroed Google social login provider user and access key
				stmt = `
					INSERT INTO provider_users(identity_id, provider_id, email)
					VALUES (?, ?, ?)
				`
				_, err = tx.Exec(stmt, user.ID, 0, user.Name)
				assert.NilError(t, err)
				key := models.AccessKey{
					Model: models.Model{
						ID: 3,
					},
					Name:           "google-access",
					IssuedFor:      user.ID,
					ProviderID:     0, // old google ID
					ExpiresAt:      time.Now().Add(1 * time.Minute),
					KeyID:          "key_id",
					SecretChecksum: []byte{},
					OrganizationMember: models.OrganizationMember{
						OrganizationID: 2,
					},
				}
				stmt = `
					INSERT INTO access_keys(id, name, issued_for, provider_id, expires_at, key_id, secret_checksum, organization_id)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				`
				_, err = tx.Exec(stmt, key.ID, key.Name, key.IssuedFor, key.ProviderID, key.ExpiresAt, key.KeyID, key.SecretChecksum, key.OrganizationID)
				assert.NilError(t, err)
			},
			cleanup: func(t *testing.T, tx WriteTxn) {
				_, err := tx.Exec(`DELETE from identities`)
				assert.NilError(t, err)
				_, err = tx.Exec(`DELETE from provider_users`)
				assert.NilError(t, err)
				_, err = tx.Exec(`DELETE from access_keys`)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, tx WriteTxn) {
				var providerUser providerUserTable
				err := tx.QueryRow(`SELECT identity_id, provider_id FROM provider_users`).Scan(&providerUser.IdentityID, &providerUser.ProviderID)
				assert.NilError(t, err)
				expectedUser := providerUserTable{
					IdentityID: 1,
					ProviderID: models.InternalGoogleProviderID,
				}
				assert.DeepEqual(t, expectedUser, providerUser)

				var accessKey accessKeyTable
				err = tx.QueryRow(`SELECT issued_for, provider_id FROM access_keys`).Scan(&accessKey.IssuedFor, &accessKey.ProviderID)
				assert.NilError(t, err)
				expectedKey := accessKeyTable{
					IssuedFor:  1,
					ProviderID: models.InternalGoogleProviderID,
				}
				assert.DeepEqual(t, expectedKey, accessKey)
			},
		},
		{
			label: testCaseLine(addDestinationCredentials().ID),
			expected: func(t *testing.T, db WriteTxn) {
				// schema changes are tested with schema comparison
			},
		},
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

	var initialSchema string
	runStep(t, "initial schema", func(t *testing.T) {
		rawDB, err := newRawDB(NewDBOptions{DSN: database.PostgresDriver(t, "").DSN})
		assert.NilError(t, err)

		db := &DB{DB: rawDB}
		opts := migrator.Options{InitSchema: initializeSchema}
		m := migrator.New(db, opts, nil)
		assert.NilError(t, m.Migrate())

		initialSchema = dumpSchema(t, os.Getenv("POSTGRESQL_CONNECTION"), "--schema-only")

		_, err = db.Exec("DROP SCHEMA IF EXISTS testing CASCADE")
		assert.NilError(t, err)
	})

	migratedDBDSN := database.PostgresDriver(t, "").DSN
	rawDB, err := newRawDB(NewDBOptions{DSN: migratedDBDSN})
	assert.NilError(t, err)
	db := &DB{DB: rawDB}
	for i, tc := range testCases {
		runStep(t, tc.label.Name, func(t *testing.T) {
			fmt.Printf("    %v: test case %v\n", tc.label.Line, tc.label.Name)
			run(t, i, tc, db)
		})
	}

	runStep(t, "compare initial schema to migrated schema", func(t *testing.T) {
		migratedSchema := dumpSchema(t, os.Getenv("POSTGRESQL_CONNECTION"), "--schema-only")

		if golden.FlagUpdate() {
			writeSchema(t, migratedSchema)
			return
		}
		if !assert.Check(t, is.Equal(initialSchema, migratedSchema)) {
			t.Log(`
The migrated schema does not match the initial schema in ./schema.sql.

If you just added a new migration, run the tests again with -update to apply the
changes to schema.sql:

    go test -run TestMigrations ./internal/server/data -update

If you changed schema.sql, add the missing migration to the migrations() function
in ./migrations.go, add a test case to this test, and run the tests again.
`)
		}
	})

	runStep(t, "setup a new DB with the migrated database", func(t *testing.T) {
		models.SkipSymmetricKey = true
		t.Cleanup(func() {
			models.SkipSymmetricKey = false
		})
		_, err := NewDB(NewDBOptions{DSN: migratedDBDSN})
		assert.NilError(t, err)
	})

	runStep(t, "check test case cleanup", func(t *testing.T) {
		// delete the default org, that we expect to exist.
		_, err := db.Exec(` DELETE FROM organizations where id = ?`, defaultOrganizationID)
		assert.NilError(t, err)
		_, err = db.Exec(`DELETE FROM settings where organization_id = ?`, defaultOrganizationID)
		assert.NilError(t, err)

		data := dumpSchema(t, os.Getenv("POSTGRESQL_CONNECTION"),
			"--section=data",
			"--exclude-table=testing.migrations",
			"--inserts",
			"--no-comments")
		stmts, err := schema.TrimComments(data)
		assert.NilError(t, err)

		if !assert.Check(t, is.Equal(stmts, "")) {
			t.Log(`
Stale data was left over from a migration test case. Make sure the cleanup
function in the test case removes all rows that are added by the setup function
and the migration.`)
		}
	})
}

func parseTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339Nano, s)
	assert.NilError(t, err)
	return v
}

func runStep(t *testing.T, name string, fn func(t *testing.T)) {
	if !t.Run(name, fn) {
		t.FailNow()
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

var isEnvironmentCI = os.Getenv("CI") != ""

func dumpSchema(t *testing.T, conn string, args ...string) string {
	t.Helper()
	if _, err := exec.LookPath("pg_dump"); err != nil {
		msg := "pg_dump is required to run this test. Install pg_dump or set $PATH to include it."
		if isEnvironmentCI {
			t.Fatalf(msg)
		}
		t.Skip(msg)
	}

	conf, err := pgx.ParseConfig(conn)
	assert.NilError(t, err, "failed to parse connection string")

	envs := os.Environ()
	addEnv := func(v string) {
		envs = append(envs, v)
	}

	if conf.Host != "" {
		addEnv("PGHOST=" + conf.Host)
	}
	if conf.Port != 0 {
		addEnv(fmt.Sprintf("PGPORT=%d", conf.Port))
	}
	if conf.User != "" {
		addEnv("PGUSER=" + conf.User)
	}
	if conf.Database != "" {
		addEnv("PGDATABASE=" + conf.Database)
	}
	if conf.Password != "" {
		addEnv("PGPASSWORD=" + conf.Password)
	}

	out := new(bytes.Buffer)
	// https://www.postgresql.org/docs/current/app-pgdump.html
	args = append(args, "--no-owner", "--no-tablespaces", "--schema=testing")
	cmd := exec.Command("pg_dump", args...)
	cmd.Env = envs
	cmd.Stdout = out
	cmd.Stderr = os.Stderr

	assert.NilError(t, cmd.Run())
	return out.String()
}

func writeSchema(t *testing.T, raw string) {
	stmts, err := schema.ParseSchema(raw)
	assert.NilError(t, err)

	var out bytes.Buffer
	out.WriteString(`-- SQL generated by TestMigrations DO NOT EDIT.
-- Instead of editing this file, add a migration to ./migrations.go and run:
--
--     go test -run TestMigrations ./internal/server/data -update
--
`)
	for _, stmt := range stmts {
		if stmt.TableName == "migrations" {
			continue
		}
		out.WriteString(stmt.Value)
	}

	t.Log("Writing new schema to schema.sql. Check 'git diff' for changes!")
	// nolint:gosec
	err = os.WriteFile("schema.sql", out.Bytes(), 0o644)
	assert.NilError(t, err)
}

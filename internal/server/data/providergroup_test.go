package data

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

var cmpModelsGroupShallow = cmp.Comparer(func(x, y models.Group) bool {
	return x.Name == y.Name && x.OrganizationID == y.OrganizationID
})

func TestCreateProviderGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		org := &models.Organization{Name: "something", Domain: "example.com"}
		assert.NilError(t, CreateOrganization(db, org))

		tx := txnForTestCase(t, db, org.ID)

		infraProviderID := InfraProvider(tx).ID

		t.Run("valid", func(t *testing.T) {
			group := &models.Group{Name: "default"}
			err := CreateGroup(tx, group)
			assert.NilError(t, err)

			pg := &models.ProviderGroup{
				ProviderID: infraProviderID,
				Name:       "default",
			}
			err = CreateProviderGroup(tx, pg)
			assert.NilError(t, err)

			// check that the provider group we fetch from the DB matches what is expected
			retrieved, err := GetProviderGroup(tx, pg.ProviderID, pg.Name)
			assert.NilError(t, err)
			assert.DeepEqual(t, retrieved, pg, cmpTimeWithDBPrecision)
		})
		t.Run("provider ID not specified fails", func(t *testing.T) {
			pg := &models.ProviderGroup{
				Name: "default",
			}
			err := CreateProviderGroup(tx, pg)
			assert.ErrorContains(t, err, "providerID is required")
		})
		t.Run("name not specified fails", func(t *testing.T) {
			pg := &models.ProviderGroup{
				ProviderID: 1234,
			}
			err := CreateProviderGroup(tx, pg)
			assert.ErrorContains(t, err, "name is required")
		})
	})
}

func TestGetProviderGroup(t *testing.T) {
	type testCase struct {
		name        string
		setup       func(t *testing.T, tx *Transaction) (providerID uid.ID, name string)
		checkResult func(t *testing.T, err error, tx *Transaction, result *models.ProviderGroup)
	}

	testCases := []testCase{
		{
			name: "get existing group",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, name string) {
				providerID = InfraProvider(tx).ID
				name = "group 1"

				testSetup := []TestProviderGroup{
					{
						Provider:  InfraProvider(tx),
						GroupName: name,
					},
				}

				setupTestProviderGroups(t, tx, testSetup)

				return providerID, name
			},
			checkResult: func(t *testing.T, err error, tx *Transaction, result *models.ProviderGroup) {
				assert.NilError(t, err)
				assert.Equal(t, result.ProviderID, InfraProvider(tx).ID)
				assert.Equal(t, result.Name, "group 1")
			},
		},
		{
			name: "get non-existent provider ID",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, name string) {
				providerID = 123
				name = "group 1"

				testSetup := []TestProviderGroup{
					{
						Provider:  InfraProvider(tx),
						GroupName: name,
					},
				}

				setupTestProviderGroups(t, tx, testSetup)

				return providerID, name
			},
			checkResult: func(t *testing.T, err error, tx *Transaction, result *models.ProviderGroup) {
				assert.ErrorContains(t, err, "record not found")
			},
		},
		{
			name: "get non-existent provider group name",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, name string) {
				providerID = 123
				name = "does not exist"

				testSetup := []TestProviderGroup{
					{
						Provider:  InfraProvider(tx),
						GroupName: "name",
					},
				}

				setupTestProviderGroups(t, tx, testSetup)

				return providerID, name
			},
			checkResult: func(t *testing.T, err error, tx *Transaction, result *models.ProviderGroup) {
				assert.ErrorContains(t, err, "record not found")
			},
		},
	}

	runDBTests(t, func(t *testing.T, db *DB) {
		org := &models.Organization{Name: "something", Domain: "example.com"}
		assert.NilError(t, CreateOrganization(db, org))

		tx := txnForTestCase(t, db, org.ID)

		for _, tc := range testCases {
			providerID, name := tc.setup(t, tx)

			result, err := GetProviderGroup(tx, providerID, name)

			tc.checkResult(t, err, tx, result)

			// clean up
			_, err = tx.Exec("DELETE FROM identities")
			assert.NilError(t, err)
			_, err = tx.Exec("DELETE FROM groups")
			assert.NilError(t, err)
			_, err = tx.Exec("DELETE FROM provider_groups_provider_users")
			assert.NilError(t, err)
			_, err = tx.Exec("DELETE FROM provider_groups")
			assert.NilError(t, err)
			_, err = tx.Exec("DELETE FROM provider_users")
			assert.NilError(t, err)
			_, err = tx.Exec("DELETE FROM providers WHERE name != 'infra'")
			assert.NilError(t, err)
		}
	})
}

func TestListProviderGroups(t *testing.T) {
	type testCase struct {
		name  string
		setup func(t *testing.T, tx *Transaction) (opts ListProviderGroupOptions, expected []models.ProviderGroup)
	}

	testCases := []testCase{
		{
			name: "list all groups",
			setup: func(t *testing.T, tx *Transaction) (opts ListProviderGroupOptions, expected []models.ProviderGroup) {
				testSetup := []TestProviderGroup{
					{
						Provider:  InfraProvider(tx),
						GroupName: "group 1",
					},
					{
						Provider:  InfraProvider(tx),
						GroupName: "group 2",
					},
				}

				opts = ListProviderGroupOptions{}
				expected = setupTestProviderGroups(t, tx, testSetup)

				return opts, expected
			},
		},
		{
			name: "list all groups for provider",
			setup: func(t *testing.T, tx *Transaction) (opts ListProviderGroupOptions, expected []models.ProviderGroup) {
				testSetup := []TestProviderGroup{
					{
						Provider:  InfraProvider(tx),
						GroupName: "group 1",
					},
					{
						Provider:  InfraProvider(tx),
						GroupName: "group 2",
					},
				}

				opts = ListProviderGroupOptions{ByProviderID: InfraProvider(tx).ID}
				expected = setupTestProviderGroups(t, tx, testSetup)

				return opts, expected
			},
		},
		{
			name: "list all groups for member ID",
			setup: func(t *testing.T, tx *Transaction) (opts ListProviderGroupOptions, expected []models.ProviderGroup) {
				user := models.Identity{
					Name: "hello@example.com",
				}
				err := CreateIdentity(tx, &user)
				assert.NilError(t, err)

				testSetup := []TestProviderGroup{
					{
						Provider:  InfraProvider(tx),
						GroupName: "group",
						Members:   []models.Identity{user},
					},
				}

				opts = ListProviderGroupOptions{ByMemberIdentityID: user.ID}
				expected = setupTestProviderGroups(t, tx, testSetup)

				return opts, expected
			},
		},
		{
			name: "list all groups for member ID and provider ID",
			setup: func(t *testing.T, tx *Transaction) (opts ListProviderGroupOptions, expected []models.ProviderGroup) {
				user := models.Identity{
					Name: "hello@example.com",
				}
				err := CreateIdentity(tx, &user)
				assert.NilError(t, err)

				okta := &models.Provider{
					Name: "okta",
					Kind: models.ProviderKindOkta,
				}
				err = CreateProvider(tx, okta)
				assert.NilError(t, err)

				testSetup := []TestProviderGroup{
					{
						Provider:  InfraProvider(tx),
						GroupName: "group",
						Members:   []models.Identity{user},
					},
					// this provider group should not be returned in the test result
					{
						Provider:  okta,
						GroupName: "group",
						Members:   []models.Identity{user},
					},
				}

				opts = ListProviderGroupOptions{ByMemberIdentityID: user.ID, ByProviderID: InfraProvider(tx).ID}

				testProviderGroups := setupTestProviderGroups(t, tx, testSetup)
				expected = []models.ProviderGroup{testProviderGroups[0]} // the infra provider group

				return opts, expected
			},
		},
	}

	runDBTests(t, func(t *testing.T, db *DB) {
		org := &models.Organization{Name: "something", Domain: "example.com"}
		assert.NilError(t, CreateOrganization(db, org))

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tx := txnForTestCase(t, db, org.ID)

				opts, expected := tc.setup(t, tx)

				result, err := ListProviderGroups(tx, opts)

				assert.NilError(t, err)
				assert.DeepEqual(t, result, expected, cmpTimeWithDBPrecision)
			})
		}
	})
}

func TestAddProviderUserToProviderGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		org := &models.Organization{Name: "something", Domain: "example.com"}
		assert.NilError(t, CreateOrganization(db, org))
		tx := txnForTestCase(t, db, org.ID)

		testSetup := []TestProviderGroup{
			{
				Provider:  InfraProvider(tx),
				GroupName: "Everyone",
			},
			{
				Provider:  InfraProvider(tx),
				GroupName: "Developers",
			},
		}
		_ = setupTestProviderGroups(t, tx, testSetup)

		spike := models.Identity{
			Name: "spike@infrahq.com",
		}

		createIdentities(t, tx, &spike)

		pu, err := CreateProviderUser(tx, InfraProvider(tx), &spike)
		assert.NilError(t, err)

		t.Run("add provider user to provider groups", func(t *testing.T) {
			stmt := `
				SELECT provider_id FROM provider_groups_provider_users
				WHERE provider_user_identity_id = ?
			`
			err = tx.QueryRow(stmt, pu.IdentityID).Scan()
			assert.ErrorContains(t, err, "no rows in result set")

			err = addMemberToProviderGroups(tx, pu, []string{"Everyone", "Developers"})
			assert.NilError(t, err)

			stmt = `
				SELECT provider_group_name FROM provider_groups_provider_users
				WHERE provider_id = ? AND provider_user_identity_id = ?
			`
			rows, err := tx.Query(stmt, pu.ProviderID, pu.IdentityID)
			assert.NilError(t, err)
			defer rows.Close()

			var result []string
			for rows.Next() {
				var name string
				err = rows.Scan(&name)
				assert.NilError(t, err)

				result = append(result, name)
			}

			assert.DeepEqual(t, result, []string{"Developers", "Everyone"})
		})
	})
}

func TestRemoveProviderUserFromProviderGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		org := &models.Organization{Name: "something", Domain: "example.com"}
		assert.NilError(t, CreateOrganization(db, org))
		tx := txnForTestCase(t, db, org.ID)

		testSetup := []TestProviderGroup{
			{
				Provider:  InfraProvider(tx),
				GroupName: "Everyone",
			},
			{
				Provider:  InfraProvider(tx),
				GroupName: "Developers",
			},
		}
		_ = setupTestProviderGroups(t, tx, testSetup)

		spike := models.Identity{
			Name: "spike@infrahq.com",
		}

		createIdentities(t, tx, &spike)

		pu, err := CreateProviderUser(tx, InfraProvider(tx), &spike)
		assert.NilError(t, err)

		t.Run("remove provider user from a provider group", func(t *testing.T) {
			err = addMemberToProviderGroups(tx, pu, []string{"Everyone", "Developers"})
			assert.NilError(t, err)

			stmt := `
				SELECT provider_group_name FROM provider_groups_provider_users
				WHERE provider_id = ? AND provider_user_identity_id = ?
			`
			rows, err := tx.Query(stmt, pu.ProviderID, pu.IdentityID)
			assert.NilError(t, err)
			defer rows.Close()

			var result []string
			for rows.Next() {
				var name string
				err = rows.Scan(&name)
				assert.NilError(t, err)

				result = append(result, name)
			}

			assert.DeepEqual(t, result, []string{"Developers", "Everyone"})

			err = removeMemberFromProviderGroups(tx, pu, []string{"Everyone"})
			assert.NilError(t, err)

			rows, err = tx.Query(stmt, pu.ProviderID, pu.IdentityID)
			assert.NilError(t, err)
			defer rows.Close()

			result = []string{}
			for rows.Next() {
				var name string
				err = rows.Scan(&name)
				assert.NilError(t, err)

				result = append(result, name)
			}

			assert.DeepEqual(t, result, []string{"Developers"})
		})
		t.Run("remove provider user from all provider groups", func(t *testing.T) {
			err = addMemberToProviderGroups(tx, pu, []string{"Everyone", "Developers"})
			assert.NilError(t, err)

			stmt := `
				SELECT provider_group_name FROM provider_groups_provider_users
				WHERE provider_id = ? AND provider_user_identity_id = ?
			`
			rows, err := tx.Query(stmt, pu.ProviderID, pu.IdentityID)
			assert.NilError(t, err)
			defer rows.Close()

			var result []string
			for rows.Next() {
				var name string
				err = rows.Scan(&name)
				assert.NilError(t, err)

				result = append(result, name)
			}

			assert.DeepEqual(t, result, []string{"Developers", "Everyone"})

			err = removeMemberFromProviderGroups(tx, pu, []string{"Everyone", "Developers"})
			assert.NilError(t, err)

			err = tx.QueryRow(stmt, pu.ProviderID, pu.IdentityID).Scan()
			assert.ErrorContains(t, err, "no rows in result set")
		})
	})
}

type TestProviderGroup struct {
	Provider  *models.Provider
	GroupName string
	Members   []models.Identity
}

func setupTestProviderGroups(t *testing.T, tx *Transaction, testProviderGroups []TestProviderGroup) []models.ProviderGroup {
	// parent groups need to be create first, in case any provider groups share the same parent
	parentGroups := make(map[string]*models.Group)
	for _, testPg := range testProviderGroups {
		if parentGroups[testPg.GroupName] == nil {
			group := &models.Group{Name: testPg.GroupName}
			err := CreateGroup(tx, group)
			assert.NilError(t, err)

			parentGroups[group.Name] = group
		}
	}

	created := []models.ProviderGroup{}
	for _, testPg := range testProviderGroups {
		pg := &models.ProviderGroup{
			ProviderID: testPg.Provider.ID,
			Name:       testPg.GroupName,
		}
		err := CreateProviderGroup(tx, pg)
		assert.NilError(t, err)

		memberIDs := []uid.ID{}
		for i := range testPg.Members {
			member, err := CreateProviderUser(tx, testPg.Provider, &testPg.Members[i])
			assert.NilError(t, err)

			err = addMemberToProviderGroups(tx, member, []string{testPg.GroupName})
			assert.NilError(t, err)

			memberIDs = append(memberIDs, member.IdentityID)
		}

		if len(memberIDs) > 0 {
			err = AddUsersToGroup(tx, parentGroups[testPg.GroupName].ID, pg.Name, pg.ProviderID, memberIDs)
			assert.NilError(t, err)
		}

		created = append(created, *pg)
	}

	return created
}

package data

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

var cmpModelsGroupShallow = cmp.Comparer(func(x, y models.Group) bool {
	return x.Name == y.Name && x.OrganizationID == y.OrganizationID
})

func TestCreateIdentity(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("success", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)
			bond := models.Identity{
				Name:      "jbond@infrahq.com",
				CreatedBy: uid.ID(777),
			}
			err := CreateIdentity(tx, &bond)
			assert.NilError(t, err)
			assert.Assert(t, bond.ID != 0)
			assert.Assert(t, bond.VerificationToken != "", "verification token must be set")

			actual, err := GetIdentity(tx, ByID(bond.ID))
			assert.NilError(t, err)
			assert.DeepEqual(t, actual, &bond, cmpTimeWithDBPrecision)
		})
	})
}

func createIdentities(t *testing.T, db GormTxn, identities ...*models.Identity) {
	t.Helper()
	for _, user := range identities {
		err := CreateIdentity(db, user)
		assert.NilError(t, err, user.Name)
		_, err = CreateProviderUser(db, InfraProvider(db), user)
		assert.NilError(t, err, user.Name)

		for _, group := range user.Groups {
			err = AddUsersToGroup(db, group.ID, []uid.ID{user.ID})
			assert.NilError(t, err)
		}
	}
}

// TODO: combine test cases for CreateIdentity

func TestCreateIdentity_DuplicateName(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			bond   = models.Identity{Name: "jbond@infrahq.com"}
			bourne = models.Identity{Name: "jbourne@infrahq.com"}
			bauer  = models.Identity{Name: "jbauer@infrahq.com"}
		)

		createIdentities(t, db, &bond, &bourne, &bauer)

		b := bond
		b.ID = 0
		err := CreateIdentity(db, &b)
		assert.ErrorContains(t, err, "a user with that name already exists")
	})
}

func TestCreateIdentity_DuplicateNameAfterDelete(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			bond   = models.Identity{Name: "jbond@infrahq.com"}
			bourne = models.Identity{Name: "jbourne@infrahq.com"}
			bauer  = models.Identity{Name: "jbauer@infrahq.com"}
		)

		createIdentities(t, db, &bond, &bourne, &bauer)

		err := DeleteIdentities(db, InfraProvider(db).ID, ByName(bond.Name))
		assert.NilError(t, err)

		err = CreateIdentity(db, &models.Identity{Name: bond.Name})
		assert.NilError(t, err)
	})
}

func TestGetIdentity(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			bond   = models.Identity{Name: "jbond@infrahq.com"}
			bourne = models.Identity{Name: "jbourne@infrahq.com"}
			bauer  = models.Identity{Name: "jbauer@infrahq.com"}
		)

		createIdentities(t, db, &bond, &bourne, &bauer)

		identity, err := GetIdentity(db, ByName(bond.Name))
		assert.NilError(t, err)
		assert.Assert(t, 0 != identity.ID)
	})
}

// TODO: test cases for UpdateIdentity

func TestListIdentities(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			everyone  = models.Group{Name: "Everyone"}
			engineers = models.Group{Name: "Engineering"}
			product   = models.Group{Name: "Product"}
		)
		createGroups(t, db, &everyone, &engineers, &product)

		var (
			bond = models.Identity{
				Name:   "jbond@infrahq.com",
				Groups: []models.Group{everyone, engineers},
			}
			bourne = models.Identity{
				Name:   "jbourne@infrahq.com",
				Groups: []models.Group{everyone, product},
			}
			bauer = models.Identity{
				Name:   "jbauer@infrahq.com",
				Groups: []models.Group{everyone},
			}
		)
		createIdentities(t, db, &bond, &bourne, &bauer)

		connector := InfraConnectorIdentity(db)

		t.Run("list all", func(t *testing.T) {
			identities, err := ListIdentities(db, nil)
			assert.NilError(t, err)
			expected := []models.Identity{*connector, bauer, bond, bourne}
			assert.DeepEqual(t, identities, expected, cmpModelsIdentityShallow)
		})

		t.Run("filter by name", func(t *testing.T) {
			identities, err := ListIdentities(db, nil, ByName(bourne.Name))
			assert.NilError(t, err)
			expected := []models.Identity{bourne}
			assert.DeepEqual(t, identities, expected, cmpModelsIdentityShallow)
		})

		t.Run("filter identities by group", func(t *testing.T) {
			actual, err := ListIdentities(db, nil, ByOptionalIdentityGroupID(everyone.ID))
			assert.NilError(t, err)
			expected := []models.Identity{bauer, bond, bourne}
			assert.DeepEqual(t, actual, expected, cmpModelsIdentityShallow)
		})

		t.Run("filter identities by different group", func(t *testing.T) {
			actual, err := ListIdentities(db, nil, ByOptionalIdentityGroupID(engineers.ID))
			assert.NilError(t, err)
			expected := []models.Identity{bond}
			assert.DeepEqual(t, actual, expected, cmpModelsIdentityShallow)
		})

		t.Run("filter identities by group and name", func(t *testing.T) {
			actual, err := ListIdentities(db, nil, ByOptionalIdentityGroupID(everyone.ID), ByName(bauer.Name))
			assert.NilError(t, err)
			expected := []models.Identity{bauer}
			assert.DeepEqual(t, actual, expected, cmpModelsIdentityShallow)
		})
	})
}

var cmpModelsIdentityShallow = cmp.Comparer(func(x, y models.Identity) bool {
	return x.Name == y.Name
})

func TestDeleteIdentity(t *testing.T) {
	type testCase struct {
		name   string
		setup  func(t *testing.T, tx *Transaction) (providerID uid.ID, identity models.Identity)
		verify func(t *testing.T, tx *Transaction, err error, identity models.Identity)
	}
	testCases := []testCase{
		{
			name: "valid delete infra provider user",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, identity models.Identity) {
				var (
					bond   = models.Identity{Name: "jbond@infrahq.com"}
					bourne = models.Identity{Name: "jbourne@infrahq.com"}
				)

				createIdentities(t, tx, &bond, &bourne)

				hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
				assert.NilError(t, err)
				err = CreateCredential(tx, &models.Credential{IdentityID: bond.ID, PasswordHash: hash})
				assert.NilError(t, err)

				group := &models.Group{
					Name: "test group",
				}
				err = CreateGroup(tx, group)
				assert.NilError(t, err)
				err = AddUsersToGroup(tx, group.ID, []uid.ID{bond.ID})
				assert.NilError(t, err)

				err = CreateGrant(tx, &models.Grant{Subject: bond.PolyID(), Privilege: "admin", Resource: "infra"})
				assert.NilError(t, err)

				return InfraProvider(tx).ID, bond
			},
			verify: func(t *testing.T, tx *Transaction, err error, identity models.Identity) {
				assert.NilError(t, err)

				_, err = GetIdentity(tx, ByName("jbond@infrahq.com"))
				assert.Error(t, err, "record not found")

				// when an identity has no more references its resources are cleaned up
				_, err = GetCredential(tx, ByIdentityID(identity.ID))
				assert.Error(t, err, "record not found")
				groupIDs, err := groupIDsForUser(tx, identity.ID)
				assert.NilError(t, err)
				assert.Equal(t, len(groupIDs), 0)
				grants, err := ListGrants(tx, ListGrantsOptions{BySubject: identity.PolyID()})
				assert.NilError(t, err)
				assert.Equal(t, len(grants), 0)

				// deleting a identity should not delete unrelated identities
				_, err = GetIdentity(tx, ByName("jbourne@infrahq.com"))
				assert.NilError(t, err)
			},
		},
		{
			name: "deleting non-existent user does not fail",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, identity models.Identity) {
				return InfraProvider(tx).ID, models.Identity{Name: "DNE"}
			},
			verify: func(t *testing.T, tx *Transaction, err error, identity models.Identity) {
				assert.NilError(t, err)
			},
		},
		{
			name: "delete identity in provider outside infra does not delete credentials",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, identity models.Identity) {
				id := &models.Identity{Name: "jbond@infrahq.com"}
				createIdentities(t, tx, id)

				err := CreateCredential(tx, &models.Credential{IdentityID: id.ID, PasswordHash: []byte("abc")})
				assert.NilError(t, err)

				provider := &models.Provider{
					Name: "other",
					Kind: models.ProviderKindOIDC,
				}
				err = CreateProvider(tx, provider)
				assert.NilError(t, err)

				_, err = CreateProviderUser(tx, provider, id)
				assert.NilError(t, err)

				return provider.ID, *id
			},
			verify: func(t *testing.T, tx *Transaction, err error, identity models.Identity) {
				assert.NilError(t, err)

				id, err := GetIdentity(tx, ByName(identity.Name))
				assert.NilError(t, err) // still exists in infra provider

				_, err = GetCredential(tx, ByIdentityID(id.ID))
				assert.NilError(t, err) // still exists in infra provider
			},
		},
	}
	runDBTests(t, func(t *testing.T, db *DB) {
		org := &models.Organization{Name: "something", Domain: "example.com"}
		assert.NilError(t, CreateOrganization(db, org))

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tx := txnForTestCase(t, db, org.ID)

				providerID, identity := tc.setup(t, tx)

				err := DeleteIdentities(tx, providerID, ByName(identity.Name))

				tc.verify(t, tx, err, identity)
			})
		}
	})
}

func TestDeleteIdentityWithGroups(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			bond   = models.Identity{Name: "jbond@infrahq.com"}
			bourne = models.Identity{Name: "jbourne@infrahq.com"}
			bauer  = models.Identity{Name: "jbauer@infrahq.com"}
		)
		group := &models.Group{Name: "Agents"}
		err := CreateGroup(db, group)
		assert.NilError(t, err)

		createIdentities(t, db, &bond, &bourne, &bauer)

		err = AddUsersToGroup(db, group.ID, []uid.ID{bond.ID, bourne.ID, bauer.ID})
		assert.NilError(t, err)

		err = DeleteIdentities(db, InfraProvider(db).ID, ByName(bond.Name))
		assert.NilError(t, err)

		group, err = GetGroup(db, ByID(group.ID))
		assert.NilError(t, err)
		assert.Equal(t, group.TotalUsers, 2)
	})
}

func TestAssignIdentityToGroups(t *testing.T) {
	tests := []struct {
		Name           string
		StartingGroups []string       // groups identity starts with
		ExistingGroups []string       // groups from last provider sync
		IncomingGroups []string       // groups from this provider sync
		ExpectedGroups []models.Group // groups identity should have at end
	}{
		{
			Name:           "test where the provider is trying to add a group the identity doesn't have elsewhere",
			StartingGroups: []string{"foo"},
			ExistingGroups: []string{},
			IncomingGroups: []string{"foo2"},
			ExpectedGroups: []models.Group{
				{
					Name: "foo",
					OrganizationMember: models.OrganizationMember{
						OrganizationID: 1000,
					},
				},
				{
					Name: "foo2",
					OrganizationMember: models.OrganizationMember{
						OrganizationID: 1000,
					},
				},
			},
		},
		{
			Name:           "test where the provider is trying to add a group the identity has from elsewhere",
			StartingGroups: []string{"foo"},
			ExistingGroups: []string{},
			IncomingGroups: []string{"foo", "foo2"},
			ExpectedGroups: []models.Group{
				{
					Name: "foo",
					OrganizationMember: models.OrganizationMember{
						OrganizationID: 1000,
					},
				},
				{
					Name: "foo2",
					OrganizationMember: models.OrganizationMember{
						OrganizationID: 1000,
					},
				},
			},
		},
		{
			Name:           "test where the group with the same name exists in another org",
			StartingGroups: []string{},
			ExistingGroups: []string{},
			IncomingGroups: []string{"Everyone"},
			ExpectedGroups: []models.Group{
				{
					Name: "Everyone",
					OrganizationMember: models.OrganizationMember{
						OrganizationID: 1000,
					},
				},
			},
		},
	}

	runDBTests(t, func(t *testing.T, db *DB) {
		otherOrg := &models.Organization{Name: "Other", Domain: "other.example.org"}
		assert.NilError(t, CreateOrganization(db, otherOrg))
		tx := txnForTestCase(t, db, otherOrg.ID)
		group := &models.Group{Name: "Everyone"}
		assert.NilError(t, CreateGroup(tx, group))
		assert.NilError(t, tx.Commit())

		for i, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				// setup identity
				identity := &models.Identity{Name: fmt.Sprintf("foo+%d@example.com", i)}
				err := CreateIdentity(db, identity)
				assert.NilError(t, err)

				// setup identity's groups
				for _, gn := range test.StartingGroups {
					g, err := GetGroup(db, ByName(gn))
					if errors.Is(err, internal.ErrNotFound) {
						g = &models.Group{Name: gn}
						err = CreateGroup(db, g)
						assert.NilError(t, err)
					}
					assert.NilError(t, AddUsersToGroup(db, g.ID, []uid.ID{identity.ID}))
				}

				// setup providerUser record
				provider := InfraProvider(db)
				pu, err := CreateProviderUser(db, provider, identity)
				assert.NilError(t, err)

				pu.Groups = test.ExistingGroups
				err = UpdateProviderUser(db, pu)
				assert.NilError(t, err)

				err = AssignIdentityToGroups(db, identity, provider, test.IncomingGroups)
				assert.NilError(t, err)

				// check the result
				actual, err := ListGroups(db, nil, ByGroupMember(identity.ID))
				assert.NilError(t, err)

				assert.DeepEqual(t, actual, test.ExpectedGroups, cmpModelsGroupShallow)
			})
		}
	})
}

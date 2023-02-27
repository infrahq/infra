package data

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
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

			actual, err := GetIdentity(tx, GetIdentityOptions{ByID: bond.ID})
			assert.NilError(t, err)
			assert.DeepEqual(t, actual, &bond, cmpTimeWithDBPrecision)
		})
	})
}

func createIdentities(t *testing.T, db WriteTxn, identities ...*models.Identity) {
	t.Helper()
	for _, user := range identities {
		err := CreateIdentity(db, user)
		assert.NilError(t, err, user.Name)
		if len(user.Providers) == 0 {
			_, err = CreateProviderUser(db, InfraProvider(db), user)
			assert.NilError(t, err, user.Name)
		} else {
			for i := range user.Providers {
				_, err = CreateProviderUser(db, &user.Providers[i], user)
				assert.NilError(t, err, user.Name)
			}
		}

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

		opts := DeleteIdentitiesOptions{
			ByProviderID: InfraProvider(db).ID,
			ByID:         bond.ID,
		}
		err := DeleteIdentities(db, opts)
		assert.NilError(t, err)

		err = CreateIdentity(db, &models.Identity{Name: bond.Name})
		assert.NilError(t, err)
	})
}

func TestSetSSHLoginName(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		type testCase struct {
			name     string
			email    string
			expected string
			setup    func(t *testing.T, tx *Transaction)
		}

		run := func(t *testing.T, tc testCase) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			if tc.setup != nil {
				tc.setup(t, tx)
			}

			user := &models.Identity{Name: tc.email}
			assert.NilError(t, insert(tx, (*identitiesTable)(user)))

			generate.SetSeed(500)

			username, err := setSSHLoginName(tx, *user)
			assert.NilError(t, err)
			assert.Equal(t, username, tc.expected)

			actual, err := GetIdentity(tx, GetIdentityOptions{ByID: user.ID})
			assert.NilError(t, err)
			assert.Equal(t, actual.SSHLoginName, username)
		}

		testCases := []testCase{
			{
				name:     "valid name from email",
				email:    "developer@example.com",
				expected: "developer",
			},
			{
				name:  "conflict on first try",
				email: "taken@example.com",
				setup: func(t *testing.T, tx *Transaction) {
					user := &models.Identity{
						Name:              "taken@otherdomain.com",
						SSHLoginName:      "taken",
						VerificationToken: "10001",
					}
					assert.NilError(t, insert(tx, (*identitiesTable)(user)))
				},
				expected: "taken446",
			},
			{
				name:  "conflict on second try",
				email: "taken@example.com",
				setup: func(t *testing.T, tx *Transaction) {
					user := &models.Identity{
						Name:              "taken@otherdomain.com",
						SSHLoginName:      "taken",
						VerificationToken: "10001",
					}
					assert.NilError(t, insert(tx, (*identitiesTable)(user)))
					user = &models.Identity{
						Name:              "taken@thirddomain.com",
						SSHLoginName:      "taken446",
						VerificationToken: "10002",
					}
					assert.NilError(t, insert(tx, (*identitiesTable)(user)))
				},
				expected: "taken740",
			},
			{
				name:     "uppercase and invalid characters are normalized",
				email:    "AH.What@example.com",
				expected: "ahwhat",
			},
			{
				name:     "starts with number is normalized",
				email:    "12rings@example.com",
				expected: "u12rings",
			},
			{
				name:     "long username is truncated",
				email:    "thisusernameisTOOOOOOOOOOlongforlinux@example.com",
				expected: "thisusernameistoooooooooolon",
			},
			{
				name:  "long username with conflict",
				email: "thisusernameisTOOOOOOOOOOlongforlinux@example.com",
				setup: func(t *testing.T, tx *Transaction) {
					user := &models.Identity{
						Name:              "taken@otherdomain.com",
						SSHLoginName:      "thisusernameistoooooooooolon",
						VerificationToken: "10003",
					}
					assert.NilError(t, insert(tx, (*identitiesTable)(user)))
				},
				expected: "thisusernameistoooooooooolon446",
			},
			{
				name:     "short username",
				email:    "m@example.com",
				expected: "m446",
			},
			{
				name:     "username conflicts with reserved name",
				email:    "root@example.com",
				expected: "root446",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				run(t, tc)
			})
		}
	})
}

func TestGetIdentity(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		group := models.Group{Name: "usa"}
		err := CreateGroup(db, &group)
		group.TotalUsers = 1
		assert.NilError(t, err)

		var (
			bond   = models.Identity{Name: "jbond@infrahq.com", SSHLoginName: "jbond"}
			bourne = models.Identity{Name: "jbourne@infrahq.com", Groups: []models.Group{group}}
			bauer  = models.Identity{Name: "jbauer@infrahq.com", Providers: []models.Provider{*InfraProvider(db)}}
			salt   = models.Identity{Name: "salt@infrahq.com", Providers: []models.Provider{googleProvider()}}
		)

		createIdentities(t, db, &bond, &bourne, &bauer, &salt)

		bondKey := &models.UserPublicKey{
			UserID:      bond.ID,
			Name:        "the-name",
			PublicKey:   "the-key",
			KeyType:     "ssh-rsa",
			Fingerprint: "the-fingerprint",
			ExpiresAt:   time.Now().Add(time.Hour).Truncate(time.Millisecond),
		}
		err = AddUserPublicKey(db, bondKey)
		assert.NilError(t, err)

		t.Run("ID or name are required", func(t *testing.T) {
			_, err := GetIdentity(db, GetIdentityOptions{})
			assert.ErrorContains(t, err, "GetIdentity must specify id or name")
		})
		t.Run("by ID", func(t *testing.T) {
			identity, err := GetIdentity(db, GetIdentityOptions{ByID: bond.ID})
			assert.NilError(t, err)
			assert.DeepEqual(t, *identity, bond, cmpTimeWithDBPrecision)
		})
		t.Run("by name", func(t *testing.T) {
			identity, err := GetIdentity(db, GetIdentityOptions{ByName: bond.Name})
			assert.NilError(t, err)
			assert.DeepEqual(t, *identity, bond, cmpTimeWithDBPrecision)
		})
		t.Run("load groups", func(t *testing.T) {
			identity, err := GetIdentity(db, GetIdentityOptions{ByName: bourne.Name, LoadGroups: true})
			assert.NilError(t, err)
			assert.DeepEqual(t, *identity, bourne, cmpTimeWithDBPrecision)
		})
		t.Run("load providers", func(t *testing.T) {
			identity, err := GetIdentity(db, GetIdentityOptions{ByName: bauer.Name, LoadProviders: true})
			assert.NilError(t, err)
			assert.DeepEqual(t, *identity, bauer, cmpTimeWithDBPrecision)
		})
		t.Run("load providers only exists in social", func(t *testing.T) {
			identity, err := GetIdentity(db, GetIdentityOptions{ByName: salt.Name, LoadProviders: true})
			assert.NilError(t, err)
			assert.DeepEqual(t, *identity, salt, cmpTimeWithDBPrecision)
		})
		t.Run("load public keys", func(t *testing.T) {
			identity, err := GetIdentity(db, GetIdentityOptions{ByName: bond.Name, LoadPublicKeys: true})
			assert.NilError(t, err)

			expected := bond // shallow copy
			expected.PublicKeys = []models.UserPublicKey{
				{
					Model:       bondKey.Model,
					UserID:      bond.ID,
					Name:        "the-name",
					PublicKey:   "the-key",
					KeyType:     "ssh-rsa",
					Fingerprint: "the-fingerprint",
					ExpiresAt:   bondKey.ExpiresAt,
				},
			}
			assert.DeepEqual(t, *identity, expected, cmpTimeWithDBPrecision)
		})
		t.Run("from other organization", func(t *testing.T) {
			_, err := GetIdentity(db, GetIdentityOptions{
				ByID:             bond.ID,
				FromOrganization: 71234,
			})
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
	})
}

func TestListIdentities(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			everyone = models.Group{Name: "Everyone"}
			product  = models.Group{Name: "Product"}
		)
		createGroups(t, db, &everyone, &product)

		providers := []models.Provider{*InfraProvider(db)}
		var (
			bond = models.Identity{
				Name:      "jbond@infrahq.com",
				Providers: providers,
			}
			salt = models.Identity{
				Name:      "salt@infrahq.com",
				CreatedBy: 1000,
				Providers: providers,
			}
			bourne = models.Identity{
				Name:      "jbourne@infrahq.com",
				Groups:    []models.Group{everyone, product},
				Providers: providers,
			}
			bauer = models.Identity{
				Name:      "jbauer@infrahq.com",
				Groups:    []models.Group{everyone},
				Providers: providers,
			}
		)
		createIdentities(t, db, &bond, &salt, &bourne, &bauer)

		connector := InfraConnectorIdentity(db)

		t.Run("list all", func(t *testing.T) {
			identities, err := ListIdentities(db, ListIdentityOptions{})
			assert.NilError(t, err)
			expected := []models.Identity{*connector, bauer, bond, bourne, salt}
			assert.DeepEqual(t, identities, expected, cmpModelsIdentityShallow)
		})

		t.Run("filter by ID", func(t *testing.T) {
			identities, err := ListIdentities(db, ListIdentityOptions{ByID: bond.ID})
			assert.NilError(t, err)
			expected := []models.Identity{bond}
			assert.DeepEqual(t, identities, expected, cmpModelsIdentityShallow)
		})

		t.Run("filter by IDs", func(t *testing.T) {
			identities, err := ListIdentities(db, ListIdentityOptions{ByIDs: []uid.ID{bond.ID, salt.ID}})
			assert.NilError(t, err)
			expected := []models.Identity{bond, salt}
			assert.DeepEqual(t, identities, expected, cmpModelsIdentityShallow)
		})

		t.Run("filter by name", func(t *testing.T) {
			identities, err := ListIdentities(db, ListIdentityOptions{ByName: bond.Name})
			assert.NilError(t, err)
			expected := []models.Identity{bond}
			assert.DeepEqual(t, identities, expected, cmpModelsIdentityShallow)
		})

		t.Run("filter by not name", func(t *testing.T) {
			identities, err := ListIdentities(db, ListIdentityOptions{ByNotName: bond.Name})
			assert.NilError(t, err)
			expected := []models.Identity{*connector, bauer, bourne, salt}
			assert.DeepEqual(t, identities, expected, cmpModelsIdentityShallow)
		})

		t.Run("filter identities by group", func(t *testing.T) {
			actual, err := ListIdentities(db, ListIdentityOptions{ByGroupID: everyone.ID})
			assert.NilError(t, err)
			expected := []models.Identity{bauer, bourne}
			assert.DeepEqual(t, actual, expected, cmpModelsIdentityShallow)
		})

		t.Run("filter identities by group and name", func(t *testing.T) {
			actual, err := ListIdentities(db, ListIdentityOptions{ByGroupID: everyone.ID, ByName: bauer.Name})
			assert.NilError(t, err)
			expected := []models.Identity{bauer}
			assert.DeepEqual(t, actual, expected, cmpModelsIdentityShallow)
		})

		t.Run("load groups", func(t *testing.T) {
			actual, err := ListIdentities(db, ListIdentityOptions{LoadGroups: true})
			assert.NilError(t, err)
			expected := []models.Identity{*connector, bauer, bond, bourne, salt}
			assert.DeepEqual(t, actual, expected, cmpModelsIdentityPreloadGroupsShallow)
		})

		t.Run("load providers", func(t *testing.T) {
			actual, err := ListIdentities(db, ListIdentityOptions{LoadProviders: true})
			assert.NilError(t, err)
			expected := []models.Identity{*connector, bauer, bond, bourne, salt}
			assert.DeepEqual(t, actual, expected, cmpModelsIdentityPreloadProvidersShallow)
		})
		t.Run("by public key fingerprint", func(t *testing.T) {
			pubKey := &models.UserPublicKey{
				UserID:      salt.ID,
				Fingerprint: "the-fingerprint",
				KeyType:     "ssh-rsa",
				PublicKey:   "the-public-key",
				ExpiresAt:   time.Now().Add(time.Hour).Truncate(time.Millisecond),
			}
			assert.NilError(t, AddUserPublicKey(db, pubKey))

			actual, err := ListIdentities(db, ListIdentityOptions{
				ByPublicKeyFingerprint: "the-fingerprint",
				LoadPublicKeys:         true,
			})
			assert.NilError(t, err)
			saltWithKey := salt // shallow copy
			saltWithKey.PublicKeys = []models.UserPublicKey{*pubKey}
			saltWithKey.Providers = nil
			expected := []models.Identity{saltWithKey}
			assert.DeepEqual(t, actual, expected, cmpModel)
		})
	})
}

var cmpModelsIdentityShallow = cmp.Comparer(func(x, y models.Identity) bool {
	return x.Name == y.Name
})

var cmpModelsIdentityPreloadGroupsShallow = cmp.Comparer(func(x, y models.Identity) bool {
	if len(x.Groups) != len(y.Groups) {
		return false
	}
	for i := range x.Groups {
		if x.Groups[i].Name != y.Groups[i].Name {
			return false
		}
	}
	return x.Name == y.Name
})

var cmpModelsIdentityPreloadProvidersShallow = cmp.Comparer(func(x, y models.Identity) bool {
	if len(x.Providers) != len(y.Providers) {
		return false
	}
	for i := range x.Providers {
		if x.Providers[i].Name != y.Providers[i].Name {
			return false
		}
	}
	return x.Name == y.Name
})

func TestUpdateIdentity(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		identity := models.Identity{
			Name:              "Alice",
			Verified:          false,
			VerificationToken: "aaa",
		}
		err := CreateIdentity(db, &identity)
		assert.NilError(t, err)

		identity.Name = "Bob"
		identity.Verified = true
		identity.VerificationToken = "bbb"

		err = UpdateIdentity(db, &identity)
		assert.NilError(t, err)

		result, err := GetIdentity(db, GetIdentityOptions{ByID: identity.ID})
		assert.NilError(t, err)
		assert.DeepEqual(t, *result, identity, cmpTimeWithDBPrecision)
	})
}

func TestDeleteIdentities(t *testing.T) {
	type testCase struct {
		name   string
		setup  func(t *testing.T, tx *Transaction) (opts DeleteIdentitiesOptions, identity models.Identity)
		verify func(t *testing.T, tx *Transaction, err error, identity models.Identity)
	}
	testCases := []testCase{
		{
			name: "valid delete infra provider user",
			setup: func(t *testing.T, tx *Transaction) (opts DeleteIdentitiesOptions, identity models.Identity) {
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

				err = CreateGrant(tx, &models.Grant{Subject: models.NewSubjectForUser(bond.ID), Privilege: "admin", Resource: "infra"})
				assert.NilError(t, err)

				return DeleteIdentitiesOptions{
					ByProviderID: InfraProvider(tx).ID,
					ByID:         bond.ID,
				}, bond
			},
			verify: func(t *testing.T, tx *Transaction, err error, identity models.Identity) {
				assert.NilError(t, err)

				_, err = GetIdentity(tx, GetIdentityOptions{ByName: "jbond@infrahq.com"})
				assert.Error(t, err, "record not found")

				// when an identity has no more references its resources are cleaned up
				_, err = GetCredentialByUserID(tx, identity.ID)
				assert.Error(t, err, "record not found")
				groupIDs, err := ListGroupIDsForUser(tx, identity.ID)
				assert.NilError(t, err)
				assert.Equal(t, len(groupIDs), 0)
				grants, err := ListGrants(tx, ListGrantsOptions{BySubject: models.NewSubjectForUser(identity.ID)})
				assert.NilError(t, err)
				assert.Equal(t, len(grants), 0)

				// deleting a identity should not delete unrelated identities
				_, err = GetIdentity(tx, GetIdentityOptions{ByName: "jbourne@infrahq.com"})
				assert.NilError(t, err)
			},
		},
		{
			name: "deleting non-existent user does not fail",
			setup: func(t *testing.T, tx *Transaction) (opts DeleteIdentitiesOptions, identity models.Identity) {
				return DeleteIdentitiesOptions{
					ByProviderID: InfraProvider(tx).ID,
					ByID:         123456789,
				}, models.Identity{}
			},
			verify: func(t *testing.T, tx *Transaction, err error, identity models.Identity) {
				assert.NilError(t, err)
			},
		},
		{
			name: "delete by ID",
			setup: func(t *testing.T, tx *Transaction) (opts DeleteIdentitiesOptions, identity models.Identity) {
				id := &models.Identity{Name: "name@infrahq.com"}
				createIdentities(t, tx, id)

				key1 := &models.UserPublicKey{
					UserID:      id.ID,
					Name:        "testing",
					PublicKey:   "the-pub-key",
					KeyType:     "ssh-rsa",
					Fingerprint: "the-fingerprint",
				}
				assert.NilError(t, AddUserPublicKey(tx, key1))

				return DeleteIdentitiesOptions{
					ByProviderID: InfraProvider(tx).ID,
					ByID:         id.ID,
				}, *id
			},
			verify: func(t *testing.T, tx *Transaction, err error, identity models.Identity) {
				assert.NilError(t, err)
				_, err = GetIdentity(tx, GetIdentityOptions{ByID: identity.ID})
				assert.Error(t, err, "record not found")

				keys, err := listUserPublicKeys(tx, identity.ID)
				assert.NilError(t, err)
				assert.Equal(t, len(keys), 0)
			},
		},
		{
			name: "delete by IDs",
			setup: func(t *testing.T, tx *Transaction) (opts DeleteIdentitiesOptions, identity models.Identity) {
				id1 := &models.Identity{
					Model: models.Model{ID: 1},
					Name:  "name1@infrahq.com",
				}
				id2 := &models.Identity{
					Model: models.Model{ID: 2},
					Name:  "name2@infrahq.com",
				}
				id3 := &models.Identity{
					Model: models.Model{ID: 3},
					Name:  "name3@infrahq.com",
				}
				createIdentities(t, tx, id1, id2, id3)
				return DeleteIdentitiesOptions{
					ByProviderID: InfraProvider(tx).ID,
					ByIDs:        []uid.ID{id1.ID, id2.ID},
				}, models.Identity{} // not used
			},
			verify: func(t *testing.T, tx *Transaction, err error, identity models.Identity) {
				assert.NilError(t, err)
				_, err = GetIdentity(tx, GetIdentityOptions{ByID: 1})
				assert.Error(t, err, "record not found")
				_, err = GetIdentity(tx, GetIdentityOptions{ByID: 2})
				assert.Error(t, err, "record not found")
				_, err = GetIdentity(tx, GetIdentityOptions{ByID: 3})
				assert.NilError(t, err) // still exists
			},
		},
		{
			name: "delete by name",
			setup: func(t *testing.T, tx *Transaction) (opts DeleteIdentitiesOptions, identity models.Identity) {
				id1 := &models.Identity{
					Model: models.Model{ID: 1},
					Name:  "name1@infrahq.com",
				}
				id2 := &models.Identity{
					Model: models.Model{ID: 2},
					Name:  "name2@infrahq.com",
				}
				createIdentities(t, tx, id1, id2)
				return DeleteIdentitiesOptions{
					ByProviderID: InfraProvider(tx).ID,
					ByID:         id1.ID,
				}, models.Identity{} // not used
			},
			verify: func(t *testing.T, tx *Transaction, err error, identity models.Identity) {
				assert.NilError(t, err)
				_, err = GetIdentity(tx, GetIdentityOptions{ByID: 1})
				assert.Error(t, err, "record not found")
				_, err = GetIdentity(tx, GetIdentityOptions{ByID: 2})
				assert.NilError(t, err) // still exists
			},
		},
	}
	runDBTests(t, func(t *testing.T, db *DB) {
		org := &models.Organization{Name: "something", Domain: "example.com"}
		assert.NilError(t, CreateOrganization(db, org))

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tx := txnForTestCase(t, db, org.ID)

				opts, identity := tc.setup(t, tx)
				err := DeleteIdentities(tx, opts)

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

		opts := DeleteIdentitiesOptions{
			ByProviderID: InfraProvider(db).ID,
			ByID:         bond.ID,
		}
		err = DeleteIdentities(db, opts)
		assert.NilError(t, err)

		group, err = GetGroup(db, GetGroupOptions{ByID: group.ID})
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
		{
			Name:           "test where user has groups from this provider removed",
			StartingGroups: []string{"foo"},
			ExistingGroups: []string{"foo"},
			IncomingGroups: []string{"foo2"},
			ExpectedGroups: []models.Group{
				{
					Name: "foo2",
					OrganizationMember: models.OrganizationMember{
						OrganizationID: 1000,
					},
				},
			},
		},
		{
			Name:           "test where the user has duplicate groups",
			StartingGroups: []string{"foo"},
			ExistingGroups: []string{"foo"},
			IncomingGroups: []string{"foo2", "foo2"},
			ExpectedGroups: []models.Group{
				{
					Name: "foo2",
					OrganizationMember: models.OrganizationMember{
						OrganizationID: 1000,
					},
				},
			},
		},
		{
			Name:           "test where the group has a comma in its name",
			StartingGroups: []string{"foo"},
			ExistingGroups: []string{},
			IncomingGroups: []string{"foo", "foo,2"},
			ExpectedGroups: []models.Group{
				{
					Name: "foo",
					OrganizationMember: models.OrganizationMember{
						OrganizationID: 1000,
					},
				},
				{
					Name: "foo,2",
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
					g, err := GetGroup(db, GetGroupOptions{ByName: gn})
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

				_, err = AssignIdentityToGroups(db, pu, test.ExistingGroups)
				assert.NilError(t, err)

				result, err := AssignIdentityToGroups(db, pu, test.IncomingGroups)
				assert.NilError(t, err)

				assert.DeepEqual(t, result, test.ExpectedGroups, cmpModelsGroupShallow)

				// check the persisted result
				persisted, err := ListGroups(db, ListGroupsOptions{ByGroupMember: identity.ID})
				assert.NilError(t, err)
				assert.DeepEqual(t, persisted, test.ExpectedGroups, cmpModelsGroupShallow)
			})
		}
	})
}

func TestCountAllIdentities(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		createIdentities(t, db,
			&models.Identity{Name: "one"},
			&models.Identity{Name: "two"},
			&models.Identity{Name: "three"},
			&models.Identity{Name: "four"},
			&models.Identity{Name: "five"})

		actual, err := CountAllIdentities(db)
		assert.NilError(t, err)
		assert.Equal(t, actual, int64(6)) // 5 + connector
	})
}

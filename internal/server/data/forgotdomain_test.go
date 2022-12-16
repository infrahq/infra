package data

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestGetForgottenDomainsForEmail(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		orgA := &models.Organization{Name: "A Team", Domain: "ateam"}
		err := CreateOrganization(db, orgA)
		assert.NilError(t, err)

		orgB := &models.Organization{Name: "B Team", Domain: "bteam"}
		err = CreateOrganization(db, orgB)
		assert.NilError(t, err)

		orgC := &models.Organization{Name: "C Team", Domain: "cteam"}
		err = CreateOrganization(db, orgC)
		assert.NilError(t, err)

		userA := &models.Identity{
			Name:               "john.smith@ateam.com",
			OrganizationMember: models.OrganizationMember{OrganizationID: orgA.ID},
			LastSeenAt:         time.Date(2022, 1, 2, 3, 4, 5, 600, time.UTC),
		}
		err = CreateIdentity(db, userA)
		assert.NilError(t, err)

		t.Run("no orgs", func(t *testing.T) {
			results, err := GetForgottenDomainsForEmail(db, "francis.lynch@teamusa.org")
			assert.NilError(t, err)
			assert.Equal(t, len(results), 0)
		})

		t.Run("single orgs", func(t *testing.T) {
			results, err := GetForgottenDomainsForEmail(db, userA.Name)
			assert.NilError(t, err)

			expected := []ForgottenDomain{{
				OrganizationName:   orgA.Name,
				OrganizationDomain: orgA.Domain,
				LastSeenAt:         userA.LastSeenAt,
			}}
			assert.DeepEqual(t, results, expected, cmpTimeWithDBPrecision)
		})

		userB := &models.Identity{
			Name:               "john.smith@ateam.com",
			OrganizationMember: models.OrganizationMember{OrganizationID: orgB.ID},
		}
		err = CreateIdentity(db, userB)
		assert.NilError(t, err)

		t.Run("multi orgs", func(t *testing.T) {
			results, err := GetForgottenDomainsForEmail(db, userA.Name)
			assert.NilError(t, err)

			expected := []ForgottenDomain{
				{
					OrganizationName:   orgA.Name,
					OrganizationDomain: orgA.Domain,
					LastSeenAt:         userA.LastSeenAt,
				},
				{
					OrganizationName:   orgB.Name,
					OrganizationDomain: orgB.Domain,
					LastSeenAt:         time.Time{},
				},
			}
			assert.DeepEqual(t, results, expected, cmpTimeWithDBPrecision)
		})

		t.Run("deleted user", func(t *testing.T) {
			deletedUser := &models.Identity{Name: "john.smith@ateam.com", OrganizationMember: models.OrganizationMember{OrganizationID: orgC.ID}}
			deletedUser.DeletedAt.Time = time.Now()
			deletedUser.DeletedAt.Valid = true
			err = CreateIdentity(db, deletedUser)

			results, err := GetForgottenDomainsForEmail(db, userA.Name)
			assert.NilError(t, err)
			assert.Equal(t, len(results), 2)
		})

		t.Run("deleted organization", func(t *testing.T) {
			deletedOrg := &models.Organization{Name: "D Team", Domain: "dteam"}
			deletedOrg.DeletedAt.Time = time.Now()
			deletedOrg.DeletedAt.Valid = true
			err = CreateOrganization(db, deletedOrg)
			assert.NilError(t, err)

			deletedUser := &models.Identity{Name: "john.smith@ateam.com", OrganizationMember: models.OrganizationMember{OrganizationID: deletedOrg.ID}}
			deletedUser.DeletedAt.Time = time.Now()
			deletedUser.DeletedAt.Valid = true
			err = CreateIdentity(db, deletedUser)

			results, err := GetForgottenDomainsForEmail(db, userA.Name)
			assert.NilError(t, err)
			assert.Equal(t, len(results), 2, "wrong number of users")
		})
	})
}

package data

import (
	"math"
	"strings"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateGrant(db *gorm.DB, grant *models.Grant) error {
	return add(db, grant)
}

func GetGrant(db *gorm.DB, selectors ...SelectorFunc) (*models.Grant, error) {
	return get[models.Grant](db, selectors...)
}

func ListUserGrants(db *gorm.DB, userID uid.ID) (result []models.Grant, err error) {
	return list[models.Grant](db, ByIdentityUserID(userID))
}

func ListGroupGrants(db *gorm.DB, groupID uid.ID) (result []models.Grant, err error) {
	return list[models.Grant](db, ByIdentityGroupID(groupID))
}

func ListGrants(db *gorm.DB, selectors ...SelectorFunc) ([]models.Grant, error) {
	return list[models.Grant](db, selectors...)
}

func DeleteGrants(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := list[models.Grant](db, selectors...)
	if err != nil {
		return err
	}

	ids := make([]uid.ID, 0)
	for _, g := range toDelete {
		ids = append(ids, g.ID)
	}

	return deleteAll[models.Grant](db, ByIDs(ids))
}

func Can(db *gorm.DB, identity, privilege, resource string) (bool, error) {
	grants, err := list[models.Grant](db, ByIdentity(identity), ByPrivilege(privilege), ByResource(resource))
	if err != nil {
		return false, err
	}

	for _, grant := range grants {
		if grant.Matches(identity, privilege, resource) {
			return true, nil
		}
	}

	return false, nil
}

// wildcardCombinations turns infra.foo.1 into:
// infra.foo.1
// infra.foo.*
// infra.*
// See TestWildcardCombinations for details
// the idea is to count in binary and use the binary int as a bitmask for which
// elements to swap out with a wildcard
func wildcardCombinations(s string) []string {
	results := []string{}
	parts := strings.Split(s, ".")
	max := math.Pow(2, float64(len(parts)))

	for i := 0; i < int(math.Ceil(max))/2; i++ {
		if i&0b11 == 0b10 { // skip *.<id> types, as it makes no sense.
			continue
		}
		parts = strings.Split(s, ".")
		j := i
		pos := len(parts) - 1
		for j > 0 {
			bit := j & 1
			j = j >> 1
			if bit == 1 {
				parts[pos] = "*"
			}
			pos--
			if pos == 0 {
				break
			}
		}
		s := strings.Join(parts, ".")
		for strings.HasSuffix(s, ".*.*") {
			s = s[:len(s)-2]
		}
		results = append(results, s)
	}

	return results
}

func ByPrivilege(s string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if s == "" {
			return db
		}

		return db.Where("privilege = ?", s)
	}
}

func ByResource(s string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if s == "" {
			return db
		}

		resources := wildcardCombinations(s)
		return db.Where("resource in (?)", resources)
	}
}

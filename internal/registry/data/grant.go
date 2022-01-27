package data

import (
	"fmt"
	"math"
	"strings"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/uid"
)

func CreateGrant(db *gorm.DB, grant *models.Grant) error {
	return add(db, grant)
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

// CreateOrUpdateGrant is deprecated; this function does not work properly, and can't be logically fixed;
// eg it can't remove grants that should no longer exist
func CreateOrUpdateGrant(db *gorm.DB, grant *models.Grant) (*models.Grant, error) {
	// A grant is unique by its resource, identity, and privilege
	g := &models.Grant{}
	err := db.Model((*models.Grant)(nil)).Where("identity = ? and privilege = ? and resource = ?", grant.Identity, grant.Privilege, grant.Resource).First(g).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if err == gorm.ErrRecordNotFound {
		err := CreateGrant(db, grant)
		return grant, err
	}

	if err := update(db, g.ID, grant); err != nil {
		return nil, err
	}

	return get[models.Grant](db, ByID(g.ID))
}

func GetGrant(db *gorm.DB, selectors ...SelectorFunc) (*models.Grant, error) {
	return get[models.Grant](db, selectors...)
}

func ListUserGrants(db *gorm.DB, userID uid.ID) (result []models.Grant, err error) {
	return list[models.Grant](db, ByIdentityUserID(userID))
}

func DeleteGrants(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := list[models.Grant](db, selectors...)
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uid.ID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return removeAll[models.Grant](db, ByIDs(ids))
	}

	return internal.ErrNotFound
}

func ByDestinationKind(kind models.DestinationKind) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(kind) == 0 {
			return db
		}

		switch kind {
		case models.DestinationKindInfra, models.DestinationKindKubernetes:
			return db.Where("kind = ?", kind)
		default:
			// panic("unknown grant kind: " + string(kind))
			db.Logger.Error(db.Statement.Context, "unknown destination kind: "+string(kind))
			_ = db.AddError(fmt.Errorf("%w: unknown destination kind: %q", internal.ErrBadRequest, string(kind)))

			return db.Where("1 = 2")
		}
	}
}

func ByProviderKind(kind models.ProviderKind) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(kind) == 0 {
			return db
		}

		switch kind {
		case models.ProviderKindOkta:
			return db.Where("kind = ?", kind)
		default:
			db.Logger.Error(db.Statement.Context, "unknown destination kind: "+string(kind))
			_ = db.AddError(fmt.Errorf("%w: unknown destination kind: %q", internal.ErrBadRequest, string(kind)))

			return db.Where("1 = 2")
		}
	}
}
func ByDomain(domain string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(domain) == 0 {
			return db
		}

		return db.Where("domain = ?", domain)
	}
}

func NotByIDs(ids []uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(ids) == 0 {
			return db
		}

		return db.Where("id not in (?)", ids)
	}
}

func ByIdentityUserID(userID uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("identity = ?", "u:"+userID.String())
	}
}

func ByIdentity(s string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("identity = ?", s)
	}
}

func ByResource(s string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		resources := wildcardCombinations(s)
		return db.Where("resource in (?)", resources)
	}
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
		return db.Where("privilege = ?", s)
	}
}

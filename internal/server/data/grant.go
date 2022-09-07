package data

import (
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type grantsTable models.Grant

func (g grantsTable) Table() string {
	return "grants"
}

func (g grantsTable) Columns() []string {
	return []string{"created_at", "created_by", "deleted_at", "id", "organization_id", "privilege", "resource", "subject", "updated_at"}
}

func (g grantsTable) Values() []any {
	return []any{g.CreatedAt, g.CreatedBy, g.DeletedAt, g.ID, g.OrganizationID, g.Privilege, g.Resource, g.Subject, g.UpdatedAt}
}

func (g *grantsTable) ScanFields() []any {
	return []any{&g.CreatedAt, &g.CreatedBy, &g.DeletedAt, &g.ID, &g.OrganizationID, &g.Privilege, &g.Resource, &g.Subject, &g.UpdatedAt}
}

func CreateGrant(tx WriteTxn, grant *models.Grant) error {
	switch {
	case grant.Subject == "":
		return fmt.Errorf("subject is required")
	case grant.Privilege == "":
		return fmt.Errorf("privilege is required")
	case grant.Resource == "":
		return fmt.Errorf("resource is required")
	}

	// Use a savepoint so that we can query for the duplicate grant on conflict
	if _, err := tx.Exec("SAVEPOINT beforeCreate"); err != nil {
		// ignore "not in a transaction" error, because outside of a transaction
		// the db conn can continue to be used after the conflict error.
		if !isPgErrorCode(err, pgerrcode.NoActiveSQLTransaction) {
			return err
		}
	}
	if err := insert(tx, (*grantsTable)(grant)); err != nil {
		_, _ = tx.Exec("ROLLBACK TO SAVEPOINT beforeCreate")
		return handleError(err)
	}
	_, _ = tx.Exec("RELEASE SAVEPOINT beforeCreate")
	return nil
}

func isPgErrorCode(err error, code string) bool {
	pgError := &pgconn.PgError{}
	return errors.As(err, &pgError) && pgError.Code == code
}

func GetGrant(db GormTxn, selectors ...SelectorFunc) (*models.Grant, error) {
	return get[models.Grant](db, selectors...)
}

func ListGrants(db GormTxn, p *Pagination, selectors ...SelectorFunc) ([]models.Grant, error) {
	return list[models.Grant](db, p, selectors...)
}

type DeleteGrantsOptions struct {
	// ByID instructs DeleteGrants to delete the grant with this ID. When set
	// all other fields on this struct are ignored.
	ByID uid.ID
	// BySubject instructs DeleteGrants to delete all grants that match this
	// subject. When set other fields below this on this struct are ignored.
	BySubject uid.PolymorphicID

	// ByCreatedBy instructs DeleteGrants to delete all the grants that were
	// created by this user. Can be used with NotIDs
	ByCreatedBy uid.ID
	// NotIDs instructs DeleteGrants to exclude any grants with these IDs to
	// be excluded. In other words, these IDs will not be deleted, even if they
	// match ByCreatedBy.
	// Can only be used with ByCreatedBy.
	NotIDs []uid.ID
}

func DeleteGrants(tx WriteTxn, opts DeleteGrantsOptions) error {
	query := querybuilder.New("UPDATE grants")
	query.B("SET deleted_at = ?", time.Now())
	query.B("WHERE organization_id = ? AND", tx.OrganizationID())
	query.B("deleted_at is null AND")

	switch {
	case opts.ByID != 0:
		query.B("id = ?", opts.ByID)
	case opts.BySubject != "":
		query.B("subject = ?", opts.BySubject)
	case opts.ByCreatedBy != 0:
		query.B("created_by = ?", opts.ByCreatedBy)
		if len(opts.NotIDs) > 0 {
			query.B("AND id not in (?)", opts.NotIDs)
		}
	default:
		return fmt.Errorf("DeleteGrants requires an ID to delete")
	}

	_, err := tx.Exec(query.String(), query.Args...)
	return err
}

func ByOptionalPrivilege(s string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if s == "" {
			return db
		}

		return db.Where("privilege = ?", s)
	}
}

func GrantsInheritedBySubject(subjectID uid.PolymorphicID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		switch {
		case subjectID.IsIdentity():
			userID, err := subjectID.ID()
			if err != nil {
				logging.Errorf("invalid subject id %q", subjectID)
				return db.Where("1 = 0")
			}
			var groupIDs []uid.ID
			err = db.Session(&gorm.Session{NewDB: true}).Raw("select distinct group_id from identities_groups where identity_id = ?", userID).Pluck("group_id", &groupIDs).Error
			if err != nil {
				logging.Errorf("GrantsInheritedByUser: %s", err)
				_ = db.AddError(err)
				return db.Where("1 = 0")
			}

			subjects := []string{subjectID.String()}
			for _, groupID := range groupIDs {
				subjects = append(subjects, uid.NewGroupPolymorphicID(groupID).String())
			}
			return db.Where("subject in (?)", subjects)
		case subjectID.IsGroup():
			return BySubject(subjectID)(db)
		default:
			panic("unhandled subject type")
		}
	}
}

func ByPrivilege(s string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("privilege = ?", s)
	}
}

func ByOptionalResource(s string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if s == "" {
			return db
		}

		return db.Where("resource = ?", s)
	}
}

func ByResource(s string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("resource = ?", s)
	}
}

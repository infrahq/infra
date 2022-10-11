package data

import (
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"

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

type GetGrantOptions struct {
	// ByID instructs GetGrant to return the grant with this primary key ID.
	// When set all other fields on this struct are ignored.
	ByID uid.ID

	// BySubject instructs GetGrant to return the grant with this subject. Must
	// be used with ByPrivilege, and ByResource.
	BySubject uid.PolymorphicID
	// ByPrivilege instructs GetGrant to return the grant with this privilege. Must
	// be used with BySubject, and ByResource.
	ByPrivilege string
	// ByResource instructs GetGrant to return the grant with this resource. Must
	// be used with BySubject, and ByPrivilege.
	ByResource string
}

func GetGrant(tx ReadTxn, opts GetGrantOptions) (*models.Grant, error) {
	table := &grantsTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(table))
	query.B("FROM grants")
	query.B("WHERE organization_id = ?", tx.OrganizationID())
	query.B("AND deleted_at is null")

	switch {
	case opts.ByID != 0:
		query.B("AND id = ?", opts.ByID)
	case opts.BySubject != "":
		query.B("AND subject = ?", opts.BySubject)
		query.B("AND privilege = ?", opts.ByPrivilege)
		query.B("AND resource = ?", opts.ByResource)
	default:
		return nil, fmt.Errorf("GetGrant requires an ID or subject")
	}

	err := tx.QueryRow(query.String(), query.Args...).Scan(table.ScanFields()...)
	if err != nil {
		return nil, handleReadError(err)
	}
	return (*models.Grant)(table), nil
}

type ListGrantsOptions struct {
	BySubject     uid.PolymorphicID
	ByPrivileges  []string
	ByResource    string
	ByDestination string

	// IncludeInheritedFromGroups instructs ListGrants to include grants from
	// groups where the user is a member. This option can only be used when
	// BySubject is a non-zero userID.
	IncludeInheritedFromGroups bool

	// ExcludeConnectorGrant instructs ListGrants to exclude grants where
	// privilege=connector and resource=infra.
	ExcludeConnectorGrant bool

	Pagination *Pagination
}

func ListGrants(tx ReadTxn, opts ListGrantsOptions) ([]models.Grant, error) {
	table := grantsTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(table))
	if opts.Pagination != nil {
		query.B(", count(*) OVER()")
	}
	query.B("FROM grants")
	query.B("WHERE deleted_at is null")
	query.B("AND organization_id = ?", tx.OrganizationID())

	if opts.BySubject != "" {
		if !opts.IncludeInheritedFromGroups {
			query.B("AND subject = ?", opts.BySubject)
		} else {
			subjects := []string{opts.BySubject.String()}

			userID, err := opts.BySubject.ID()
			if err != nil || !opts.BySubject.IsIdentity() {
				return nil, fmt.Errorf("IncludeInheritedFromGroups requires a userId subject")
			}
			// FIXME: store userID and groupID as a field on the grants table so
			// that we can replace this with a sub-select or join.
			groupIDs, err := groupIDsForUser(tx, userID)
			if err != nil {
				return nil, err
			}
			for _, id := range groupIDs {
				subjects = append(subjects, uid.NewGroupPolymorphicID(id).String())
			}
			query.B(`AND subject IN (?)`, subjects)
		}
	}
	if len(opts.ByPrivileges) > 0 {
		query.B("AND privilege IN (?)", opts.ByPrivileges)
	}
	if opts.ByResource != "" {
		query.B("AND resource = ?", opts.ByResource)
	}
	if opts.ByDestination != "" {
		query.B("AND (resource = ? OR resource LIKE ?)", opts.ByDestination, opts.ByDestination+".%")
	}
	if opts.ExcludeConnectorGrant {
		query.B("AND NOT (privilege = 'connector' AND resource = 'infra')")
	}

	query.B("ORDER BY id ASC")
	if opts.Pagination != nil {
		opts.Pagination.PaginateQuery(query)
	}

	rows, err := tx.Query(query.String(), query.Args...)
	if err != nil {
		return nil, err
	}
	return scanRows(rows, func(grant *models.Grant) []any {
		fields := (*grantsTable)(grant).ScanFields()
		if opts.Pagination != nil {
			fields = append(fields, &opts.Pagination.TotalCount)
		}
		return fields
	})
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

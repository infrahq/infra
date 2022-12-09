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

	if err := grant.OnInsert(); err != nil {
		return err
	}
	setOrg(tx, grant)

	// Use a savepoint so that we can query for the duplicate grant on conflict
	if _, err := tx.Exec("SAVEPOINT beforeCreate"); err != nil {
		// ignore "not in a transaction" error, because outside of a transaction
		// the db conn can continue to be used after the conflict error.
		if !isPgErrorCode(err, pgerrcode.NoActiveSQLTransaction) {
			return err
		}
	}

	table := (*grantsTable)(grant)
	query := querybuilder.New("INSERT INTO grants (")
	query.B(columnsForInsert(table))
	query.B(", update_index")
	query.B(") VALUES (")
	query.B(placeholderForColumns(table), table.Values()...)
	query.B(", nextval('seq_update_index'));")
	_, err := tx.Exec(query.String(), query.Args...)
	if err != nil {
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
	query.B(", update_index")
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

	fields := append(table.ScanFields(), &table.UpdateIndex)
	err := tx.QueryRow(query.String(), query.Args...).Scan(fields...)
	if err != nil {
		return nil, handleError(err)
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
	query.B(", update_index")
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
			groupIDs, err := ListGroupIDsForUser(tx, userID)
			if err != nil {
				return nil, err
			}
			for _, id := range groupIDs {
				subjects = append(subjects, uid.NewGroupPolymorphicID(id).String())
			}
			query.B(`AND subject IN`)
			queryInClause(query, subjects)
		}
	}
	if len(opts.ByPrivileges) > 0 {
		query.B("AND privilege IN")
		queryInClause(query, opts.ByPrivileges)
	}
	if opts.ByResource != "" {
		query.B("AND resource = ?", opts.ByResource)
	}
	if opts.ByDestination != "" {
		grantsByDestination(query, opts.ByDestination)
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
		fields := append((*grantsTable)(grant).ScanFields(), &grant.UpdateIndex)
		if opts.Pagination != nil {
			fields = append(fields, &opts.Pagination.TotalCount)
		}
		return fields
	})
}

func grantsByDestination(query *querybuilder.Query, destination string) {
	query.B("AND (resource = ? OR resource LIKE ?)", destination, destination+".%")
}

type GrantsMaxUpdateIndexOptions struct {
	ByDestination string
}

// GrantsMaxUpdateIndex returns the maximum update_index all the grants that
// match the query. This MUST include soft-deleted rows as well.
//
// Returns 1 if no records match the query, so that the caller can block until
// a record exists.
//
// TODO: any way to assert this tx has the right isolation level?
func GrantsMaxUpdateIndex(tx ReadTxn, opts GrantsMaxUpdateIndexOptions) (int64, error) {
	query := querybuilder.New("SELECT max(update_index) FROM grants")
	query.B("WHERE organization_id = ?", tx.OrganizationID())

	if opts.ByDestination != "" {
		grantsByDestination(query, opts.ByDestination)
	}

	var result *int64
	err := tx.QueryRow(query.String(), query.Args...).Scan(&result)
	if err != nil || result == nil {
		return 1, err
	}
	return *result, err
}

// grantJSON is used to decode the JSON payload from a channel notification.
// models.Grant does not work because it expects to decode uid.ID from a string
// not a number.
type grantJSON struct {
	Resource string
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
	query.B("SET deleted_at = ?,", time.Now())
	query.B("update_index = nextval('seq_update_index')")
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
			query.B("AND id NOT IN")
			queryInClause(query, opts.NotIDs)
		}
	default:
		return fmt.Errorf("DeleteGrants requires an ID to delete")
	}

	_, err := tx.Exec(query.String(), query.Args...)
	return err
}

func UpdateGrants(tx WriteTxn, addGrants, rmGrants []*models.Grant) error {
	// Use a savepoint so that we can query for the duplicate grant on conflict
	if _, err := tx.Exec("SAVEPOINT beforeUpdate"); err != nil {
		// ignore "not in a transaction" error, because outside of a transaction
		// the db conn can continue to be used after the conflict error.
		if !isPgErrorCode(err, pgerrcode.NoActiveSQLTransaction) {
			return err
		}
	}
	if err := createGrantsBulk(tx, addGrants); err != nil {
		_, _ = tx.Exec("ROLLBACK TO SAVEPOINT beforeUpdate")
		return handleError(err)
	}

	if err := deleteGrantsBulk(tx, rmGrants); err != nil {
		_, _ = tx.Exec("ROLLBACK TO SAVEPOINT beforeUpdate")
		return handleError(err)
	}

	_, _ = tx.Exec("RELEASE SAVEPOINT beforeUpdate")
	return nil
}

func validateGrant(grant *models.Grant) error {
	switch {
	case grant.Subject == "":
		return fmt.Errorf("subject is required")
	case grant.Privilege == "":
		return fmt.Errorf("privilege is required")
	case grant.Resource == "":
		return fmt.Errorf("resource is required")
	}
	return nil
}

func createGrantsBulk(tx WriteTxn, grants []*models.Grant) error {
	if len(grants) == 0 {
		return nil
	}

	for _, g := range grants {
		if err := validateGrant(g); err != nil {
			return err
		}
		if err := g.OnInsert(); err != nil {
			return err
		}
		setOrg(tx, g)
	}

	table := &grantsTable{}
	query := querybuilder.New("INSERT INTO grants")
	query.B("(")
	query.B(columnsForInsert(table))
	query.B(", update_index")
	query.B(") VALUES")

	for i, g := range grants {
		gt := (*grantsTable)(g)
		query.B("(")
		query.B(placeholderForColumns(table), gt.Values()...)
		query.B(", nextval('seq_update_index')")
		query.B(")")
		if i+1 != len(grants) {
			query.B(",")
		}
	}
	query.B("ON CONFLICT DO NOTHING")

	_, err := tx.Exec(query.String(), query.Args...)
	return err
}

func deleteGrantsBulk(tx WriteTxn, grants []*models.Grant) error {
	if len(grants) == 0 {
		return nil
	}

	for _, g := range grants {
		if err := validateGrant(g); err != nil {
			return err
		}
		if err := g.OnUpdate(); err != nil {
			return err
		}
	}

	query := querybuilder.New("UPDATE grants ")
	query.B("SET deleted_at = ?,", time.Now())
	query.B("update_index = nextval('seq_update_index')")
	query.B("WHERE organization_id = ? AND", tx.OrganizationID())
	query.B("deleted_at is null AND")
	query.B("id in (")

	for i, g := range grants {
		query.B("SELECT id FROM grants WHERE ")
		query.B("subject = ? AND", g.Subject)
		query.B("resource = ? AND", g.Resource)
		query.B("privilege = ?", g.Privilege)
		if i+1 != len(grants) {
			query.B("UNION")
		}
	}
	query.B(")")

	_, err := tx.Exec(query.String(), query.Args...)
	return err
}

func CountAllGrants(tx ReadTxn) (int64, error) {
	return countRows(tx, grantsTable{})
}

package data

import (
	"fmt"
	"sort"
	"time"

	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type destinationsTable models.Destination

func (d destinationsTable) Table() string {
	return "destinations"
}

func (d destinationsTable) Columns() []string {
	return []string{"connection_ca", "connection_url", "created_at", "deleted_at", "id", "kind", "last_seen_at", "name", "organization_id", "resources", "roles", "unique_id", "updated_at", "version"}
}

func (d destinationsTable) Values() []any {
	return []any{d.ConnectionCA, d.ConnectionURL, d.CreatedAt, d.DeletedAt, d.ID, d.Kind, d.LastSeenAt, d.Name, d.OrganizationID, d.Resources, d.Roles, (optionalString)(d.UniqueID), d.UpdatedAt, d.Version}
}

func (d *destinationsTable) ScanFields() []any {
	return []any{&d.ConnectionCA, &d.ConnectionURL, &d.CreatedAt, &d.DeletedAt, &d.ID, &d.Kind, &d.LastSeenAt, &d.Name, &d.OrganizationID, &d.Resources, &d.Roles, (*optionalString)(&d.UniqueID), &d.UpdatedAt, &d.Version}
}

func validateDestination(dest *models.Destination) error {
	if dest.Name == "" {
		return fmt.Errorf("Destination.Name is required")
	}
	if dest.Kind == "" {
		return fmt.Errorf("Destination.Kind is required")
	}
	return nil
}

func CreateDestination(tx WriteTxn, destination *models.Destination) error {
	if err := validateDestination(destination); err != nil {
		return err
	}
	return insert(tx, (*destinationsTable)(destination))
}

func UpdateDestination(tx WriteTxn, destination *models.Destination) error {
	if err := validateDestination(destination); err != nil {
		return err
	}
	return update(tx, (*destinationsTable)(destination))
}

// UpdateDestinationLastSeenAt sets dest.LastSeenAt to now and then updates the
// row in the database. Updates are throttled to once every 2 seconds.
// If the destination was updated recently, or the database row is already locked, the
// update will be skipped.
//
// Unlike most functions in this package, this function uses dest.OrganizationID
// not tx.OrganizationID.
func UpdateDestinationLastSeenAt(tx WriteTxn, dest *models.Destination) error {
	if time.Since(dest.LastSeenAt) < lastSeenUpdateThreshold {
		return nil
	}

	origUpdatedAt := dest.UpdatedAt
	dest.LastSeenAt = time.Now()
	if err := dest.OnUpdate(); err != nil {
		return err
	}

	table := (*destinationsTable)(dest)
	query := querybuilder.New("UPDATE destinations SET")
	query.B(columnsForUpdate(table), table.Values()...)
	query.B("WHERE deleted_at is null")
	query.B("AND organization_id = ?", dest.OrganizationID)
	// only update if the row has not changed since the SELECT
	query.B("AND updated_at = ?", origUpdatedAt)
	query.B("AND id IN (SELECT id from destinations WHERE id = ? FOR UPDATE SKIP LOCKED)", table.Primary())

	_, err := tx.Exec(query.String(), query.Args...)
	return handleError(err)
}

type GetDestinationOptions struct {
	// ByID instructs GetDestination to return the row matching this ID. When
	// this value is set, all other fields on this strut will be ignored
	ByID uid.ID
	// ByUniqueID instructs GetDestination to return the row matching this
	// uniqueID.
	ByUniqueID string
	// ByName instructs GetDestination to return the row matching this name.
	ByName string

	// FromOrganization is the organization ID of the provider. When set to a
	// non-zero value the organization ID from the transaction is ignored.
	FromOrganization uid.ID
}

func GetDestination(tx ReadTxn, opts GetDestinationOptions) (*models.Destination, error) {
	destination := destinationsTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(destination))
	query.B("FROM destinations")
	query.B("WHERE deleted_at is null")

	orgID := opts.FromOrganization
	if orgID == 0 {
		orgID = tx.OrganizationID()
	}
	query.B("AND organization_id = ?", orgID)

	switch {
	case opts.ByID != 0:
		query.B("AND id = ?", opts.ByID)
	case opts.ByUniqueID != "":
		query.B("AND unique_id = ?", opts.ByUniqueID)
	case opts.ByName != "":
		query.B("AND name = ?", opts.ByName)
	default:
		return nil, fmt.Errorf("an ID is required to GetDestination")
	}

	err := tx.QueryRow(query.String(), query.Args...).Scan(destination.ScanFields()...)
	if err != nil {
		return nil, handleError(err)
	}
	return (*models.Destination)(&destination), nil
}

type ListDestinationsOptions struct {
	ByUniqueID string
	ByName     string
	ByKind     string

	Pagination *Pagination
}

func ListDestinations(tx ReadTxn, opts ListDestinationsOptions) ([]models.Destination, error) {
	table := destinationsTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(table))
	if opts.Pagination != nil {
		query.B(", count(*) OVER()")
	}
	query.B("FROM destinations")
	query.B("WHERE deleted_at is null")
	query.B("AND organization_id = ?", tx.OrganizationID())

	if opts.ByUniqueID != "" {
		query.B("AND unique_id = ?", opts.ByUniqueID)
	}
	if opts.ByName != "" {
		query.B("AND name = ?", opts.ByName)
	}
	if opts.ByKind != "" {
		query.B("AND kind = ?", opts.ByKind)
	}

	query.B("ORDER BY name")
	if opts.Pagination != nil {
		opts.Pagination.PaginateQuery(query)
	}

	rows, err := tx.Query(query.String(), query.Args...)
	if err != nil {
		return nil, err
	}
	return scanRows(rows, func(d *models.Destination) []any {
		fields := (*destinationsTable)(d).ScanFields()
		if opts.Pagination != nil {
			fields = append(fields, &opts.Pagination.TotalCount)
		}
		return fields
	})
}

func DeleteDestination(tx WriteTxn, id uid.ID) error {
	dest, err := GetDestination(tx, GetDestinationOptions{ByID: id})
	if err != nil {
		return handleError(err)
	}

	err = DeleteGrants(tx, DeleteGrantsOptions{ByDestination: dest.Name})
	if err != nil {
		return handleError(err)
	}

	stmt := `
		UPDATE destinations SET deleted_at = ?
		WHERE id = ? AND organization_id = ? AND deleted_at is null
	`
	_, err = tx.Exec(stmt, time.Now(), id, tx.OrganizationID())
	return handleError(err)
}

type DestinationsCount struct {
	Connected bool
	Version   string
	Count     float64
}

func CountDestinationsByConnectedVersion(tx ReadTxn) ([]DestinationsCount, error) {
	timeout := time.Now().Add(-5 * time.Minute)

	stmt := `
		SELECT COALESCE(version, '') as version, last_seen_at >= ? as connected, count(*)
		FROM destinations
		WHERE deleted_at IS NULL
		GROUP BY connected, version
	`
	rows, err := tx.Query(stmt, timeout)
	if err != nil {
		return nil, err
	}
	return scanRows(rows, func(item *DestinationsCount) []any {
		return []any{&item.Version, &item.Connected, &item.Count}
	})
}

func CountAllDestinations(tx ReadTxn) (int64, error) {
	return countRows(tx, destinationsTable{})
}

type DestinationAccess struct {
	UserID           uid.ID
	UserSSHLoginName string
	Privilege        string
	Resource         string
}

func ListDestinationAccess(tx ReadTxn, destination string) ([]DestinationAccess, error) {
	// IMPORTANT: changes to this query likely also need to be applied to DestinationAccessMaxUpdateIndex
	query := querybuilder.New("SELECT")
	query.B("identities.id, identities.ssh_login_name, grants.privilege, grants.resource")
	query.B("FROM grants")
	query.B("JOIN identities ON grants.subject_id = identities.id")
	query.B("AND grants.subject_kind = ?", models.SubjectKindUser)
	query.B("WHERE grants.deleted_at is null")
	grantsByDestination(query, destination)
	query.B("AND grants.organization_id = ?", tx.OrganizationID())
	query.B("UNION ALL SELECT")
	query.B("identities.id, identities.ssh_login_name, grants.privilege, grants.resource")
	query.B("FROM grants")
	query.B("JOIN groups ON grants.subject_id = groups.id")
	query.B("AND grants.subject_kind = ?", models.SubjectKindGroup)
	query.B("JOIN identities_groups ON identities_groups.group_id = groups.id")
	query.B("JOIN identities ON identities_groups.identity_id = identities.id")
	query.B("WHERE grants.deleted_at is null")
	grantsByDestination(query, destination)
	query.B("AND grants.organization_id = ?", tx.OrganizationID())

	rows, err := tx.Query(query.String(), query.Args...)
	if err != nil {
		return nil, err
	}
	userAccess, err := scanRows(rows, func(a *DestinationAccess) []any {
		return []any{&a.UserID, &a.UserSSHLoginName, &a.Privilege, &a.Resource}
	})
	if err != nil {
		return nil, err
	}

	// Sort after the query to avoid creating a temporary DB table
	sort.Slice(userAccess, func(i, j int) bool {
		if userAccess[i].UserID == userAccess[j].UserID {
			return userAccess[i].Privilege < userAccess[j].Privilege
		}
		return userAccess[i].UserID < userAccess[j].UserID
	})
	return userAccess, nil
}

// DestinationAccessMaxUpdateIndex returns the maximum update_index from all
// the grants and groups that match the query.
// This MUST include soft-deleted rows as well.
//
// Returns 1 if no records match the query, so that the caller can block until
// a record exists.
//
// TODO: any way to assert this tx has the right isolation level?
func DestinationAccessMaxUpdateIndex(tx ReadTxn, destination string) (int64, error) {
	// IMPORTANT: changes to this query likely also need to be applied to ListDestinationAccess
	query := querybuilder.New("SELECT max(idx) FROM (")
	query.B("SELECT max(grants.update_index) as idx FROM grants")
	query.B("WHERE grants.organization_id = ?", tx.OrganizationID())
	grantsByDestination(query, destination)
	query.B("UNION ALL SELECT")
	query.B("max(groups.membership_update_index) as idx")
	query.B("FROM grants")
	query.B("JOIN groups ON grants.subject_id = groups.id")
	query.B("AND grants.subject_kind = ?", models.SubjectKindGroup)
	query.B("AND grants.organization_id = ?", tx.OrganizationID())
	grantsByDestination(query, destination)
	query.B(") as subq")

	var result *int64
	err := tx.QueryRow(query.String(), query.Args...).Scan(&result)
	if err != nil || result == nil {
		return 1, err
	}
	return *result, err
}

package data

import (
	"fmt"
	"time"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type destinationsTable models.Destination

func (d destinationsTable) Table() string {
	return "destinations"
}

func (d destinationsTable) Columns() []string {
	return []string{"connection_ca", "connection_url", "created_at", "deleted_at", "id", "last_seen_at", "name", "organization_id", "resources", "roles", "unique_id", "updated_at", "version"}
}

func (d destinationsTable) Values() []any {
	return []any{d.ConnectionCA, d.ConnectionURL, d.CreatedAt, d.DeletedAt, d.ID, d.LastSeenAt, d.Name, d.OrganizationID, d.Resources, d.Roles, d.UniqueID, d.UpdatedAt, d.Version}
}

func (d *destinationsTable) ScanFields() []any {
	return []any{&d.ConnectionCA, &d.ConnectionURL, &d.CreatedAt, &d.DeletedAt, &d.ID, &d.LastSeenAt, &d.Name, &d.OrganizationID, &d.Resources, &d.Roles, &d.UniqueID, &d.UpdatedAt, &d.Version}
}

// destinationsUpdateTable is used to update the destination. It excludes
// the CreatedAt field, because that field is not part of the input to
// UpdateDestination.
type destinationsUpdateTable models.Destination

func (d destinationsUpdateTable) Table() string {
	return "destinations"
}

func (d destinationsUpdateTable) Columns() []string {
	return []string{"connection_ca", "connection_url", "deleted_at", "id", "last_seen_at", "name", "organization_id", "resources", "roles", "unique_id", "updated_at", "version"}
}

func (d destinationsUpdateTable) Values() []any {
	return []any{d.ConnectionCA, d.ConnectionURL, d.DeletedAt, d.ID, d.LastSeenAt, d.Name, d.OrganizationID, d.Resources, d.Roles, d.UniqueID, d.UpdatedAt, d.Version}
}

func validateDestination(dest *models.Destination) error {
	if dest.Name == "" {
		return fmt.Errorf("Destination.Name is required")
	}
	if dest.UniqueID == "" {
		return fmt.Errorf("Destination.UniqueID is required")
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
	return update(tx, (*destinationsUpdateTable)(destination))
}

func GetDestination(db GormTxn, selectors ...SelectorFunc) (*models.Destination, error) {
	return get[models.Destination](db, selectors...)
}

func ListDestinations(db GormTxn, p *Pagination, selectors ...SelectorFunc) ([]models.Destination, error) {
	return list[models.Destination](db, p, selectors...)
}

func DeleteDestination(tx WriteTxn, id uid.ID) error {
	stmt := `
		UPDATE destinations SET deleted_at = ?
		WHERE id = ? AND organization_id = ? AND deleted_at is null
	`
	_, err := tx.Exec(stmt, time.Now(), id, tx.OrganizationID())
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
		SELECT COALESCE(version, '') as version,
			   last_seen_at >= ? as connected,
			   count(*)
		FROM destinations
		WHERE deleted_at IS NULL
		GROUP BY connected, version
	`
	rows, err := tx.Query(stmt, timeout)
	if err != nil {
		return nil, err

	}
	defer rows.Close()

	var result []DestinationsCount
	for rows.Next() {
		var item DestinationsCount
		if err := rows.Scan(&item.Version, &item.Connected, &item.Count); err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	return result, rows.Err()
}

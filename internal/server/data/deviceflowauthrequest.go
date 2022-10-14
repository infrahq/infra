package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type deviceFlowAuthRequestTable models.DeviceFlowAuthRequest

func (d deviceFlowAuthRequestTable) Table() string {
	return "device_flow_auth_requests"
}

func (d deviceFlowAuthRequestTable) Columns() []string {
	return []string{"id", "client_id", "user_code", "device_code", "approved", "access_key_id", "expires_at", "created_at",
		"updated_at"}
}

func (d deviceFlowAuthRequestTable) Values() []any {
	return []any{d.ID, d.ClientID, d.UserCode, d.DeviceCode, d.Approved, d.AccessKeyID, d.ExpiresAt, d.CreatedAt, d.UpdatedAt}
}

func (d *deviceFlowAuthRequestTable) ScanFields() []any {
	return []any{d.ID, d.ClientID, d.UserCode, d.DeviceCode, d.Approved, d.AccessKeyID, d.ExpiresAt, d.CreatedAt, d.UpdatedAt}
}

// deviceFlowAuthRequestUpdateTable is used to update the DeviceFlowAuthRequest. It excludes
// the CreatedAt field, because that field is not part of the input to
// UpdateDeviceFlowAuthRequest.
type deviceFlowAuthRequestUpdateTable models.DeviceFlowAuthRequest

func (d deviceFlowAuthRequestUpdateTable) Table() string {
	return "device_flow_auth_requests"
}

func (d deviceFlowAuthRequestUpdateTable) Columns() []string {
	return []string{"id", "client_id", "user_code", "device_code", "approved", "access_key_id", "expires_at",
		"updated_at", "deleted_at"}
}

func (d deviceFlowAuthRequestUpdateTable) Values() []any {
	return []any{d.ID, d.ClientID, d.UserCode, d.DeviceCode, d.Approved, d.AccessKeyID, d.ExpiresAt, d.UpdatedAt, d.DeletedAt}
}

func validateDeviceFlowAuthRequest(dfar *models.DeviceFlowAuthRequest) error {
	return nil
}

func CreateDeviceFlowAuthRequest(tx WriteTxn, dfar *models.DeviceFlowAuthRequest) error {
	if err := validateDeviceFlowAuthRequest(dfar); err != nil {
		return err
	}
	return insert(tx, (*deviceFlowAuthRequestTable)(dfar))
}

func UpdateDeviceFlowAuthRequest(tx WriteTxn, dfar *models.DeviceFlowAuthRequest) error {
	if err := validateDeviceFlowAuthRequest(dfar); err != nil {
		return err
	}
	return update(tx, (*deviceFlowAuthRequestUpdateTable)(dfar))
}

func GetDeviceFlowAuthRequest(db GormTxn, selectors ...SelectorFunc) (*models.DeviceFlowAuthRequest, error) {
	return get[models.DeviceFlowAuthRequest](db, selectors...)
}

func SetDeficeFlowAuthRequestAccessKey(tx WriteTxn, dfarID uid.ID, accessKey *models.AccessKey) error {
	_, err := tx.Exec(`
		UPDATE device_flow_auth_requests
		SET 
			access_key_id = ?,
			access_key_token = ?,
			approved = true
		WHERE id = ?
	`, accessKey.ID, accessKey.Token(), dfarID)
	return handleError(err)
}

func ByClientIDAndDeviceCode(clientID, deviceCode string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("client_id = ? and device_code = ?", clientID, deviceCode)
	}
}

func ByUserCode(userCode string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("user_code = ?", userCode)
	}
}

// func ListDeviceFlowAuthRequests(db GormTxn, p *Pagination, selectors ...SelectorFunc) ([]models.DeviceFlowAuthRequest, error) {
// 	return list[models.DeviceFlowAuthRequest](db, p, selectors...)
// }

// func DeleteDeviceFlowAuthRequest(tx WriteTxn, id uid.ID) error {
// 	stmt := `
// 		UPDATE destinations SET deleted_at = ?
// 		WHERE id = ? AND organization_id = ? AND deleted_at is null
// 	`
// 	_, err := tx.Exec(stmt, time.Now(), id, tx.OrganizationID())
// 	return handleError(err)
// }

// type DeviceFlowAuthRequestsCount struct {
// 	Connected bool
// 	Version   string
// 	Count     float64
// }

// func CountDeviceFlowAuthRequestsByConnectedVersion(tx ReadTxn) ([]DeviceFlowAuthRequestsCount, error) {
// 	timeout := time.Now().Add(-5 * time.Minute)

// 	stmt := `
// 		SELECT COALESCE(version, '') as version,
// 			   last_seen_at >= ? as connected,
// 			   count(*)
// 		FROM destinations
// 		WHERE deleted_at IS NULL
// 		GROUP BY connected, version
// 	`
// 	rows, err := tx.Query(stmt, timeout)
// 	if err != nil {
// 		return nil, err

// 	}
// 	defer rows.Close()

// 	var result []DeviceFlowAuthRequestsCount
// 	for rows.Next() {
// 		var item DeviceFlowAuthRequestsCount
// 		if err := rows.Scan(&item.Version, &item.Connected, &item.Count); err != nil {
// 			return nil, err
// 		}
// 		result = append(result, item)
// 	}

// 	return result, rows.Err()
// }

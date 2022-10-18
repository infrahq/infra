package data

import (
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type deviceFlowAuthRequestTable models.DeviceFlowAuthRequest

func (d deviceFlowAuthRequestTable) Table() string {
	return "device_flow_auth_requests"
}

func (d deviceFlowAuthRequestTable) Columns() []string {
	return []string{"id", "user_code", "device_code", "approved", "access_key_id", "expires_at", "created_at",
		"updated_at"}
}

func (d deviceFlowAuthRequestTable) Values() []any {
	return []any{d.ID, d.UserCode, d.DeviceCode, d.Approved, d.AccessKeyID, d.ExpiresAt, d.CreatedAt, d.UpdatedAt}
}

func (d *deviceFlowAuthRequestTable) ScanFields() []any {
	return []any{d.ID, d.UserCode, d.DeviceCode, d.Approved, d.AccessKeyID, d.ExpiresAt, d.CreatedAt, d.UpdatedAt}
}

func validateDeviceFlowAuthRequest(dfar *models.DeviceFlowAuthRequest) error {
	err := validate.Error{}
	validationRules := []validate.ValidationRule{
		validate.String("user_code", dfar.UserCode, 8, 8, validate.DeviceFlowUserCode),
		validate.String("device_code", dfar.DeviceCode, 38, 38, validate.AlphaNumeric),
		validate.Required("expires_at", dfar.ExpiresAt),
		validate.Date("expires_at", dfar.ExpiresAt, time.Now().Add(-1*time.Second), time.Now().Add(30*time.Minute)), // must be short-lived
	}
	for _, rule := range validationRules {
		if failure := rule.Validate(); failure != nil {
			err[failure.Name] = append(err[failure.Name], failure.Problems...)
		}
	}

	if len(err) > 0 {
		return err
	}

	return nil
}

func CreateDeviceFlowAuthRequest(tx WriteTxn, dfar *models.DeviceFlowAuthRequest) error {
	if err := validateDeviceFlowAuthRequest(dfar); err != nil {
		return err
	}
	return insert(tx, (*deviceFlowAuthRequestTable)(dfar))
}

func GetDeviceFlowAuthRequest(db GormTxn, selectors ...SelectorFunc) (*models.DeviceFlowAuthRequest, error) {
	return get[models.DeviceFlowAuthRequest](db, selectors...)
}

func SetDeviceFlowAuthRequestAccessKey(tx WriteTxn, dfarID uid.ID, accessKey *models.AccessKey) error {
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

func ByDeviceCode(deviceCode string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("device_code = ?", deviceCode)
	}
}

func ByUserCode(userCode string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("user_code = ?", userCode)
	}
}

func DeleteExpiredDeviceFlowAuthRequest(tx WriteTxn) error {
	stmt := `
		DELETE from device_flow_auth_requests
		WHERE
			deleted_at IS NOT NULL
			OR expires_at < ?
	`
	_, err := tx.Exec(stmt, time.Now())
	return handleError(err)
}

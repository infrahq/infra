package data

import (
	"fmt"
	"time"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type deviceFlowAuthRequestTable models.DeviceFlowAuthRequest

func (d deviceFlowAuthRequestTable) Table() string {
	return "device_flow_auth_requests"
}

func (d deviceFlowAuthRequestTable) Columns() []string {
	return []string{"access_key_id", "access_key_token", "created_at", "deleted_at", "device_code", "expires_at", "id", "updated_at", "user_code"}
}

func (d deviceFlowAuthRequestTable) Values() []any {
	return []any{d.AccessKeyID, d.AccessKeyToken, d.CreatedAt, d.DeletedAt, d.DeviceCode, d.ExpiresAt, d.ID, d.UpdatedAt, d.UserCode}
}

func (d *deviceFlowAuthRequestTable) ScanFields() []any {
	return []any{&d.AccessKeyID, &d.AccessKeyToken, &d.CreatedAt, &d.DeletedAt, &d.DeviceCode, &d.ExpiresAt, &d.ID, &d.UpdatedAt, &d.UserCode}
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
		return fmt.Errorf("%w: %s", internal.ErrInternalServerError, err)
	}

	return nil
}

func CreateDeviceFlowAuthRequest(tx WriteTxn, dfar *models.DeviceFlowAuthRequest) error {
	if err := validateDeviceFlowAuthRequest(dfar); err != nil {
		return err
	}
	return insert(tx, (*deviceFlowAuthRequestTable)(dfar))
}

type SelectDeviceFlowAuthRequestOptions struct {
	ByID         uid.ID
	ByDeviceCode string
	ByUserCode   string
}

func GetDeviceFlowAuthRequest(tx GormTxn, opts SelectDeviceFlowAuthRequestOptions) (*models.DeviceFlowAuthRequest, error) {
	rec := &deviceFlowAuthRequestTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(rec))
	query.B("FROM")
	query.B(rec.Table())
	query.B("WHERE deleted_at is null")
	if opts.ByID != 0 {
		query.B("AND id = ?", opts.ByID)
	}
	if opts.ByDeviceCode != "" {
		query.B("and device_code = ?", opts.ByDeviceCode)
	}
	if opts.ByUserCode != "" {
		query.B("and user_code = ?", opts.ByUserCode)
	}

	err := tx.QueryRow(query.String(), query.Args...).Scan(rec.ScanFields()...)
	if err != nil {
		return nil, handleError(err)
	}
	return (*models.DeviceFlowAuthRequest)(rec), nil
}

func SetDeviceFlowAuthRequestAccessKey(tx WriteTxn, dfarID uid.ID, accessKey *models.AccessKey) error {
	_, err := tx.Exec(`
		UPDATE device_flow_auth_requests
		SET 
			access_key_id = ?,
			access_key_token = ?
		WHERE id = ?
	`, accessKey.ID, models.EncryptedAtRest(accessKey.Token()), dfarID)
	return handleError(err)
}

func DeleteExpiredDeviceFlowAuthRequests(tx WriteTxn) error {
	stmt := `
		DELETE from device_flow_auth_requests
		WHERE
			deleted_at IS NOT NULL
			OR expires_at < ?
	`
	_, err := tx.Exec(stmt, time.Now())
	return handleError(err)
}

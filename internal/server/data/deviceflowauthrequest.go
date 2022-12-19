package data

import (
	"errors"
	"time"

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
	return []string{"created_at", "deleted_at", "device_code", "expires_at", "id", "updated_at", "user_code", "user_id", "provider_id"}
}

func (d deviceFlowAuthRequestTable) Values() []any {
	return []any{d.CreatedAt, d.DeletedAt, d.DeviceCode, d.ExpiresAt, d.ID, d.UpdatedAt, d.UserCode, d.UserID, d.ProviderID}
}

func (d *deviceFlowAuthRequestTable) ScanFields() []any {
	return []any{&d.CreatedAt, &d.DeletedAt, &d.DeviceCode, &d.ExpiresAt, &d.ID, &d.UpdatedAt, &d.UserCode, &d.UserID, &d.ProviderID}
}

// TODO: use regular if conditions here. There's no benefit to using the validate functions.
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
		return errors.New(err.Error())
	}
	return nil
}

func CreateDeviceFlowAuthRequest(tx WriteTxn, dfar *models.DeviceFlowAuthRequest) error {
	if err := validateDeviceFlowAuthRequest(dfar); err != nil {
		return err
	}
	return insert(tx, (*deviceFlowAuthRequestTable)(dfar))
}

type GetDeviceFlowAuthRequestOptions struct {
	ByDeviceCode string
	ByUserCode   string
}

func GetDeviceFlowAuthRequest(tx ReadTxn, opts GetDeviceFlowAuthRequestOptions) (*models.DeviceFlowAuthRequest, error) {
	if opts.ByDeviceCode == "" && opts.ByUserCode == "" {
		return nil, errors.New("must supply device_code or user_code to GetDeviceFlowAuthRequest")
	}

	rec := &deviceFlowAuthRequestTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(rec))
	query.B("FROM")
	query.B(rec.Table())
	query.B("WHERE deleted_at is null")
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

func DeleteExpiredDeviceFlowAuthRequests(tx WriteTxn) error {
	_, err := tx.Exec(`DELETE FROM device_flow_auth_requests where expires_at <= ?`, time.Now().UTC())
	return handleError(err)
}

func DeleteDeviceFlowAuthRequest(tx WriteTxn, dfarID uid.ID) error {
	query := querybuilder.New("UPDATE device_flow_auth_requests")
	query.B("SET deleted_at = ?", time.Now().UTC())
	query.B("WHERE id = ?", dfarID)

	_, err := tx.Exec(query.String(), query.Args...)
	return handleError(err)
}

func ApproveDeviceFlowAuthRequest(tx WriteTxn, dfarID uid.ID, userID uid.ID, providerID uid.ID) error {
	query := querybuilder.New("UPDATE device_flow_auth_requests")
	query.B("SET user_id = ?, provider_id = ?", userID, providerID)
	query.B("WHERE id = ?", dfarID)

	_, err := tx.Exec(query.String(), query.Args...)
	return handleError(err)
}

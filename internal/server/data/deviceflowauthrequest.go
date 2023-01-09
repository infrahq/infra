package data

import (
	"errors"
	"fmt"
	"time"

	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
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

func validateDeviceFlowAuthRequest(dfar *models.DeviceFlowAuthRequest) error {
	switch {
	case len(dfar.UserCode) != 8:
		return fmt.Errorf("a user code with length 8 is required")
	case len(dfar.DeviceCode) != 38:
		return fmt.Errorf("a device code with legnth 38 is required")
	case dfar.ExpiresAt.IsZero():
		return fmt.Errorf("an expiry is required")
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

package data

import (
	"time"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func GetCredentialRequest(tx ReadTxn, id, organizationID uid.ID) (*models.CredentialRequest, error) {
	query := querybuilder.New("SELECT")
	query.B("id, organization_id, expires_at, update_index, user_id, destination_id, answered, bearer_token")
	query.B("FROM credential_requests")
	query.B("WHERE organization_id = ?", organizationID)
	query.B("AND id = ?", id)

	r := &models.CredentialRequest{}
	err := tx.QueryRow(query.String(), query.Args...).Scan(&r.ID, &r.OrganizationID, &r.ExpiresAt, &r.UpdateIndex, &r.UserID, &r.DestinationID, &r.Answered, &r.BearerToken)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func CreateCredentialRequest(tx WriteTxn, cr *models.CredentialRequest) error {
	q := querybuilder.New("INSERT INTO credential_requests")
	q.B("(id, organization_id, expires_at, user_id, destination_id, update_index)")
	q.B("VALUES (?,?,?,?,?,nextval('seq_update_index'))", cr.ID, cr.OrganizationID, cr.ExpiresAt, cr.UserID, cr.DestinationID)

	logging.Debugf("Should be triggering notification for credreq_%d_%d", cr.OrganizationID, cr.DestinationID)
	_, err := tx.Exec(q.String(), q.Args...) // will trigger credential_request_insert_notify()
	if err != nil {
		return handleError(err)
	}

	return nil
}

func UpdateCredentialRequest(tx WriteTxn, cr *models.CredentialRequest) error {
	q := querybuilder.New("UPDATE credential_requests")
	q.B("SET")
	q.B("answered = ?,", true)
	q.B("bearer_token = ?,", cr.BearerToken)
	q.B("expires_at = ?", cr.ExpiresAt)
	q.B("WHERE id = ?", cr.ID)
	q.B("AND organization_id = ?", cr.OrganizationID)

	_, err := tx.Exec(q.String(), q.Args...)
	if err != nil {
		return handleError(err)
	}

	return nil
}

func ListCredentialRequests(tx ReadTxn, destinationID uid.ID) ([]models.CredentialRequest, error) {
	query := querybuilder.New("SELECT")
	query.B("id, organization_id, expires_at, update_index, user_id, destination_id")
	query.B("FROM credential_requests")
	query.B("WHERE organization_id = ?", tx.OrganizationID())
	query.B("AND destination_id = ?", destinationID)
	query.B("AND expires_at >= ?", time.Now())
	query.B("AND answered = ?", false)

	rows, err := tx.Query(query.String(), query.Args...)
	if err != nil {
		return nil, err
	}
	return scanRows(rows, func(r *models.CredentialRequest) []any {
		return []any{&r.ID, &r.OrganizationID, &r.ExpiresAt, &r.UpdateIndex, &r.UserID, &r.DestinationID}
	})
}

func CredentialRequestsMaxUpdateIndex(tx ReadTxn, destinationID uid.ID) (int64, error) {
	query := querybuilder.New("SELECT max(update_index) FROM credential_requests")
	query.B("WHERE organization_id = ?", tx.OrganizationID())
	query.B("AND destination_id = ?", destinationID)

	var result *int64
	err := tx.QueryRow(query.String(), query.Args...).Scan(&result)
	if err != nil || result == nil {
		return 1, err
	}
	return *result, err
}

func RemoveExpiredCredentialRequests(tx WriteTxn) error {
	_, err := tx.Exec("DELETE FROM credential_requests WHERE expires_at < ?", time.Now())
	if err != nil {
		return err
	}
	return nil
}

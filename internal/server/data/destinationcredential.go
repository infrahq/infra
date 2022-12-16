package data

import (
	"time"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func GetDestinationCredential(tx ReadTxn, id, organizationID uid.ID) (*models.DestinationCredential, error) {
	query := querybuilder.New("SELECT")
	query.B("id, organization_id, request_expires_at, update_index, user_id, destination_id, answered, credential_expires_at, bearer_token")
	query.B("FROM destination_credentials")
	query.B("WHERE organization_id = ?", organizationID)
	query.B("AND id = ?", id)

	r := &models.DestinationCredential{}
	err := tx.QueryRow(query.String(), query.Args...).Scan(&r.ID, &r.OrganizationID, &r.RequestExpiresAt, &r.UpdateIndex, &r.UserID, &r.DestinationID, &r.Answered, &r.CredentialExpiresAt, &r.BearerToken)
	if err != nil {
		return nil, handleError(err)
	}
	return r, nil
}

func CreateDestinationCredential(tx WriteTxn, cr *models.DestinationCredential) error {
	q := querybuilder.New("INSERT INTO destination_credentials")
	q.B("(id, organization_id, request_expires_at, user_id, destination_id, update_index)")
	q.B("VALUES (?,?,?,?,?,nextval('seq_update_index'))", cr.ID, cr.OrganizationID, cr.RequestExpiresAt, cr.UserID, cr.DestinationID)

	logging.Debugf("Posting credential notification for credreq_%s_%s", cr.OrganizationID.String(), cr.DestinationID.String())
	_, err := tx.Exec(q.String(), q.Args...) // will trigger destination_credential_insert_notify()
	if err != nil {
		return handleError(err)
	}

	return nil
}

func AnswerDestinationCredential(tx WriteTxn, cr *models.DestinationCredential) error {
	q := querybuilder.New("UPDATE destination_credentials")
	q.B("SET")
	q.B("answered = ?,", true)
	q.B("bearer_token = ?,", cr.BearerToken)
	q.B("credential_expires_at = ?", cr.CredentialExpiresAt)
	q.B("WHERE id = ?", cr.ID)
	q.B("AND organization_id = ?", cr.OrganizationID)

	logging.Debugf("Posting credential notification for cred_ans_%s_%s", cr.OrganizationID.String(), cr.ID.String())
	_, err := tx.Exec(q.String(), q.Args...) // will trigger destination_credential_update_notify()
	if err != nil {
		return handleError(err)
	}

	return nil
}

func ListDestinationCredentials(tx ReadTxn, destinationID uid.ID) ([]models.DestinationCredential, error) {
	query := querybuilder.New("SELECT")
	query.B("id, organization_id, request_expires_at, update_index, user_id, destination_id")
	query.B("FROM destination_credentials")
	query.B("WHERE organization_id = ?", tx.OrganizationID())
	query.B("AND destination_id = ?", destinationID)
	query.B("AND request_expires_at >= ?", time.Now())
	query.B("AND answered = ?", false)

	rows, err := tx.Query(query.String(), query.Args...)
	if err != nil {
		return nil, err
	}
	return scanRows(rows, func(r *models.DestinationCredential) []any {
		return []any{&r.ID, &r.OrganizationID, &r.RequestExpiresAt, &r.UpdateIndex, &r.UserID, &r.DestinationID}
	})
}

func DestinationCredentialsMaxUpdateIndex(tx ReadTxn, destinationID uid.ID) (int64, error) {
	query := querybuilder.New("SELECT max(update_index) FROM destination_credentials")
	query.B("WHERE organization_id = ?", tx.OrganizationID())
	query.B("AND destination_id = ?", destinationID)

	var result *int64
	err := tx.QueryRow(query.String(), query.Args...).Scan(&result)
	if err != nil || result == nil {
		return 1, err
	}
	return *result, err
}

func RemoveExpiredDestinationCredentials(tx WriteTxn) error {
	_, err := tx.Exec("DELETE FROM destination_credentials WHERE request_expires_at < ?", time.Now())
	if err != nil {
		return err
	}
	return nil
}

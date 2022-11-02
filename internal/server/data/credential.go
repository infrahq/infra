package data

import (
	"fmt"
	"time"

	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type credentialsTable models.Credential

func (credentialsTable) Table() string {
	return "credentials"
}

func (c credentialsTable) Columns() []string {
	return []string{"created_at", "deleted_at", "id", "identity_id", "one_time_password", "organization_id", "password_hash", "updated_at"}
}

func (c credentialsTable) Values() []any {
	return []any{c.CreatedAt, c.DeletedAt, c.ID, c.IdentityID, c.OneTimePassword, c.OrganizationID, c.PasswordHash, c.UpdatedAt}
}

func (c *credentialsTable) ScanFields() []any {
	return []any{&c.CreatedAt, &c.DeletedAt, &c.ID, &c.IdentityID, &c.OneTimePassword, &c.OrganizationID, &c.PasswordHash, &c.UpdatedAt}
}

func validateCredential(c *models.Credential) error {
	switch {
	case len(c.PasswordHash) == 0:
		return fmt.Errorf("Credential.PasswordHash is required")
	case c.IdentityID == 0:
		return fmt.Errorf("Credential.IdentityID is required")
	}
	return nil
}

func CreateCredential(db GormTxn, credential *models.Credential) error {
	if err := validateCredential(credential); err != nil {
		return err
	}
	return add(db, credential)
}

func SaveCredential(db GormTxn, credential *models.Credential) error {
	if err := validateCredential(credential); err != nil {
		return err
	}
	return save(db, credential)
}

func GetCredentialByUserID(tx ReadTxn, userID uid.ID) (*models.Credential, error) {
	if userID == 0 {
		return nil, fmt.Errorf("a userID is required to get credential")
	}

	credential := credentialsTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(credential))
	query.B("FROM credentials")
	query.B("WHERE deleted_at is NULL")
	query.B("AND organization_id = ?", tx.OrganizationID())
	query.B("AND identity_id = ?", userID)

	err := tx.QueryRow(query.String(), query.Args...).Scan(credential.ScanFields()...)
	if err != nil {
		return nil, handleError(err)
	}
	return (*models.Credential)(&credential), nil
}

func DeleteCredential(tx WriteTxn, id uid.ID) error {
	stmt := `
		UPDATE credentials
		SET deleted_at = ?
		WHERE id = ? AND organization_id = ? AND deleted_at is NULL`

	_, err := tx.Exec(stmt, time.Now(), id, tx.OrganizationID())
	return err
}

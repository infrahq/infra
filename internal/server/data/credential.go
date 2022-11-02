package data

import (
	"fmt"
	"time"

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

func GetCredential(db GormTxn, selectors ...SelectorFunc) (*models.Credential, error) {
	return get[models.Credential](db, selectors...)
}

func DeleteCredential(tx WriteTxn, id uid.ID) error {
	stmt := `
		UPDATE credentials
		SET deleted_at = ?
		WHERE id = ? AND organization_id = ? AND deleted_at is NULL`

	_, err := tx.Exec(stmt, time.Now(), id, tx.OrganizationID())
	return err
}

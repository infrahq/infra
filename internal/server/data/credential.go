package data

import (
	"fmt"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

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

func DeleteCredential(db GormTxn, id uid.ID) error {
	return delete[models.Credential](db, id)
}

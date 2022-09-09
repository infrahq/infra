package data

import (
	"fmt"
	mathrand "math/rand"

	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
)

type encryptionKeysTable models.EncryptionKey

func (e encryptionKeysTable) Table() string {
	return "encryption_keys"
}

func (e encryptionKeysTable) Columns() []string {
	return []string{"algorithm", "created_at", "deleted_at", "encrypted", "id", "key_id", "name", "root_key_id", "updated_at"}
}

func (e encryptionKeysTable) Values() []any {
	return []any{e.Algorithm, e.CreatedAt, e.DeletedAt, e.Encrypted, e.ID, e.KeyID, e.Name, e.RootKeyID, e.UpdatedAt}
}

func (e *encryptionKeysTable) ScanFields() []any {
	return []any{&e.Algorithm, &e.CreatedAt, &e.DeletedAt, &e.Encrypted, &e.ID, &e.KeyID, &e.Name, &e.RootKeyID, &e.UpdatedAt}
}

func CreateEncryptionKey(tx WriteTxn, key *models.EncryptionKey) error {
	switch {
	case key.Name == "":
		return fmt.Errorf("a name is required for EncryptionKey")
	case key.RootKeyID == "":
		return fmt.Errorf("a root key ID is required for EncryptionKey")
	case key.Algorithm == "":
		return fmt.Errorf("an algorithm is required for EncryptionKey")
	}
	if key.KeyID == 0 {
		// not a security issue; just an identifier
		key.KeyID = mathrand.Int31() // nolint:gosec
	}

	return insert(tx, (*encryptionKeysTable)(key))
}

func GetEncryptionKeyByName(tx ReadTxn, name string) (*models.EncryptionKey, error) {
	table := &encryptionKeysTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(table))
	query.B("FROM encryption_keys")
	query.B("WHERE deleted_at is null")
	query.B("AND name = ?", name)

	row := tx.QueryRow(query.String(), query.Args...)
	if err := row.Scan(table.ScanFields()...); err != nil {
		return nil, handleReadError(err)
	}
	return (*models.EncryptionKey)(table), nil
}

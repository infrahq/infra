package data

import (
	"errors"
	"fmt"
	mathrand "math/rand"

	"github.com/infrahq/secrets"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data/migrator"
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

// StdlibTxn is a transaction that uses only the methods from the database/sql
// package. It excludes the metadata that exists on the ReadTxn and WriteTxn.
// StdlibTxn is for tables that are not scoped to an org.
type StdlibTxn migrator.DB

func CreateEncryptionKey(tx StdlibTxn, key *models.EncryptionKey) error {
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

	item := (*encryptionKeysTable)(key)
	if err := item.OnInsert(); err != nil {
		return err
	}
	query := querybuilder.New("INSERT INTO encryption_keys (")
	query.B(columnsForInsert(item))
	query.B(") VALUES (")
	query.B(placeholderForColumns(item), item.Values()...)
	query.B(");")
	_, err := tx.Exec(query.String(), query.Args...)
	return handleError(err)
}

func GetEncryptionKeyByName(tx StdlibTxn, name string) (*models.EncryptionKey, error) {
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

type EncryptionKeyProvider interface {
	GenerateDataKey(rootKeyID string) (*secrets.SymmetricKey, error)
	DecryptDataKey(rootKeyID string, keyData []byte) (*secrets.SymmetricKey, error)
}

var dbKeyName = "dbkey"

func loadDBKey(tx StdlibTxn, provider EncryptionKeyProvider, rootKeyId string) error {
	keyRec, err := GetEncryptionKeyByName(tx, dbKeyName)
	if err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			return createDBKey(tx, provider, rootKeyId)
		}
		return err
	}

	sKey, err := provider.DecryptDataKey(rootKeyId, keyRec.Encrypted)
	if err != nil {
		return err
	}

	models.SymmetricKey = sKey
	return nil
}

func createDBKey(tx StdlibTxn, provider EncryptionKeyProvider, rootKeyId string) error {
	sKey, err := provider.GenerateDataKey(rootKeyId)
	if err != nil {
		return err
	}

	key := &models.EncryptionKey{
		Name:      dbKeyName,
		Encrypted: sKey.Encrypted,
		Algorithm: sKey.Algorithm,
		RootKeyID: sKey.RootKeyID,
	}
	if err = CreateEncryptionKey(tx, key); err != nil {
		return err
	}

	models.SymmetricKey = sKey
	return nil
}

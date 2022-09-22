package data

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type accessKeyTable models.AccessKey

func (a accessKeyTable) Table() string {
	return "access_keys"
}

func (a accessKeyTable) Columns() []string {
	return []string{"created_at", "deleted_at", "expires_at", "extension", "extension_deadline", "id", "issued_for", "key_id", "name", "organization_id", "provider_id", "scopes", "secret_checksum", "updated_at"}
}

func (a accessKeyTable) Values() []any {
	return []any{a.CreatedAt, a.DeletedAt, a.ExpiresAt, a.Extension, a.ExtensionDeadline, a.ID, a.IssuedFor, a.KeyID, a.Name, a.OrganizationID, a.ProviderID, a.Scopes, a.SecretChecksum, a.UpdatedAt}
}

func (a *accessKeyTable) ScanFields() []any {
	return []any{&a.CreatedAt, &a.DeletedAt, &a.ExpiresAt, &a.Extension, &a.ExtensionDeadline, &a.ID, &a.IssuedFor, &a.KeyID, &a.Name, &a.OrganizationID, &a.ProviderID, &a.Scopes, &a.SecretChecksum, &a.UpdatedAt}
}

var (
	ErrAccessKeyExpired          = fmt.Errorf("access key expired")
	ErrAccessKeyDeadlineExceeded = fmt.Errorf("%w: extension deadline exceeded", ErrAccessKeyExpired)
)

func secretChecksum(secret string) []byte {
	chksm := sha256.Sum256([]byte(secret))
	return chksm[:]
}

func validateAccessKey(accessKey *models.AccessKey) error {
	switch {
	case accessKey.IssuedFor == 0:
		return fmt.Errorf("issusedFor is required")
	case accessKey.ProviderID == 0:
		return fmt.Errorf("providerID is required")
	case len(accessKey.KeyID) != models.AccessKeyKeyLength:
		return fmt.Errorf("invalid key length")
	}
	return nil
}

func CreateAccessKey(db GormTxn, accessKey *models.AccessKey) (body string, err error) {
	if accessKey.KeyID == "" {
		accessKey.KeyID = generate.MathRandom(models.AccessKeyKeyLength, generate.CharsetAlphaNumeric)
	}

	if accessKey.Secret == "" {
		secret, err := generate.CryptoRandom(models.AccessKeySecretLength, generate.CharsetAlphaNumeric)
		if err != nil {
			return "", err
		}

		accessKey.Secret = secret
	}
	if len(accessKey.Secret) != models.AccessKeySecretLength {
		return "", fmt.Errorf("invalid secret length")
	}

	accessKey.SecretChecksum = secretChecksum(accessKey.Secret)

	if accessKey.ExpiresAt.IsZero() {
		accessKey.ExpiresAt = time.Now().Add(time.Hour * 12).UTC()
	}

	if accessKey.Name == "" {
		// set a default name for look-up and CLI usage
		if accessKey.ID == 0 {
			accessKey.ID = uid.New()
		}

		identityIssuedFor, err := GetIdentity(db, ByID(accessKey.IssuedFor))
		if err != nil {
			return "", fmt.Errorf("key name from identity: %w", err)
		}

		accessKey.Name = fmt.Sprintf("%s-%s", identityIssuedFor.Name, accessKey.ID.String())
	}

	if err := validateAccessKey(accessKey); err != nil {
		return "", err
	}

	if err := insert(db, (*accessKeyTable)(accessKey)); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s", accessKey.KeyID, accessKey.Secret), nil
}

func UpdateAccessKey(tx WriteTxn, key *models.AccessKey) error {
	if key.Secret != "" {
		key.SecretChecksum = secretChecksum(key.Secret)
	}
	if err := validateAccessKey(key); err != nil {
		return err
	}

	return update(tx, (*accessKeyTable)(key))
}

type ListAccessKeyOptions struct {
	IncludeExpired bool
	ByIssuedForID  uid.ID
	ByName         string
	Pagination     *Pagination
}

func ListAccessKeys(tx ReadTxn, opts ListAccessKeyOptions) ([]models.AccessKey, error) {
	table := &accessKeyTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(table))
	query.B(", identities.name")
	if opts.Pagination != nil {
		query.B(", count(*) OVER()")
	}
	query.B("FROM access_keys INNER JOIN identities")
	query.B("ON access_keys.issued_for = identities.id")
	query.B("WHERE access_keys.deleted_at is null AND identities.deleted_at is null")
	query.B("AND access_keys.organization_id = ?", tx.OrganizationID())

	if !opts.IncludeExpired {
		// TODO: can we remove the need to check for both the zero value and nil?
		now, zero := time.Now(), time.Time{}
		query.B("AND (expires_at > ? OR expires_at = ? OR expires_at is null)", now, zero)
		query.B("AND (extension_deadline > ? OR extension_deadline = ? OR extension_deadline is null)", now, zero)
	}
	if opts.ByIssuedForID != 0 {
		query.B("AND issued_for = ?", opts.ByIssuedForID)
	}
	if opts.ByName != "" {
		query.B("AND access_keys.name = ?", opts.ByName)
	}
	query.B("ORDER BY access_keys.name ASC")
	if opts.Pagination != nil {
		opts.Pagination.PaginateQuery(query)
	}

	rows, err := tx.Query(query.String(), query.Args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.AccessKey
	for rows.Next() {
		var key models.AccessKey

		fields := append((*accessKeyTable)(&key).ScanFields(), &key.IssuedForName)
		if opts.Pagination != nil {
			fields = append(fields, &opts.Pagination.TotalCount)
		}
		if err := rows.Scan(fields...); err != nil {
			return nil, err
		}
		result = append(result, key)
	}
	return result, rows.Err()
}

type GetAccessKeysOptions struct {
	ByID    uid.ID
	ByKeyID string
}

// GetAccessKey using the keyID. Note that the keyID is globally unique, so
// this query is not scoped by an organization_id.
func GetAccessKey(tx ReadTxn, opts GetAccessKeysOptions) (*models.AccessKey, error) {
	if opts.ByID == 0 && len(opts.ByKeyID) == 0 {
		return nil, fmt.Errorf("GetAccessKey must supply either id or key_id")
	}
	accessKey := &accessKeyTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(accessKey))
	query.B("FROM")
	query.B(accessKey.Table())
	query.B("WHERE deleted_at is null")
	if len(opts.ByKeyID) > 0 {
		query.B("AND key_id = ?", opts.ByKeyID)
	}
	if opts.ByID > 0 {
		query.B("and id = ?", opts.ByID)
	}

	err := tx.QueryRow(query.String(), query.Args...).Scan(accessKey.ScanFields()...)
	if err != nil {
		return nil, handleReadError(err)
	}
	return (*models.AccessKey)(accessKey), nil
}

type DeleteAccessKeysOptions struct {
	// ByID instructs DeleteAccessKeys to delete the key with this ID.
	ByID uid.ID
	// ByIssuedForID instructs DeleteAccessKeys to delete keys issued for this user.
	ByIssuedForID uid.ID
	// ByProviderID instructs DeleteAccessKeys to delete keys issued by this
	// provider.
	ByProviderID uid.ID
}

func DeleteAccessKeys(tx WriteTxn, opts DeleteAccessKeysOptions) error {
	query := querybuilder.New("UPDATE access_keys")
	query.B("SET deleted_at = ? WHERE", time.Now())
	switch {
	case opts.ByID != 0:
		query.B("id = ?", opts.ByID)
	case opts.ByIssuedForID != 0:
		query.B("issued_for = ?", opts.ByIssuedForID)
	case opts.ByProviderID != 0:
		query.B("provider_id = ?", opts.ByProviderID)
	default:
		return fmt.Errorf("DeleteAccessKeys requires an ID to delete")
	}
	query.B("AND organization_id = ?", tx.OrganizationID())

	_, err := tx.Exec(query.String(), query.Args...)
	return err
}

// TODO: move this to access package?
func ValidateRequestAccessKey(tx WriteTxn, authnKey string) (*models.AccessKey, error) {
	keyID, secret, ok := strings.Cut(authnKey, ".")
	if !ok {
		return nil, fmt.Errorf("invalid access key format")
	}

	t, err := GetAccessKey(tx, GetAccessKeysOptions{ByKeyID: keyID})
	if err != nil {
		return nil, fmt.Errorf("%w: could not get access key from database, it may not exist", err)
	}

	sum := secretChecksum(secret)

	if subtle.ConstantTimeCompare(t.SecretChecksum, sum) != 1 {
		return nil, fmt.Errorf("access key invalid secret")
	}

	if time.Now().UTC().After(t.ExpiresAt) {
		return nil, ErrAccessKeyExpired
	}

	if !t.ExtensionDeadline.IsZero() {
		if time.Now().UTC().After(t.ExtensionDeadline) {
			return nil, ErrAccessKeyDeadlineExceeded
		}

		t.ExtensionDeadline = time.Now().UTC().Add(t.Extension)
		if err := UpdateAccessKey(tx, t); err != nil {
			return nil, err
		}
	}

	return t, nil
}

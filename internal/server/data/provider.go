package data

import (
	"errors"
	"fmt"
	"time"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type providersTable models.Provider

func (p providersTable) Table() string {
	return "providers"
}

func (p providersTable) Columns() []string {
	return []string{"auth_url", "client_email", "client_id", "client_secret", "created_at", "created_by", "deleted_at", "domain_admin_email", "id", "kind", "name", "organization_id", "private_key", "scopes", "updated_at", "url"}
}

func (p providersTable) Values() []any {
	return []any{p.AuthURL, p.ClientEmail, p.ClientID, p.ClientSecret, p.CreatedAt, p.CreatedBy, p.DeletedAt, p.DomainAdminEmail, p.ID, p.Kind, p.Name, p.OrganizationID, p.PrivateKey, p.Scopes, p.UpdatedAt, p.URL}
}

func (p *providersTable) ScanFields() []any {
	return []any{&p.AuthURL, &p.ClientEmail, &p.ClientID, &p.ClientSecret, &p.CreatedAt, &p.CreatedBy, &p.DeletedAt, &p.DomainAdminEmail, &p.ID, &p.Kind, &p.Name, &p.OrganizationID, &p.PrivateKey, &p.Scopes, &p.UpdatedAt, &p.URL}
}

func validateProvider(p *models.Provider) error {
	switch {
	case p.Name == "":
		return fmt.Errorf("name is required")
	case p.Kind == "":
		return fmt.Errorf("kind is required")
	default:
		return nil
	}
}

func CreateProvider(tx WriteTxn, provider *models.Provider) error {
	if err := validateProvider(provider); err != nil {
		return err
	}
	return insert(tx, (*providersTable)(provider))
}

type GetProviderOptions struct {
	ByID   uid.ID
	ByName string

	// KindInfra instructs GetProvider to return the infra provider. There should
	// only ever be a single provider with this kind for each org.
	KindInfra bool
}

func GetProvider(tx ReadTxn, opts GetProviderOptions) (*models.Provider, error) {
	provider := &providersTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(provider))
	query.B("FROM providers")
	query.B("WHERE deleted_at is null")
	query.B("AND organization_id = ?", tx.OrganizationID())

	switch {
	case opts.ByID != 0:
		query.B("AND id = ?", opts.ByID)
	case opts.ByName != "":
		query.B("AND name = ?", opts.ByName)
	case opts.KindInfra:
		query.B("AND kind = ?", models.ProviderKindInfra)
	default:
		return nil, fmt.Errorf("an ID is required to get provider")
	}

	err := tx.QueryRow(query.String(), query.Args...).Scan(provider.ScanFields()...)
	if err != nil {
		return nil, handleError(err)
	}
	return (*models.Provider)(provider), nil
}

type ListProvidersOptions struct {
	ByName               string
	ExcludeInfraProvider bool
	ByIDs                []uid.ID

	// CreatedBy instructs DeleteProviders to delete all the providers that were
	// created by this user. Can be used with NotIDs.
	CreatedBy uid.ID
	// NotIDs instructs DeleteProviders to exclude any providers with these IDs to
	// be excluded. In other words, these IDs will not be deleted, even if they
	// match CreatedBy.
	// Can only be used with CreatedBy.
	NotIDs []uid.ID

	Pagination *Pagination
}

func ListProviders(tx ReadTxn, opts ListProvidersOptions) ([]models.Provider, error) {
	table := providersTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(table))
	if opts.Pagination != nil {
		query.B(", count(*) OVER()")
	}
	query.B("FROM providers")
	query.B("WHERE deleted_at is null")
	query.B("AND organization_id = ?", tx.OrganizationID())

	if opts.ByName != "" {
		query.B("AND name = ?", opts.ByName)
	}
	if opts.ExcludeInfraProvider {
		query.B("AND kind <> ?", models.ProviderKindInfra)
	}
	if len(opts.ByIDs) > 0 {
		query.B("AND id IN")
		queryInClause(query, opts.ByIDs)
	}
	if opts.CreatedBy != 0 {
		query.B("AND created_by = ?", opts.CreatedBy)
		if len(opts.NotIDs) > 0 {
			query.B("AND id NOT IN")
			queryInClause(query, opts.NotIDs)
		}
	}

	query.B("ORDER BY name ASC")
	if opts.Pagination != nil {
		opts.Pagination.PaginateQuery(query)
	}

	rows, err := tx.Query(query.String(), query.Args...)
	if err != nil {
		return nil, err
	}
	return scanRows(rows, func(grant *models.Provider) []any {
		fields := (*providersTable)(grant).ScanFields()
		if opts.Pagination != nil {
			fields = append(fields, &opts.Pagination.TotalCount)
		}
		return fields
	})
}

func UpdateProvider(tx WriteTxn, provider *models.Provider) error {
	if err := validateProvider(provider); err != nil {
		return err
	}
	return update(tx, (*providersTable)(provider))
}

type DeleteProvidersOptions struct {
	// ByID instructs DeleteProviders to delete the provider matching this ID.
	// When non-zero all other fields on this struct will be ignored
	ByID uid.ID

	// CreatedBy instructs DeleteProviders to delete all the providers that were
	// created by this user. Can be used with NotIDs.
	CreatedBy uid.ID
	// NotIDs instructs DeleteProviders to exclude any providers with these IDs to
	// be excluded. In other words, these IDs will not be deleted, even if they
	// match CreatedBy.
	// Can only be used with CreatedBy.
	NotIDs []uid.ID
}

func DeleteProviders(db WriteTxn, opts DeleteProvidersOptions) error {
	var toDelete []models.Provider
	if opts.ByID != 0 {
		if provider, _ := GetProvider(db, GetProviderOptions{ByID: opts.ByID}); provider != nil {
			toDelete = []models.Provider{*provider}
		}
	} else {
		var err error
		toDelete, err = ListProviders(db, ListProvidersOptions{
			CreatedBy: opts.CreatedBy,
			NotIDs:    opts.NotIDs,
		})
		if err != nil {
			return fmt.Errorf("listing providers: %w", err)
		}
	}

	ids := make([]uid.ID, 0)
	for _, p := range toDelete {
		ids = append(ids, p.ID)

		providerUsers, err := ListProviderUsers(db, ListProviderUsersOptions{ByProviderID: p.ID})
		if err != nil {
			return fmt.Errorf("listing provider users: %w", err)
		}

		// if a user has no other providers, we need to remove the user.
		userIDsToDelete := []uid.ID{}
		for _, providerUser := range providerUsers {
			user, err := GetIdentity(db, GetIdentityOptions{ByID: providerUser.IdentityID, LoadProviders: true})
			if err != nil {
				if errors.Is(err, internal.ErrNotFound) {
					continue
				}
				return fmt.Errorf("get user: %w", err)
			}

			if len(user.Providers) == 1 && user.Providers[0].ID == p.ID {
				userIDsToDelete = append(userIDsToDelete, user.ID)
			}
		}

		if len(userIDsToDelete) > 0 {
			opts := DeleteIdentitiesOptions{
				ByProviderID: p.ID,
				ByIDs:        userIDsToDelete,
			}
			if err := DeleteIdentities(db, opts); err != nil {
				return fmt.Errorf("delete users: %w", err)
			}
		}

		if err := DeleteProviderUsers(db, DeleteProviderUsersOptions{ByProviderID: p.ID}); err != nil {
			return fmt.Errorf("delete provider users: %w", err)
		}

		if err := DeleteAccessKeys(db, DeleteAccessKeysOptions{ByProviderID: p.ID}); err != nil {
			return fmt.Errorf("delete access keys: %w", err)
		}

		// delete any access keys used for SCIM
		if err := DeleteAccessKeys(db, DeleteAccessKeysOptions{ByIssuedForID: p.ID}); err != nil {
			return fmt.Errorf("delete provider access keys: %w", err)
		}
	}

	query := querybuilder.New(`UPDATE providers`)
	query.B(`SET deleted_at = ?`, time.Now())
	query.B(`WHERE deleted_at is null`)
	query.B(`AND organization_id = ?`, db.OrganizationID())
	query.B(`AND id IN`)
	queryInClause(query, ids)
	_, err := db.Exec(query.String(), query.Args...)
	return err
}

type providersCount struct {
	Kind  string
	Count float64
}

func CountProvidersByKind(tx ReadTxn) ([]providersCount, error) {
	rows, err := tx.Query(`
		SELECT kind, COUNT(*) AS count
		FROM providers
		WHERE kind <> 'infra'
		AND deleted_at IS NULL
		GROUP BY kind`)
	if err != nil {
		return nil, err
	}
	return scanRows(rows, func(item *providersCount) []any {
		return []any{&item.Kind, &item.Count}
	})
}

func CountAllProviders(tx ReadTxn) (int64, error) {
	return countRows(tx, providersTable{})
}

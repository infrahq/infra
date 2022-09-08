package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/uid"
)

type providerUserTable models.ProviderUser

func (p providerUserTable) Table() string {
	return "provider_users"
}

func (p providerUserTable) Columns() []string {
	return []string{"identity_id", "provider_id", "email", "last_update", "redirect_url", "access_token", "refresh_token", "expires_at", "given_name", "family_name", "active"}
}

func (p providerUserTable) Values() []any {
	return []any{p.IdentityID, p.ProviderID, p.Email, p.LastUpdate, p.RedirectURL, p.AccessToken, p.RefreshToken, p.ExpiresAt, p.GivenName, p.FamilyName, p.Active}
}

func (p *providerUserTable) ScanFields() []any {
	return []any{&p.IdentityID, &p.ProviderID, &p.Email, &p.LastUpdate, &p.RedirectURL, &p.AccessToken, &p.RefreshToken, &p.ExpiresAt, &p.GivenName, &p.FamilyName, &p.Active}
}

func (p *providerUserTable) OnInsert() error {
	p.LastUpdate = time.Now().UTC()
	return nil
}

func validateProviderUser(u *models.ProviderUser) error {
	switch {
	case u.ProviderID == 0:
		return fmt.Errorf("providerID is required")
	case u.IdentityID == 0:
		return fmt.Errorf("identityID is required")
	case u.Email == "":
		return fmt.Errorf("email is required")
	default:
		return nil
	}
}

func CreateProviderUser(db GormTxn, provider *models.Provider, ident *models.Identity) (*models.ProviderUser, error) {
	// check if we already track this provider user
	pu, err := GetProviderUser(db, provider.ID, ident.ID)
	if err != nil && !errors.Is(err, internal.ErrNotFound) {
		return nil, err
	}

	if pu != nil {
		return pu, nil
	}

	pu = &models.ProviderUser{
		ProviderID: provider.ID,
		IdentityID: ident.ID,
		Email:      ident.Name,
		LastUpdate: time.Now().UTC(),
		Active:     true,
	}
	if err := validateProviderUser(pu); err != nil {
		return nil, err
	}

	if err := insert(db, (*providerUserTable)(pu)); err != nil {
		return nil, err
	}
	return pu, nil
}

func UpdateProviderUser(tx WriteTxn, providerUser *models.ProviderUser) error {
	if err := validateProviderUser(providerUser); err != nil {
		return err
	}
	providerUser.LastUpdate = time.Now().UTC()

	pu := (*providerUserTable)(providerUser)
	query := querybuilder.New("UPDATE")
	query.B(pu.Table())
	query.B("SET")
	query.B(columnsForUpdate(pu), pu.Values()...)
	query.B("WHERE provider_id = ? AND identity_id = ?;", providerUser.ProviderID, providerUser.IdentityID)
	_, err := tx.Exec(query.String(), query.Args...)
	return handleError(err)
}

type ListProviderUsersOptions struct {
	ByProviderID   uid.ID
	ByIdentityID   uid.ID
	ByIdentityIDs  []uid.ID
	HideInactive   bool
	SCIMParameters *SCIMParameters
}

func ListProviderUsers(tx ReadTxn, opts ListProviderUsersOptions) ([]models.ProviderUser, error) {
	if opts.ByProviderID == 0 && opts.ByIdentityID == 0 && len(opts.ByIdentityIDs) == 0 {
		return nil, fmt.Errorf("ListProviderUsers must specify provider ID, identity ID, or a list of identity IDs")
	}
	table := &providerUserTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(table))
	if opts.SCIMParameters != nil {
		query.B(", count(*) OVER()")
	}
	query.B("FROM")
	query.B(table.Table())
	query.B("INNER JOIN providers ON provider_users.provider_id = providers.id AND providers.organization_id = ?", tx.OrganizationID())
	query.B("WHERE 1=1") // this is always true, used to make the logic of adding clauses simpler by always appending them with an AND
	if opts.ByProviderID != 0 {
		query.B("AND provider_id = ?", opts.ByProviderID)
	}
	if opts.ByIdentityID != 0 {
		query.B("AND identity_id = ?", opts.ByIdentityID)
	}
	if len(opts.ByIdentityIDs) != 0 {
		query.B("AND identity_id IN (?)", opts.ByIdentityIDs)
	}
	if opts.HideInactive {
		query.B("AND active = ?", opts.HideInactive)
	}

	query.B("ORDER BY email ASC")

	if opts.SCIMParameters != nil {
		// apply scim parameters
		if opts.SCIMParameters.Count != 0 {
			query.B("LIMIT ?", opts.SCIMParameters.Count)
		}
		if opts.SCIMParameters.StartIndex > 0 {
			offset := opts.SCIMParameters.StartIndex - 1 // start index begins at 1, not 0
			query.B("OFFSET ?", offset)
		}
	}

	rows, err := tx.Query(query.String(), query.Args...)
	if err != nil {
		return nil, err
	}
	result, err := scanRows(rows, func(pu *models.ProviderUser) []any {
		fields := (*providerUserTable)(pu).ScanFields()
		if opts.SCIMParameters != nil {
			fields = append(fields, &opts.SCIMParameters.TotalCount)
		}
		return fields
	})
	if err != nil {
		return nil, fmt.Errorf("scan provider users: %w", err)
	}

	if opts.SCIMParameters != nil && opts.SCIMParameters.Count == 0 {
		opts.SCIMParameters.Count = opts.SCIMParameters.TotalCount
	}

	return result, nil
}

type DeleteProviderUsersOptions struct {
	// ByIdentityID instructs DeleteProviderUsers to delete tracked provider users for this identity ID
	ByIdentityID uid.ID
	// ByProviderID instructs DeleteProviderUsers to delete tracked provider users for this provider ID
	ByProviderID uid.ID
}

func DeleteProviderUsers(tx WriteTxn, opts DeleteProviderUsersOptions) error {
	if opts.ByProviderID == 0 {
		return fmt.Errorf("DeleteProviderUsers must supply a provider_id")
	}
	query := querybuilder.New("DELETE FROM provider_users")
	query.B("WHERE provider_id = ?", opts.ByProviderID)
	if opts.ByIdentityID != 0 {
		query.B("AND identity_id = ?", opts.ByIdentityID)
	}

	_, err := tx.Exec(query.String(), query.Args...)
	return err
}

func GetProviderUser(tx ReadTxn, providerID, identityID uid.ID) (*models.ProviderUser, error) {
	pu := &providerUserTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(pu))
	query.B("FROM")
	query.B(pu.Table())
	query.B("WHERE provider_id = ? and identity_id = ?", providerID, identityID)
	err := tx.QueryRow(query.String(), query.Args...).Scan(pu.ScanFields()...)
	if err != nil {
		return nil, handleReadError(err)
	}
	return (*models.ProviderUser)(pu), nil
}

func SyncProviderUser(ctx context.Context, tx GormTxn, user *models.Identity, provider *models.Provider, oidcClient providers.OIDCClient) error {
	providerUser, err := GetProviderUser(tx, provider.ID, user.ID)
	if err != nil {
		return err
	}

	accessToken, expiry, err := oidcClient.RefreshAccessToken(ctx, providerUser)
	if err != nil {
		return fmt.Errorf("refresh provider access: %w", err)
	}

	// update the stored access token if it was refreshed
	if accessToken != string(providerUser.AccessToken) {
		logging.Debugf("access token for user at provider %s was refreshed", providerUser.ProviderID)

		providerUser.AccessToken = models.EncryptedAtRest(accessToken)
		providerUser.ExpiresAt = *expiry
		providerUser.LastUpdate = time.Now().UTC()

		err = UpdateProviderUser(tx, providerUser)
		if err != nil {
			return fmt.Errorf("update provider user on sync: %w", err)
		}
	}

	info, err := oidcClient.GetUserInfo(ctx, providerUser)
	if err != nil {
		return fmt.Errorf("oidc user sync failed: %w", err)
	}

	if err := AssignUserToProviderGroups(tx, providerUser, provider, info.Groups); err != nil {
		return fmt.Errorf("assign identity to groups: %w", err)
	}

	return nil
}

type SCIMParameters struct {
	Count      int // the number of items to return
	StartIndex int // the offset to start counting from
	TotalCount int // the total number of items that match the query
	// TODO: filter query param
}

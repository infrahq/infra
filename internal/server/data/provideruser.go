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
	return []string{"identity_id", "provider_id", "email", "groups", "last_update", "redirect_url", "access_token", "refresh_token", "expires_at"}
}

func (p providerUserTable) Values() []any {
	return []any{p.IdentityID, p.ProviderID, p.Email, p.Groups, p.LastUpdate, p.RedirectURL, p.AccessToken, p.RefreshToken, p.ExpiresAt}
}

func (p *providerUserTable) ScanFields() []any {
	return []any{&p.IdentityID, &p.ProviderID, &p.Email, &p.Groups, &p.LastUpdate, &p.RedirectURL, &p.AccessToken, &p.RefreshToken, &p.ExpiresAt}
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

func listProviderUsers(tx ReadTxn, providerID uid.ID) ([]models.ProviderUser, error) {
	table := &providerUserTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(table))
	query.B("FROM")
	query.B(table.Table())
	query.B("WHERE provider_id = ?", providerID)
	rows, err := tx.Query(query.String(), query.Args...)
	if err != nil {
		return nil, err
	}
	return scanRows(rows, func(pu *models.ProviderUser) []any {
		return (*providerUserTable)(pu).ScanFields()
	})
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

		err = UpdateProviderUser(tx, providerUser)
		if err != nil {
			return fmt.Errorf("update provider user on sync: %w", err)
		}
	}

	info, err := oidcClient.GetUserInfo(ctx, providerUser)
	if err != nil {
		return fmt.Errorf("oidc user sync failed: %w", err)
	}

	if err := AssignIdentityToGroups(tx, user, provider, info.Groups); err != nil {
		return fmt.Errorf("assign identity to groups: %w", err)
	}

	return nil
}

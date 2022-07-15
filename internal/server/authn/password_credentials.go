package authn

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// passwordCredentialAuthn allows presenting username/password credentials in exchange for an access key
type passwordCredentialAuthn struct {
	OrganizationID uid.ID
	Username       string
	Password       string
}

func NewPasswordCredentialAuthentication(orgID uid.ID, username, password string) LoginMethod {
	return &passwordCredentialAuthn{
		OrganizationID: orgID,
		Username:       username,
		Password:       password,
	}
}

func (a *passwordCredentialAuthn) Authenticate(_ context.Context, db *gorm.DB) (*models.Organization, *models.Identity, *models.Provider, AuthScope, error) {
	scope := AuthScope{}

	org, err := data.GetOrganization(db, data.ByID(a.OrganizationID))
	if err != nil {
		return nil, nil, nil, scope, fmt.Errorf("could not get org: '%s' %w", a.OrganizationID, err)
	}

	identity, err := data.GetIdentity(db, data.ByName(a.Username))
	if err != nil {
		return nil, nil, nil, scope, fmt.Errorf("could not get identity for username: %w", err)
	}

	// Infra users can have only one username/password combo, look it up
	userCredential, err := data.GetCredential(db, data.ByIdentityID(identity.ID))
	if err != nil {
		return nil, nil, nil, scope, fmt.Errorf("validate creds get user: %w", err)
	}

	// compare the stored hash of the user's password and the hash of the presented password
	err = bcrypt.CompareHashAndPassword(userCredential.PasswordHash, []byte(a.Password))
	if err != nil {
		// this probably means the password was wrong
		return nil, nil, nil, scope, fmt.Errorf("could not verify password: %w", err)
	}

	if userCredential.OneTimePassword {
		// scope the login down to Password Reset Only
		scope.PasswordResetOnly = true
	}

	// authentication was a success
	return org, identity, data.InfraProvider(db), scope, nil // password login is always for infra users
}

func (a *passwordCredentialAuthn) Name() string {
	return "credentials"
}

func (a *passwordCredentialAuthn) RequiresUpdate(db *gorm.DB) (bool, error) {
	identity, err := data.GetIdentity(db, data.ByName(a.Username))
	if err != nil {
		return false, fmt.Errorf("could not get identity for username: %w", err)
	}

	cred, err := data.GetCredential(db, data.ByIdentityID(identity.ID))
	if err != nil {
		return false, fmt.Errorf("could not get credential for username: %w", err)
	}

	return cred.OneTimePassword, nil
}

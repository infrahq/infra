package authn

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/infrahq/infra/internal/server/data"
)

// passwordCredentialAuthn allows presenting username/password credentials in exchange for an access key
type passwordCredentialAuthn struct {
	Username string
	Password string
}

func NewPasswordCredentialAuthentication(username, password string) LoginMethod {
	return &passwordCredentialAuthn{
		Username: username,
		Password: password,
	}
}

func (a *passwordCredentialAuthn) Authenticate(_ context.Context, db data.GormTxn, requestedExpiry time.Time) (AuthenticatedIdentity, error) {
	if a.Username == "" {
		return AuthenticatedIdentity{}, fmt.Errorf("username required for password authentication")
	}
	identity, err := data.GetIdentity(db, data.GetIdentityOptions{ByName: a.Username})
	if err != nil {
		return AuthenticatedIdentity{}, fmt.Errorf("could not get identity for username: %w", err)
	}

	// Infra users can have only one username/password combo, look it up
	userCredential, err := data.GetCredential(db, data.ByIdentityID(identity.ID))
	if err != nil {
		return AuthenticatedIdentity{}, fmt.Errorf("validate creds get user: %w", err)
	}

	// compare the stored hash of the user's password and the hash of the presented password
	err = bcrypt.CompareHashAndPassword(userCredential.PasswordHash, []byte(a.Password))
	if err != nil {
		// this probably means the password was wrong
		return AuthenticatedIdentity{}, fmt.Errorf("could not verify password: %w", err)
	}

	authnIdentity := AuthenticatedIdentity{
		Identity:      identity,
		Provider:      data.InfraProvider(db),
		SessionExpiry: requestedExpiry,
	}

	if userCredential.OneTimePassword {
		// scope the login down to Password Reset Only
		authnIdentity.AuthScope.PasswordResetOnly = true
		authnIdentity.CredentialUpdateRequired = true
	}

	// authentication was a success
	return authnIdentity, nil // password login is always for infra users
}

func (a *passwordCredentialAuthn) Name() string {
	return "credentials"
}

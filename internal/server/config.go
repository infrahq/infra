package server

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type BootstrapConfig struct {
	DefaultOrganizationDomain string
	Users                     []User
}

type User struct {
	Name      string
	AccessKey Secret
	Password  Secret
	InfraRole string
}

func (u User) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("name", u.Name),
	}
}

func (c BootstrapConfig) ValidationRules() []validate.ValidationRule {
	// no-op implement to satisfy the interface
	return nil
}

// Secret provides backwards compatibility for the old secret loading microformat
// with file:, env:, and plaintext: prefix. Custom secret providers are not
// supported.
type Secret string

func (s *Secret) Set(raw string) error {
	switch {
	case strings.HasPrefix(raw, "file:"):
		content, err := os.ReadFile(strings.TrimPrefix(raw, "file:"))
		if err != nil {
			return err
		}
		*s = Secret(content)
	case strings.HasPrefix(raw, "env:"):
		*s = Secret(os.Getenv(strings.TrimPrefix(raw, "env:")))
	case strings.HasPrefix(raw, "plaintext:"):
		*s = Secret(strings.TrimPrefix(raw, "plaintext:"))
	default:
		*s = Secret(raw)
	}
	return nil
}

func (s Secret) String() string {
	return "<redacted>"
}

func (s Secret) GoString() string {
	return "<redacted>"
}

func (s Server) loadConfig(config BootstrapConfig) error {
	if err := validate.Validate(config); err != nil {
		return err
	}

	org := s.db.DefaultOrg

	tx, err := s.db.Begin(context.Background(), nil)
	if err != nil {
		return err
	}
	defer logError(tx.Rollback, "failed to rollback loadConfig transaction")
	tx = tx.WithOrgID(org.ID)

	if config.DefaultOrganizationDomain != org.Domain {
		org.Domain = config.DefaultOrganizationDomain
		if err := data.UpdateOrganization(tx, org); err != nil {
			return fmt.Errorf("update default org domain: %w", err)
		}
	}

	for _, u := range config.Users {
		if err := s.loadUser(tx, u); err != nil {
			return fmt.Errorf("load user %v: %w", u.Name, err)
		}
	}

	return tx.Commit()
}

func loadGrant(tx data.WriteTxn, userID uid.ID, role string) error {
	if role == "" {
		return nil
	}
	_, err := data.GetGrant(tx, data.GetGrantOptions{
		BySubject:   models.NewSubjectForUser(userID),
		ByResource:  access.ResourceInfraAPI,
		ByPrivilege: role,
	})
	if err == nil || !errors.Is(err, internal.ErrNotFound) {
		return err
	}

	grant := &models.Grant{
		Subject:   models.NewSubjectForUser(userID),
		Resource:  access.ResourceInfraAPI,
		Privilege: role,
		CreatedBy: models.CreatedBySystem,
	}
	return data.CreateGrant(tx, grant)
}

func (s Server) loadUser(db data.WriteTxn, input User) error {
	identity, err := data.GetIdentity(db, data.GetIdentityOptions{ByName: input.Name})
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return err
		}

		if input.Name != models.InternalInfraConnectorIdentityName {
			_, err := mail.ParseAddress(input.Name)
			if err != nil {
				logging.Warnf("user name %q in server configuration is not a valid email, please update this name to a valid email", input.Name)
			}
		}

		identity = &models.Identity{
			Name:      input.Name,
			CreatedBy: models.CreatedBySystem,
		}

		if err := data.CreateIdentity(db, identity); err != nil {
			return err
		}

		_, err = data.CreateProviderUser(db, data.InfraProvider(db), identity)
		if err != nil {
			return err
		}
	}

	if err := s.loadCredential(db, identity, input.Password); err != nil {
		return err
	}

	if err := s.loadAccessKey(db, identity, input.AccessKey); err != nil {
		return err
	}

	if err := loadGrant(db, identity.ID, input.InfraRole); err != nil {
		return err
	}

	return nil
}

func (s Server) loadCredential(db data.WriteTxn, identity *models.Identity, password Secret) error {
	if password == "" {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	credential, err := data.GetCredentialByUserID(db, identity.ID)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return err
		}

		credential := &models.Credential{
			IdentityID:   identity.ID,
			PasswordHash: hash,
		}

		if err := data.CreateCredential(db, credential); err != nil {
			return err
		}

		if _, err := data.CreateProviderUser(db, data.InfraProvider(db), identity); err != nil {
			return err
		}

		return nil
	}

	credential.PasswordHash = hash

	return data.UpdateCredential(db, credential)
}

func (s Server) loadAccessKey(db data.WriteTxn, identity *models.Identity, key Secret) error {
	if key == "" {
		return nil
	}

	keyID, secret, ok := strings.Cut(string(key), ".")
	if !ok {
		return fmt.Errorf("invalid access key format")
	}

	accessKey, err := data.GetAccessKeyByKeyID(db, keyID)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return err
		}

		provider := data.InfraProvider(db)

		accessKey := &models.AccessKey{
			IssuedFor:  identity.ID,
			ExpiresAt:  time.Now().AddDate(10, 0, 0),
			KeyID:      keyID,
			Secret:     secret,
			ProviderID: provider.ID,
			Scopes:     models.CommaSeparatedStrings{models.ScopeAllowCreateAccessKey}, // allows user to create access keys
		}

		if _, err := data.CreateAccessKey(db, accessKey); err != nil {
			return err
		}

		if _, err := data.CreateProviderUser(db, provider, identity); err != nil {
			return err
		}

		return nil
	}

	if accessKey.IssuedFor != identity.ID {
		return fmt.Errorf("access key assigned to %q is already assigned to another user, a user's access key must have a unique ID", identity.Name)
	}

	accessKey.Secret = secret

	return data.UpdateAccessKey(db, accessKey)
}

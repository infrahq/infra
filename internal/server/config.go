package server

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type Provider struct {
	Name         string
	URL          string
	ClientID     string
	ClientSecret string
	Kind         string
	AuthURL      string
	Scopes       []string

	// fields used to directly query an external API
	PrivateKey       string
	ClientEmail      string
	DomainAdminEmail string
}

func (p Provider) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		api.ValidateName(p.Name),
		validate.Required("name", p.Name),
		validate.Required("url", p.URL),
		validate.Required("clientID", p.ClientID),
		validate.Required("clientSecret", p.ClientSecret),
	}
}

type Grant struct {
	User     string
	Group    string
	Resource string
	Role     string
}

func (g Grant) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.RequireOneOf(
			validate.Field{Name: "user", Value: g.User},
			validate.Field{Name: "group", Value: g.Group},
		),
		validate.Required("resource", g.Resource),
	}
}

type User struct {
	Name      string
	AccessKey string
	Password  string
}

func (u User) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("name", u.Name),
	}
}

type Config struct {
	DefaultOrganizationDomain string
	Providers                 []Provider
	Grants                    []Grant
	Users                     []User
}

func (c Config) ValidationRules() []validate.ValidationRule {
	// no-op implement to satisfy the interface
	return nil
}

func (s Server) loadConfig(config Config) error {
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

	// inject internal infra provider
	config.Providers = append(config.Providers, Provider{
		Name: models.InternalInfraProviderName,
		Kind: models.ProviderKindInfra.String(),
	})

	config.Users = append(config.Users, User{
		Name: models.InternalInfraConnectorIdentityName,
	})

	config.Grants = append(config.Grants, Grant{
		User:     models.InternalInfraConnectorIdentityName,
		Role:     models.InfraConnectorRole,
		Resource: "infra",
	})

	if err := s.loadProviders(tx, config.Providers); err != nil {
		return fmt.Errorf("load providers: %w", err)
	}

	// extract users from grants and add them to users
	for _, g := range config.Grants {
		switch {
		case g.User != "":
			config.Users = append(config.Users, User{Name: g.User})
		}
	}

	if err := s.loadUsers(tx, config.Users); err != nil {
		return fmt.Errorf("load users: %w", err)
	}

	if err := s.loadGrants(tx, config.Grants); err != nil {
		return fmt.Errorf("load grants: %w", err)
	}

	return tx.Commit()
}

func (s Server) loadProviders(db data.WriteTxn, providers []Provider) error {
	keep := []uid.ID{}

	for _, p := range providers {
		provider, err := s.loadProvider(db, p)
		if err != nil {
			return err
		}

		keep = append(keep, provider.ID)
	}

	// remove any provider previously defined by config
	if err := data.DeleteProviders(db, data.DeleteProvidersOptions{
		CreatedBy: models.CreatedBySystem,
		NotIDs:    keep,
	}); err != nil {
		return err
	}

	return nil
}

func (s Server) loadProvider(db data.WriteTxn, input Provider) (*models.Provider, error) {
	// provider kind is an optional field
	kind, err := models.ParseProviderKind(input.Kind)
	if err != nil {
		return nil, fmt.Errorf("could not parse provider in config load: %w", err)
	}

	clientSecret := input.ClientSecret
	provider, err := data.GetProvider(db, data.GetProviderOptions{ByName: input.Name})
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		provider := &models.Provider{
			Name:         input.Name,
			URL:          input.URL,
			ClientID:     input.ClientID,
			ClientSecret: models.EncryptedAtRest(clientSecret),
			AuthURL:      input.AuthURL,
			Scopes:       input.Scopes,
			Kind:         kind,
			CreatedBy:    models.CreatedBySystem,

			PrivateKey:       models.EncryptedAtRest(input.PrivateKey),
			ClientEmail:      input.ClientEmail,
			DomainAdminEmail: input.DomainAdminEmail,
		}

		if provider.Kind != models.ProviderKindInfra {
			// only call the provider to resolve info if it is not known
			if input.AuthURL == "" && len(input.Scopes) == 0 {
				providerClient := providers.NewOIDCClient(*provider, clientSecret, "")
				authServerInfo, err := providerClient.AuthServerInfo(context.Background())
				if err != nil {
					if errors.Is(err, context.DeadlineExceeded) {
						return nil, fmt.Errorf("%w: %s", internal.ErrBadGateway, err)
					}
					return nil, err
				}

				provider.AuthURL = authServerInfo.AuthURL
				provider.Scopes = authServerInfo.ScopesSupported
			}

			// check that the scopes we need are set
			supportedScopes := make(map[string]bool)
			for _, s := range provider.Scopes {
				supportedScopes[s] = true
			}
			if !supportedScopes["openid"] || !supportedScopes["email"] {
				return nil, fmt.Errorf("required scopes 'openid' and 'email' not found on provider %q", input.Name)
			}
		}

		if err := data.CreateProvider(db, provider); err != nil {
			return nil, err
		}

		return provider, nil
	}

	// provider already exists, update it
	provider.URL = input.URL
	provider.ClientID = input.ClientID
	provider.ClientSecret = models.EncryptedAtRest(clientSecret)
	provider.Kind = kind

	if err := data.UpdateProvider(db, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

func (s Server) loadGrants(db data.WriteTxn, grants []Grant) error {
	keep := make([]uid.ID, 0, len(grants))

	for _, g := range grants {
		grant, err := s.loadGrant(db, g)
		if err != nil {
			return err
		}

		keep = append(keep, grant.ID)
	}

	// remove any grant previously defined by config
	if err := data.DeleteGrants(db, data.DeleteGrantsOptions{
		NotIDs:      keep,
		ByCreatedBy: models.CreatedBySystem,
	}); err != nil {
		return err
	}

	return nil
}

func (Server) loadGrant(db data.WriteTxn, input Grant) (*models.Grant, error) {
	var id uid.PolymorphicID

	switch {
	case input.User != "":
		user, err := data.GetIdentity(db, data.GetIdentityOptions{ByName: input.User})
		if err != nil {
			return nil, err
		}

		id = uid.NewIdentityPolymorphicID(user.ID)

	case input.Group != "":
		group, err := data.GetGroup(db, data.GetGroupOptions{ByName: input.Group})
		if err != nil {
			if !errors.Is(err, internal.ErrNotFound) {
				return nil, err
			}

			logging.Debugf("creating placeholder group %q", input.Group)

			// group does not exist yet, create a placeholder
			group = &models.Group{
				Name:      input.Group,
				CreatedBy: models.CreatedBySystem,
			}

			if err := data.CreateGroup(db, group); err != nil {
				return nil, err
			}
		}

		id = uid.NewGroupPolymorphicID(group.ID)

	default:
		return nil, errors.New("invalid grant: missing identity")
	}

	if len(input.Role) == 0 {
		input.Role = models.BasePermissionConnect
	}

	grant, err := data.GetGrant(db, data.GetGrantOptions{
		BySubject:   id,
		ByResource:  input.Resource,
		ByPrivilege: input.Role,
	})
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		grant = &models.Grant{
			Subject:   id,
			Resource:  input.Resource,
			Privilege: input.Role,
			CreatedBy: models.CreatedBySystem,
		}

		if err := data.CreateGrant(db, grant); err != nil {
			return nil, err
		}
	}

	return grant, nil
}

func (s Server) loadUsers(db data.WriteTxn, users []User) error {
	keep := make([]uid.ID, 0, len(users)+1)

	for _, i := range users {
		user, err := s.loadUser(db, i)
		if err != nil {
			return err
		}

		keep = append(keep, user.ID)
	}

	// remove any users previously defined by config
	opts := data.DeleteIdentitiesOptions{
		ByProviderID: data.InfraProvider(db).ID,
		ByNotIDs:     keep,
		CreatedBy:    models.CreatedBySystem,
	}
	if err := data.DeleteIdentities(db, opts); err != nil {
		return err
	}

	return nil
}

func (s Server) loadUser(db data.WriteTxn, input User) (*models.Identity, error) {
	identity, err := data.GetIdentity(db, data.GetIdentityOptions{ByName: input.Name})
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
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
			return nil, err
		}

		_, err = data.CreateProviderUser(db, data.InfraProvider(db), identity)
		if err != nil {
			return nil, err
		}

	}

	if err := s.loadCredential(db, identity, input.Password); err != nil {
		return nil, err
	}

	if err := s.loadAccessKey(db, identity, input.AccessKey); err != nil {
		return nil, err
	}

	return identity, nil
}

func (s Server) loadCredential(db data.WriteTxn, identity *models.Identity, password string) error {
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

	if err := data.UpdateCredential(db, credential); err != nil {
		return err
	}

	return nil
}

func (s Server) loadAccessKey(db data.WriteTxn, identity *models.Identity, key string) error {
	if key == "" {
		return nil
	}

	keyID, secret, ok := strings.Cut(key, ".")
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

	if err := data.UpdateAccessKey(db, accessKey); err != nil {
		return err
	}

	return nil
}

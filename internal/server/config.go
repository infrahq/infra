package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/secrets"
	"github.com/infrahq/infra/uid"
)

type Provider struct {
	Name         string `mapstructure:"name" validate:"required"`
	URL          string `mapstructure:"url" validate:"required"`
	ClientID     string `mapstructure:"clientID" validate:"required"`
	ClientSecret string `mapstructure:"clientSecret" validate:"required"`
}

type Grant struct {
	User     string `mapstructure:"user" validate:"excluded_with=Group,excluded_with=Machine"`
	Group    string `mapstructure:"group" validate:"excluded_with=User,excluded_with=Machine"`
	Machine  string `mapstructure:"machine" validate:"excluded_with=User,excluded_with=Group"`
	Provider string `mapstructure:"provider"`
	Role     string `mapstructure:"role" validate:"required"`
	Resource string `mapstructure:"resource" validate:"required"`
}

type Config struct {
	Providers []Provider `mapstructure:"providers" validate:"dive"`
	Grants    []Grant    `mapstructure:"grants" validate:"dive"`
}

type KeyProvider struct {
	Kind   string      `yaml:"kind" validate:"required"`
	Config interface{} // contains secret-provider-specific config
}

var _ yaml.Unmarshaler = &KeyProvider{}

type nativeSecretProviderConfig struct {
	SecretStorageName string `yaml:"secretProvider"`
}

func (s *Server) importSecretKeys() error {
	var err error

	if s.keys == nil {
		s.keys = map[string]secrets.SymmetricKeyProvider{}
	}

	// default to file-based native secret provider
	s.keys["native"] = secrets.NewNativeSecretProvider(s.secrets["file"])

	for _, keyConfig := range s.options.Keys {
		switch keyConfig.Kind {
		case "native":
			cfg, ok := keyConfig.Config.(nativeSecretProviderConfig)
			if !ok {
				return fmt.Errorf("expected key config to be NativeSecretProviderConfig, but was %t", keyConfig.Config)
			}

			storageProvider, found := s.secrets[cfg.SecretStorageName]
			if !found {
				return fmt.Errorf("secret storage name %q not found", cfg.SecretStorageName)
			}

			sp := secrets.NewNativeSecretProvider(storageProvider)
			s.keys[keyConfig.Kind] = sp
		case "awskms":
			cfg, ok := keyConfig.Config.(secrets.AWSKMSConfig)
			if !ok {
				return fmt.Errorf("expected key config to be AWSKMSConfig, but was %t", keyConfig.Config)
			}

			cfg.AccessKeyID, err = secrets.GetSecret(cfg.AccessKeyID, s.secrets)
			if err != nil {
				return fmt.Errorf("getting secret for awskms accessKeyID: %w", err)
			}

			cfg.SecretAccessKey, err = secrets.GetSecret(cfg.SecretAccessKey, s.secrets)
			if err != nil {
				return fmt.Errorf("getting secret for awskms secretAccessKey: %w", err)
			}

			sp, err := secrets.NewAWSKMSSecretProviderFromConfig(cfg)
			if err != nil {
				return err
			}

			s.keys[keyConfig.Kind] = sp
		case "vault":
			cfg, ok := keyConfig.Config.(secrets.VaultConfig)
			if !ok {
				return fmt.Errorf("expected key config to be VaultConfig, but was %t", keyConfig.Config)
			}

			cfg.Token, err = secrets.GetSecret(cfg.Token, s.secrets)
			if err != nil {
				return err
			}

			sp, err := secrets.NewVaultSecretProviderFromConfig(cfg)
			if err != nil {
				return err
			}

			s.keys[keyConfig.Kind] = sp
		}
	}

	return nil
}

func (sp *KeyProvider) UnmarshalYAML(unmarshal func(interface{}) error) error {
	tmp := &simpleConfigSecretProvider{}

	if err := unmarshal(&tmp); err != nil {
		return fmt.Errorf("unmarshalling secret provider: %w", err)
	}

	sp.Kind = tmp.Kind

	switch sp.Kind {
	case "vault":
		p := secrets.NewVaultConfig()
		if err := unmarshal(&p); err != nil {
			return fmt.Errorf("unmarshal yaml: %w", err)
		}

		sp.Config = p
	case "awskms":
		p := secrets.NewAWSKMSConfig()
		if err := unmarshal(&p); err != nil {
			return fmt.Errorf("unmarshal yaml: %w", err)
		}

		if err := unmarshal(&p.AWSConfig); err != nil {
			return fmt.Errorf("unmarshal yaml: %w", err)
		}

		sp.Config = p
	case "native":
		p := nativeSecretProviderConfig{}
		if err := unmarshal(&p); err != nil {
			return fmt.Errorf("unmarshal yaml: %w", err)
		}

		sp.Config = p
	default:
		return fmt.Errorf("unknown key provider type %q, expected one of %q", sp.Kind, secrets.SymmetricKeyProviderKinds)
	}

	return nil
}

type SecretProvider struct {
	Kind   string      `yaml:"kind" validate:"required"`
	Name   string      `yaml:"name"` // optional
	Config interface{} // contains secret-provider-specific config
}

type simpleConfigSecretProvider struct {
	Kind string `yaml:"kind"`
	Name string `yaml:"name"`
}

var baseSecretStorageKinds = []string{
	"env",
	"file",
	"plaintext",
	"kubernetes",
}

func isABaseSecretStorageKind(s string) bool {
	for _, item := range baseSecretStorageKinds {
		if item == s {
			return true
		}
	}

	return false
}

func (s *Server) importSecrets() error {
	if s.secrets == nil {
		s.secrets = map[string]secrets.SecretStorage{}
	}

	loadSecretConfig := func(secret SecretProvider) (err error) {
		name := secret.Name
		if len(name) == 0 {
			name = secret.Kind
		}

		if _, found := s.secrets[name]; found {
			return fmt.Errorf("duplicate secret configuration for %q, please provide a unique name for this secret configuration", name)
		}

		switch secret.Kind {
		case "vault":
			cfg, ok := secret.Config.(secrets.VaultConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be VaultConfig, but was %t", secret.Config)
			}

			cfg.Token, err = secrets.GetSecret(cfg.Token, s.secrets)
			if err != nil {
				return err
			}

			vault, err := secrets.NewVaultSecretProviderFromConfig(cfg)
			if err != nil {
				return fmt.Errorf("creating vault provider: %w", err)
			}

			s.secrets[name] = vault
		case "awsssm":
			cfg, ok := secret.Config.(secrets.AWSSSMConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be AWSSSMConfig, but was %t", secret.Config)
			}

			cfg.AccessKeyID, err = secrets.GetSecret(cfg.AccessKeyID, s.secrets)
			if err != nil {
				return err
			}

			cfg.SecretAccessKey, err = secrets.GetSecret(cfg.SecretAccessKey, s.secrets)
			if err != nil {
				return err
			}

			ssm, err := secrets.NewAWSSSMSecretProviderFromConfig(cfg)
			if err != nil {
				return fmt.Errorf("creating aws ssm: %w", err)
			}

			s.secrets[name] = ssm
		case "awssecretsmanager":
			cfg, ok := secret.Config.(secrets.AWSSecretsManagerConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be AWSSecretsManagerConfig, but was %t", secret.Config)
			}

			cfg.AccessKeyID, err = secrets.GetSecret(cfg.AccessKeyID, s.secrets)
			if err != nil {
				return err
			}

			cfg.SecretAccessKey, err = secrets.GetSecret(cfg.SecretAccessKey, s.secrets)
			if err != nil {
				return err
			}

			sm, err := secrets.NewAWSSecretsManagerFromConfig(cfg)
			if err != nil {
				return fmt.Errorf("creating aws sm: %w", err)
			}

			s.secrets[name] = sm
		case "kubernetes":
			cfg, ok := secret.Config.(secrets.KubernetesConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be KubernetesConfig, but was %t", secret.Config)
			}

			k8s, err := secrets.NewKubernetesSecretProviderFromConfig(cfg)
			if err != nil {
				return fmt.Errorf("creating k8s secret provider: %w", err)
			}

			s.secrets[name] = k8s
		case "env":
			cfg, ok := secret.Config.(secrets.GenericConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be GenericConfig, but was %t", secret.Config)
			}

			f := secrets.NewEnvSecretProviderFromConfig(cfg)
			s.secrets[name] = f
		case "file":
			cfg, ok := secret.Config.(secrets.FileConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be FileConfig, but was %t", secret.Config)
			}

			f := secrets.NewFileSecretProviderFromConfig(cfg)
			s.secrets[name] = f
		case "plaintext", "":
			cfg, ok := secret.Config.(secrets.GenericConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be GenericConfig, but was %t", secret.Config)
			}

			f := secrets.NewPlainSecretProviderFromConfig(cfg)
			s.secrets[name] = f
		default:
			return fmt.Errorf("unknown secret provider type %q", secret.Kind)
		}

		return nil
	}

	// check all base types first
	for _, secret := range s.options.Secrets {
		if !isABaseSecretStorageKind(secret.Kind) {
			continue
		}

		if err := loadSecretConfig(secret); err != nil {
			return err
		}
	}

	if err := s.loadDefaultSecretConfig(); err != nil {
		return err
	}

	// now load non-base types which might depend on them.
	for _, secret := range s.options.Secrets {
		if isABaseSecretStorageKind(secret.Kind) {
			continue
		}

		if err := loadSecretConfig(secret); err != nil {
			return err
		}
	}

	return nil
}

// loadDefaultSecretConfig loads configuration for types that should be available,
// assuming the user didn't override the configuration for them.
func (s *Server) loadDefaultSecretConfig() error {
	// set up the default supported types
	if _, found := s.secrets["env"]; !found {
		f := secrets.NewEnvSecretProviderFromConfig(secrets.GenericConfig{})
		s.secrets["env"] = f
	}

	if _, found := s.secrets["file"]; !found {
		f := secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{})
		s.secrets["file"] = f
	}

	if _, found := s.secrets["plaintext"]; !found {
		f := secrets.NewPlainSecretProviderFromConfig(secrets.GenericConfig{})
		s.secrets["plaintext"] = f
	}

	if _, found := s.secrets["kubernetes"]; !found {
		// only setup k8s automatically if KUBERNETES_SERVICE_HOST is defined; ie, we are in the clustes.
		if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
			k8s, err := secrets.NewKubernetesSecretProviderFromConfig(secrets.NewKubernetesConfig())
			if err != nil {
				return fmt.Errorf("creating k8s secret provider: %w", err)
			}

			s.secrets["kubernetes"] = k8s
		}
	}

	return nil
}

func (sp *SecretProvider) UnmarshalYAML(unmarshal func(interface{}) error) error {
	tmp := &simpleConfigSecretProvider{}

	if err := unmarshal(&tmp); err != nil {
		return fmt.Errorf("unmarshalling secret provider: %w", err)
	}

	sp.Kind = tmp.Kind
	sp.Name = tmp.Name

	switch tmp.Kind {
	case "vault":
		p := secrets.NewVaultConfig()
		if err := unmarshal(&p); err != nil {
			return fmt.Errorf("unmarshal yaml: %w", err)
		}

		sp.Config = p
	case "awsssm":
		p := secrets.AWSSSMConfig{}
		if err := unmarshal(&p); err != nil {
			return fmt.Errorf("unmarshal yaml: %w", err)
		}

		if err := unmarshal(&p.AWSConfig); err != nil {
			return fmt.Errorf("unmarshal yaml: %w", err)
		}

		sp.Config = p
	case "awssecretsmanager":
		p := secrets.AWSSecretsManagerConfig{}
		if err := unmarshal(&p); err != nil {
			return fmt.Errorf("unmarshal yaml: %w", err)
		}

		if err := unmarshal(&p.AWSConfig); err != nil {
			return fmt.Errorf("unmarshal yaml: %w", err)
		}

		sp.Config = p
	case "kubernetes":
		p := secrets.NewKubernetesConfig()
		if err := unmarshal(&p); err != nil {
			return fmt.Errorf("unmarshal yaml: %w", err)
		}

		sp.Config = p
	case "env":
		p := secrets.GenericConfig{}
		if err := unmarshal(&p); err != nil {
			return fmt.Errorf("unmarshal yaml: %w", err)
		}

		sp.Config = p
	case "file":
		p := secrets.FileConfig{}
		if err := unmarshal(&p); err != nil {
			return fmt.Errorf("unmarshal yaml: %w", err)
		}

		if err := unmarshal(&p.GenericConfig); err != nil {
			return fmt.Errorf("unmarshal yaml: %w", err)
		}

		sp.Config = p
	case "plaintext", "":
		p := secrets.GenericConfig{}
		if err := unmarshal(&p); err != nil {
			return fmt.Errorf("unmarshal yaml: %w", err)
		}

		sp.Config = p
	default:
		return fmt.Errorf("unknown secret provider type %q, expected one of %q", tmp.Kind, secrets.SecretStorageProviderKinds)
	}

	return nil
}

// setupInfraIdentityProvider creates the internal identity provider where local identities are stored
func (s *Server) setupInfraIdentityProvider() error {
	_, err := data.GetProvider(s.db, data.ByName(models.InternalInfraProviderName))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return fmt.Errorf("setup infra provider: %w", err)
		}

		if err := data.CreateProvider(s.db, &models.Provider{Name: models.InternalInfraProviderName, CreatedBy: models.CreatedBySystem}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) importAccessKeys() error {
	type key struct {
		Secret string
		Role   string
	}

	keys := map[string]key{
		"admin": {
			Secret: s.options.AdminAccessKey,
			Role:   models.InfraAdminRole,
		},
		"connector": {
			Secret: s.options.AccessKey,
			Role:   models.InfraConnectorRole,
		},
	}

	infraProvider, err := data.GetProvider(s.db, data.ByName(models.InternalInfraProviderName))
	if err != nil {
		return err
	}

	for k, v := range keys {
		if v.Secret == "" {
			logging.S.Debugf("%s: secret not set; skipping", k)
			continue
		}

		raw, err := secrets.GetSecret(v.Secret, s.secrets)
		if err != nil && !errors.Is(err, secrets.ErrNotFound) {
			return fmt.Errorf("%s secret: %w", k, err)
		}

		if raw == "" {
			logging.S.Debugf("%s: secret value not set; skipping", k)
			continue
		}

		parts := strings.Split(raw, ".")
		if len(parts) < 2 {
			return fmt.Errorf("%s format: invalid token; expected two parts separated by a '.' character", k)
		}

		// create the machine identity if it doesn't exist
		machine, err := data.GetIdentity(s.db, data.ByName(k))
		if err != nil {
			if !errors.Is(err, internal.ErrNotFound) {
				return fmt.Errorf("get identity: %w", err)
			}

			machine = &models.Identity{
				Name:       k,
				Kind:       models.MachineKind,
				ProviderID: infraProvider.ID,
				LastSeenAt: time.Now().UTC(),
			}

			err = data.CreateIdentity(s.db, machine)
			if err != nil {
				return fmt.Errorf("create identity: %w", err)
			}

			grant := &models.Grant{
				Subject:   machine.PolyID(),
				Privilege: v.Role,
				Resource:  "infra",
			}

			err = data.CreateGrant(s.db, grant)
			if err != nil {
				return fmt.Errorf("create grant: %w", err)
			}
		}

		name := fmt.Sprintf("default %s access key", k)

		accessKey, err := data.GetAccessKey(s.db, data.ByIssuedFor(machine.ID))
		if err != nil {
			if !errors.Is(err, internal.ErrNotFound) {
				return err
			}
		}

		if accessKey != nil {
			sum := sha256.Sum256([]byte(parts[1]))

			// if token name, key, and secret checksum match input, skip recreating the token
			if accessKey.Name == name && subtle.ConstantTimeCompare([]byte(accessKey.KeyID), []byte(parts[0])) == 1 && subtle.ConstantTimeCompare(accessKey.SecretChecksum, sum[:]) == 1 {
				logging.S.Debugf("%s: skip recreating token", k)
				continue
			}

			err = data.DeleteAccessKeys(s.db, data.ByName(name))
			if err != nil {
				return fmt.Errorf("%s delete: %w", k, err)
			}
		}

		accessKey = &models.AccessKey{
			Name:      name,
			KeyID:     parts[0],
			Secret:    parts[1],
			IssuedFor: machine.ID,
			ExpiresAt: time.Now().Add(math.MaxInt64).UTC(),
		}
		if _, err := data.CreateAccessKey(s.db, accessKey); err != nil {
			return fmt.Errorf("%s create: %w", k, err)
		}
	}

	return nil
}

func loadConfig(db *gorm.DB, config Config) error {
	if err := validator.New().Struct(config); err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := loadProviders(tx, config.Providers); err != nil {
			return err
		}

		if err := loadGrants(tx, config.Grants); err != nil {
			return err
		}

		return nil
	})
}

func loadProviders(db *gorm.DB, providers []Provider) error {
	toKeep := make([]uid.ID, 0)

	for _, p := range providers {
		provider, err := loadProvider(db, p)
		if err != nil {
			return err
		}

		toKeep = append(toKeep, provider.ID)
	}

	// remove _all_ providers previously loaded from config
	err := data.DeleteProviders(db, data.ByNotIDs(toKeep), data.CreatedBy(models.CreatedByConfig))
	if err != nil {
		return err
	}

	return nil
}

func loadProvider(db *gorm.DB, input Provider) (*models.Provider, error) {
	provider, err := data.GetProvider(db, data.ByName(input.Name))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		provider := &models.Provider{
			Name:         input.Name,
			URL:          input.URL,
			ClientID:     input.ClientID,
			ClientSecret: models.EncryptedAtRest(input.ClientSecret),
			CreatedBy:    models.CreatedByConfig,
		}

		if err := data.CreateProvider(db, provider); err != nil {
			return nil, err
		}

		return provider, nil
	}

	// provider already exists, update it
	provider.URL = input.URL
	provider.ClientID = input.ClientID
	provider.ClientSecret = models.EncryptedAtRest(input.ClientSecret)
	provider.CreatedBy = models.CreatedByConfig

	if err := data.SaveProvider(db, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

func loadGrants(db *gorm.DB, grants []Grant) error {
	toKeep := make([]uid.ID, 0)

	providers, err := data.ListProviders(db)
	if err != nil {
		return err
	}

	providersMap := make(map[string]uid.ID)
	for _, provider := range providers {
		providersMap[provider.Name] = provider.ID
	}

	for _, g := range grants {
		grant, err := loadGrant(db, g, providersMap)
		if err != nil {
			return err
		}

		toKeep = append(toKeep, grant.ID)
	}

	// remove _all_ grants previously defined by config
	err = data.DeleteGrants(db, data.ByNotIDs(toKeep), data.CreatedBy(models.CreatedByConfig))
	if err != nil {
		return err
	}

	return nil
}

func loadGrant(db *gorm.DB, input Grant, providers map[string]uid.ID) (*models.Grant, error) {
	var (
		id         uid.PolymorphicID
		providerID uid.ID
	)

	if input.User != "" || input.Group != "" {
		// user/group grants require additional input validation
		if len(providers) < 1 {
			return nil, errors.New("no providers configured")
		}

		if len(providers) > 1 && input.Provider == "" {
			return nil, errors.New("grant provider must be specified")
		}

		provider := input.Provider
		if provider == "" {
			for key := range providers {
				provider = key
			}
		}

		var ok bool

		providerID, ok = providers[provider]
		if !ok {
			return nil, fmt.Errorf("unknown provider: %s", provider)
		}
	}

	switch {
	case input.User != "":
		user, err := data.GetIdentity(db, data.ByName(input.User))
		if err != nil {
			if !errors.Is(err, internal.ErrNotFound) {
				return nil, err
			}

			logging.S.Debugf("creating placeholder user %q", input.User)

			// user does not exist yet, create a placeholder
			user = &models.Identity{
				Name:       input.User,
				ProviderID: providerID,
				Kind:       models.UserKind,
			}

			if err := data.CreateIdentity(db, user); err != nil {
				return nil, err
			}
		}

		id = uid.NewIdentityPolymorphicID(user.ID)

	case input.Group != "":
		group, err := data.GetGroup(db, data.ByName(input.Group))
		if err != nil {
			if !errors.Is(err, internal.ErrNotFound) {
				return nil, err
			}

			logging.S.Debugf("creating placeholder group %q", input.Group)

			// group does not exist yet, create a placeholder
			group = &models.Group{
				Name:       input.Group,
				ProviderID: providerID,
			}

			if err := data.CreateGroup(db, group); err != nil {
				return nil, err
			}
		}

		id = uid.NewGroupPolymorphicID(group.ID)

	case input.Machine != "":
		machine, err := data.GetIdentity(db, data.ByName(input.Machine))
		if err != nil {
			if !errors.Is(err, internal.ErrNotFound) {
				return nil, err
			}

			logging.S.Debugf("creating machine %q", input.Machine)

			infraProvider, err := data.GetProvider(db, data.ByName(models.InternalInfraProviderName))
			if err != nil {
				return nil, fmt.Errorf("machine internal provider: %w", err)
			}

			// machine does not exist, create it
			machine = &models.Identity{
				Name:       input.Machine,
				Kind:       models.MachineKind,
				ProviderID: infraProvider.ID,
			}

			if err := data.CreateIdentity(db, machine); err != nil {
				return nil, err
			}
		}

		id = uid.NewIdentityPolymorphicID(machine.ID)

	default:
		return nil, errors.New("invalid grant: missing identity")
	}

	if len(input.Role) == 0 {
		input.Role = models.BasePermissionConnect
	}

	grant, err := data.GetGrant(db, data.BySubject(id), data.ByResource(input.Resource), data.ByPrivilege(input.Role))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		grant = &models.Grant{
			Subject:   id,
			Resource:  input.Resource,
			Privilege: input.Role,
			CreatedBy: models.CreatedByConfig,
		}

		if err := data.CreateGrant(db, grant); err != nil {
			return nil, err
		}
	}

	return grant, nil
}

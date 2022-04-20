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
	"github.com/infrahq/secrets"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
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
	Role     string `mapstructure:"role" validate:"required"`
	Resource string `mapstructure:"resource" validate:"required"`
}

type Config struct {
	Providers []Provider `mapstructure:"providers" validate:"dive"`
	Grants    []Grant    `mapstructure:"grants" validate:"dive"`
}

type KeyProvider struct {
	Kind   string      `mapstructure:"kind" validate:"required"`
	Config interface{} // contains secret-provider-specific config
}

type nativeKeyProviderConfig struct {
	SecretProvider string `mapstructure:"secretProvider"`
}

type AWSConfig struct {
	Endpoint        string `mapstructure:"endpoint" validate:"required"`
	Region          string `mapstructure:"region" validate:"required"`
	AccessKeyID     string `mapstructure:"accessKeyID" validate:"required"`
	SecretAccessKey string `mapstructure:"secretAccessKey" validate:"required"`
}

type AWSKMSConfig struct {
	AWSConfig `mapstructure:",squash"`

	EncryptionAlgorithm string `mapstructure:"encryptionAlgorithm"`
	// aws tags?
}

type AWSSecretsManagerConfig struct {
	AWSConfig `mapstructure:",squash"`
}

type AWSSSMConfig struct {
	AWSConfig `mapstructure:",squash"`
	KeyID     string `mapstructure:"keyID" validate:"required"` // KMS key to use for decryption
}

type GenericConfig struct {
	Base64           bool `mapstructure:"base64"`
	Base64URLEncoded bool `mapstructure:"base64UrlEncoded"`
	Base64Raw        bool `mapstructure:"base64Raw"`
}

type FileConfig struct {
	GenericConfig `mapstructure:",squash"`
	Path          string `mapstructure:"path" validate:"required"`
}

type KubernetesConfig struct {
	Namespace string `mapstructure:"namespace"`
}

type VaultConfig struct {
	TransitMount string `mapstructure:"transitMount"`              // mounting point. defaults to /transit
	SecretMount  string `mapstructure:"secretMount"`               // mounting point. defaults to /secret
	Token        string `mapstructure:"token" validate:"required"` // vault token... should authenticate as machine to vault instead?
	Namespace    string `mapstructure:"namespace"`
	Address      string `mapstructure:"address" validate:"required"`
}

func importKeyProviders(
	cfg []KeyProvider,
	storage map[string]secrets.SecretStorage,
	keys map[string]secrets.SymmetricKeyProvider,
) error {
	var err error

	// default to file-based native secret provider
	keys["native"] = secrets.NewNativeKeyProvider(storage["file"])

	for _, keyConfig := range cfg {
		switch keyConfig.Kind {
		case "native":
			cfg, ok := keyConfig.Config.(nativeKeyProviderConfig)
			if !ok {
				return fmt.Errorf("expected key config to be nativeKeyProviderConfig, but was %t", keyConfig.Config)
			}

			storageProvider, found := storage[cfg.SecretProvider]
			if !found {
				return fmt.Errorf("secret storage name %q not found", cfg.SecretProvider)
			}

			sp := secrets.NewNativeKeyProvider(storageProvider)
			keys[keyConfig.Kind] = sp
		case "awskms":
			cfg, ok := keyConfig.Config.(AWSKMSConfig)
			if !ok {
				return fmt.Errorf("expected key config to be AWSKMSConfig, but was %t", keyConfig.Config)
			}

			cfg.AccessKeyID, err = secrets.GetSecret(cfg.AccessKeyID, storage)
			if err != nil {
				return fmt.Errorf("getting secret for awskms accessKeyID: %w", err)
			}

			cfg.SecretAccessKey, err = secrets.GetSecret(cfg.SecretAccessKey, storage)
			if err != nil {
				return fmt.Errorf("getting secret for awskms secretAccessKey: %w", err)
			}

			kmsCfg := secrets.NewAWSKMSConfig()
			kmsCfg.AWSConfig.AccessKeyID = cfg.AccessKeyID
			kmsCfg.AWSConfig.Endpoint = cfg.Endpoint
			kmsCfg.AWSConfig.Region = cfg.Region
			kmsCfg.AWSConfig.SecretAccessKey = cfg.SecretAccessKey
			if len(cfg.EncryptionAlgorithm) > 0 {
				kmsCfg.EncryptionAlgorithm = cfg.EncryptionAlgorithm
			}

			sp, err := secrets.NewAWSKMSSecretProviderFromConfig(kmsCfg)
			if err != nil {
				return err
			}

			keys[keyConfig.Kind] = sp
		case "vault":
			cfg, ok := keyConfig.Config.(VaultConfig)
			if !ok {
				return fmt.Errorf("expected key config to be VaultConfig, but was %t", keyConfig.Config)
			}

			cfg.Token, err = secrets.GetSecret(cfg.Token, storage)
			if err != nil {
				return err
			}

			vcfg := secrets.NewVaultConfig()
			if len(cfg.TransitMount) > 0 {
				vcfg.TransitMount = cfg.TransitMount
			}
			if len(cfg.SecretMount) > 0 {
				vcfg.SecretMount = cfg.SecretMount
			}
			if len(cfg.Address) > 0 {
				vcfg.Address = cfg.Address
			}
			vcfg.Token = cfg.Token
			vcfg.Namespace = cfg.Namespace

			sp, err := secrets.NewVaultSecretProviderFromConfig(vcfg)
			if err != nil {
				return err
			}

			keys[keyConfig.Kind] = sp
		}
	}

	return nil
}

func (kp *KeyProvider) PrepareForDecode(data interface{}) error {
	kind := getKindFromUnstructured(data)
	switch kind {
	case "vault":
		kp.Config = VaultConfig{}
	case "awskms":
		kp.Config = AWSKMSConfig{}
	case "native":
		kp.Config = nativeKeyProviderConfig{}
	default:
		// unknown kind error is handled by import importKeyProviders
	}

	return nil
}

type SecretProvider struct {
	Kind   string      `mapstructure:"kind"`
	Name   string      `mapstructure:"name"`
	Config interface{} // contains secret-provider-specific config
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

func importSecrets(cfg []SecretProvider, storage map[string]secrets.SecretStorage) error {
	loadSecretConfig := func(secret SecretProvider) (err error) {
		name := secret.Name
		if len(name) == 0 {
			name = secret.Kind
		}

		if _, found := storage[name]; found {
			return fmt.Errorf("duplicate secret configuration for %q, please provide a unique name for this secret configuration", name)
		}

		switch secret.Kind {
		case "vault":
			cfg, ok := secret.Config.(VaultConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be VaultConfig, but was %t", secret.Config)
			}

			cfg.Token, err = secrets.GetSecret(cfg.Token, storage)
			if err != nil {
				return err
			}

			vcfg := secrets.NewVaultConfig()
			if len(cfg.TransitMount) > 0 {
				vcfg.TransitMount = cfg.TransitMount
			}
			if len(cfg.SecretMount) > 0 {
				vcfg.SecretMount = cfg.SecretMount
			}
			if len(cfg.Address) > 0 {
				vcfg.Address = cfg.Address
			}
			vcfg.Token = cfg.Token
			vcfg.Namespace = cfg.Namespace

			vault, err := secrets.NewVaultSecretProviderFromConfig(vcfg)
			if err != nil {
				return fmt.Errorf("creating vault provider: %w", err)
			}

			storage[name] = vault
		case "awsssm":
			cfg, ok := secret.Config.(AWSSSMConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be AWSSSMConfig, but was %t", secret.Config)
			}

			cfg.AccessKeyID, err = secrets.GetSecret(cfg.AccessKeyID, storage)
			if err != nil {
				return err
			}

			cfg.SecretAccessKey, err = secrets.GetSecret(cfg.SecretAccessKey, storage)
			if err != nil {
				return err
			}

			ssmcfg := secrets.AWSSSMConfig{
				AWSConfig: secrets.AWSConfig{
					Endpoint:        cfg.Endpoint,
					Region:          cfg.Region,
					AccessKeyID:     cfg.AccessKeyID,
					SecretAccessKey: cfg.SecretAccessKey,
				},
				KeyID: cfg.KeyID,
			}

			ssm, err := secrets.NewAWSSSMSecretProviderFromConfig(ssmcfg)
			if err != nil {
				return fmt.Errorf("creating aws ssm: %w", err)
			}

			storage[name] = ssm
		case "awssecretsmanager":
			cfg, ok := secret.Config.(AWSSecretsManagerConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be AWSSecretsManagerConfig, but was %t", secret.Config)
			}

			cfg.AccessKeyID, err = secrets.GetSecret(cfg.AccessKeyID, storage)
			if err != nil {
				return err
			}

			cfg.SecretAccessKey, err = secrets.GetSecret(cfg.SecretAccessKey, storage)
			if err != nil {
				return err
			}

			smCfg := secrets.AWSSecretsManagerConfig{
				AWSConfig: secrets.AWSConfig{
					Endpoint:        cfg.Endpoint,
					Region:          cfg.Region,
					AccessKeyID:     cfg.AccessKeyID,
					SecretAccessKey: cfg.SecretAccessKey,
				},
			}

			sm, err := secrets.NewAWSSecretsManagerFromConfig(smCfg)
			if err != nil {
				return fmt.Errorf("creating aws sm: %w", err)
			}

			storage[name] = sm
		case "kubernetes":
			cfg, ok := secret.Config.(KubernetesConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be KubernetesConfig, but was %t", secret.Config)
			}

			kcfg := secrets.NewKubernetesConfig()
			if len(cfg.Namespace) > 0 {
				kcfg.Namespace = cfg.Namespace
			}

			k8s, err := secrets.NewKubernetesSecretProviderFromConfig(kcfg)
			if err != nil {
				return fmt.Errorf("creating k8s secret provider: %w", err)
			}

			storage[name] = k8s
		case "env":
			cfg, ok := secret.Config.(GenericConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be GenericConfig, but was %t", secret.Config)
			}

			gcfg := secrets.GenericConfig{
				Base64:           cfg.Base64,
				Base64URLEncoded: cfg.Base64URLEncoded,
				Base64Raw:        cfg.Base64Raw,
			}

			f := secrets.NewEnvSecretProviderFromConfig(gcfg)
			storage[name] = f
		case "file":
			cfg, ok := secret.Config.(FileConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be FileConfig, but was %t", secret.Config)
			}

			fcfg := secrets.FileConfig{
				GenericConfig: secrets.GenericConfig{
					Base64:           cfg.Base64,
					Base64URLEncoded: cfg.Base64URLEncoded,
					Base64Raw:        cfg.Base64Raw,
				},
				Path: cfg.Path,
			}

			f := secrets.NewFileSecretProviderFromConfig(fcfg)
			storage[name] = f
		case "plaintext", "":
			cfg, ok := secret.Config.(GenericConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be GenericConfig, but was %t", secret.Config)
			}

			gcfg := secrets.GenericConfig{
				Base64:           cfg.Base64,
				Base64URLEncoded: cfg.Base64URLEncoded,
				Base64Raw:        cfg.Base64Raw,
			}

			f := secrets.NewPlainSecretProviderFromConfig(gcfg)
			storage[name] = f
		default:
			return fmt.Errorf("unknown secret provider type %q", secret.Kind)
		}

		return nil
	}

	// check all base types first
	for _, secret := range cfg {
		if !isABaseSecretStorageKind(secret.Kind) {
			continue
		}

		if err := loadSecretConfig(secret); err != nil {
			return err
		}
	}

	if err := loadDefaultSecretConfig(storage); err != nil {
		return err
	}

	// now load non-base types which might depend on them.
	for _, secret := range cfg {
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
func loadDefaultSecretConfig(storage map[string]secrets.SecretStorage) error {
	// set up the default supported types
	if _, found := storage["env"]; !found {
		f := secrets.NewEnvSecretProviderFromConfig(secrets.GenericConfig{})
		storage["env"] = f
	}

	if _, found := storage["file"]; !found {
		f := secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{})
		storage["file"] = f
	}

	if _, found := storage["plaintext"]; !found {
		f := secrets.NewPlainSecretProviderFromConfig(secrets.GenericConfig{})
		storage["plaintext"] = f
	}

	if _, found := storage["kubernetes"]; !found {
		// only setup k8s automatically if KUBERNETES_SERVICE_HOST is defined; ie, we are in the clustes.
		if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
			k8s, err := secrets.NewKubernetesSecretProviderFromConfig(secrets.NewKubernetesConfig())
			if err != nil {
				return fmt.Errorf("creating k8s secret provider: %w", err)
			}

			storage["kubernetes"] = k8s
		}
	}

	return nil
}

// PrepareForDecode prepares the SecretProvider for mapstructure.Decode by
// setting a concrete type for the config based on the kind. Failures to decode
// will be handled by mapstructure, or by importSecrets.
func (sp *SecretProvider) PrepareForDecode(data interface{}) error {
	kind := getKindFromUnstructured(data)
	switch kind {
	case "vault":
		sp.Config = VaultConfig{}
	case "awsssm":
		sp.Config = AWSSSMConfig{}
	case "awssecretsmanager":
		sp.Config = AWSSecretsManagerConfig{}
	case "kubernetes":
		sp.Config = KubernetesConfig{}
	case "env":
		sp.Config = GenericConfig{}
	case "file":
		sp.Config = FileConfig{}
	case "plaintext", "":
		sp.Kind = "plaintext"
		sp.Config = GenericConfig{}
	default:
		// unknown kind error is handled by importSecrets
	}

	return nil
}

func getKindFromUnstructured(data interface{}) string {
	switch raw := data.(type) {
	case map[string]interface{}:
		if v, ok := raw["kind"].(string); ok {
			return v
		}
	case map[interface{}]interface{}:
		if v, ok := raw["kind"].(string); ok {
			return v
		}
	case *SecretProvider:
		return raw.Kind
	}
	return ""
}

// setupInternalInfraIdentityProvider creates the internal identity provider where local identities are stored
func (s *Server) setupInternalInfraIdentityProvider() error {
	provider, err := data.GetProvider(s.db, data.ByName(models.InternalInfraProviderName))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return fmt.Errorf("setup infra provider: %w", err)
		}

		provider = &models.Provider{
			Name:      models.InternalInfraProviderName,
			CreatedBy: models.CreatedBySystem,
		}

		if err := data.CreateProvider(s.db, provider); err != nil {
			return err
		}
	}

	s.InternalProvider = provider

	return nil
}

// setupInternalIdentity creates built-in identites for the internal identity provider
func (s *Server) setupInternalInfraIdentity(name, role string) (*models.Identity, error) {
	id, err := data.GetIdentity(s.db, data.ByName(name))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, fmt.Errorf("get identity: %w", err)
		}

		id = &models.Identity{
			Name:       name,
			Kind:       models.MachineKind,
			LastSeenAt: time.Now().UTC(),
		}

		err = data.CreateIdentity(s.db, id)
		if err != nil {
			return nil, fmt.Errorf("create identity: %w", err)
		}

		_, err = data.CreateProviderUser(s.db, data.InfraProvider(s.db), id)
		if err != nil {
			return nil, fmt.Errorf("create identity: %w", err)
		}

		grant := &models.Grant{
			Subject:   id.PolyID(),
			Privilege: role,
			Resource:  models.InternalInfraProviderName,
		}

		err = data.CreateGrant(s.db, grant)
		if err != nil {
			return nil, fmt.Errorf("create grant: %w", err)
		}
	}

	if s.InternalIdentities == nil {
		s.InternalIdentities = make(map[string]*models.Identity)
	}

	s.InternalIdentities[name] = id

	return id, nil
}

func (s *Server) importAccessKeys() error {
	type key struct {
		Secret string
		Role   string
	}

	keys := map[string]key{
		models.InternalInfraAdminIdentityName: {
			Secret: s.options.AdminAccessKey,
			Role:   models.InfraAdminRole,
		},
		models.InternalInfraConnectorIdentityName: {
			Secret: s.options.AccessKey,
			Role:   models.InfraConnectorRole,
		},
	}

	for k, v := range keys {
		id, err := s.setupInternalInfraIdentity(k, v.Role)
		if err != nil {
			return fmt.Errorf("setup built-in: %w", err)
		}

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

		name := fmt.Sprintf("default-%s-access-key", k)

		accessKey, err := data.GetAccessKey(s.db, data.ByIssuedFor(id.ID))
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
			Name:       name,
			KeyID:      parts[0],
			Secret:     parts[1],
			IssuedFor:  id.ID,
			ProviderID: data.InfraProvider(s.db).ID,
			ExpiresAt:  time.Now().Add(math.MaxInt64).UTC(),
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

	for _, g := range grants {
		grant, err := loadGrant(db, g)
		if err != nil {
			return err
		}

		toKeep = append(toKeep, grant.ID)
	}

	// remove _all_ grants previously defined by config
	err := data.DeleteGrants(db, data.ByNotIDs(toKeep), data.CreatedBy(models.CreatedByConfig))
	if err != nil {
		return err
	}

	return nil
}

func loadGrant(db *gorm.DB, input Grant) (*models.Grant, error) {
	var id uid.PolymorphicID

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
				Name: input.User,
				Kind: models.UserKind,
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
				Name: input.Group,
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

			// machine does not exist, create it
			machine = &models.Identity{
				Name: input.Machine,
				Kind: models.MachineKind,
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

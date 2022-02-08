package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/config"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/secrets"
)

type KeyProvider struct {
	Kind   string      `yaml:"kind" validate:"required"`
	Config interface{} // contains secret-provider-specific config
}

var _ yaml.Unmarshaler = &KeyProvider{}

type nativeSecretProviderConfig struct {
	SecretStorageName string `yaml:"secretProvider"`
}

func (s *Server) importConfig() error {
	if s.options.Import == nil {
		logging.L.Debug("Skipping config import, import not specified")
		return nil
	}

	adminAccessKey, err := s.GetSecret(s.options.AdminAccessKey)
	if err != nil {
		return fmt.Errorf("importing config: %w", err)
	}

	client := &api.Client{
		Url:   "https://localhost:443",
		Token: adminAccessKey,
		Http: http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					//nolint:gosec // purposely set for localhost
					InsecureSkipVerify: true,
				},
			},
		},
	}

	return config.Import(client, *s.options.Import, true)
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

			cfg.AccessKeyID, err = s.GetSecret(cfg.AccessKeyID)
			if err != nil {
				return fmt.Errorf("getting secret for awskms accessKeyID: %w", err)
			}

			cfg.SecretAccessKey, err = s.GetSecret(cfg.SecretAccessKey)
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

			cfg.Token, err = s.GetSecret(cfg.Token)
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

			cfg.Token, err = s.GetSecret(cfg.Token)
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

			cfg.AccessKeyID, err = s.GetSecret(cfg.AccessKeyID)
			if err != nil {
				return err
			}

			cfg.SecretAccessKey, err = s.GetSecret(cfg.SecretAccessKey)
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

			cfg.AccessKeyID, err = s.GetSecret(cfg.AccessKeyID)
			if err != nil {
				return err
			}

			cfg.SecretAccessKey, err = s.GetSecret(cfg.SecretAccessKey)
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

func (s *Server) importAccessKeys() error {
	type key struct {
		Secret      string
		Permissions []string
	}

	keys := map[string]key{
		"admin": {
			Secret: s.options.AdminAccessKey,
			Permissions: []string{
				string(access.PermissionAllInfra),
			},
		},
		"engine": {
			Secret: s.options.AccessKey,
			Permissions: []string{
				string(access.PermissionUserRead),
				string(access.PermissionGroupRead),
				string(access.PermissionGrantRead),
				string(access.PermissionDestinationRead),
				string(access.PermissionDestinationCreate),
				string(access.PermissionDestinationUpdate),
			},
		},
	}

	for k, v := range keys {
		if v.Secret == "" {
			logging.S.Debugf("%s: secret not set; skipping", k)
			continue
		}

		raw, err := s.GetSecret(v.Secret)
		if err != nil && !errors.Is(err, secrets.ErrNotFound) {
			return fmt.Errorf("%s secret: %w", k, err)
		}

		if raw == "" {
			logging.S.Debugf("%s: secret value not set; skipping", k)
			continue
		}

		ak, err := data.LookupAccessKey(s.db, raw)
		if err != nil {
			if !errors.Is(err, internal.ErrNotFound) {
				return fmt.Errorf("%s lookup: %w", k, err)
			}
		}

		parts := strings.Split(raw, ".")
		if len(parts) < 2 {
			return fmt.Errorf("%s format: %w", k, err)
		}

		if ak != nil {
			sum := sha256.Sum256([]byte(parts[1]))

			// if token name, permissions, and secret checksum all match the input, skip recreating the token
			if ak.Name == k && ak.Permissions == strings.Join(v.Permissions, " ") && subtle.ConstantTimeCompare(ak.SecretChecksum, sum[:]) != 1 {
				logging.S.Debugf("%s: skip recreating token", k)
				continue
			}

			err = data.DeleteAccessKeys(s.db, data.ByName(k))
			if err != nil {
				return fmt.Errorf("%s delete: %w", k, err)
			}
		}

		token := &models.AccessKey{
			Name:        k,
			Key:         parts[0],
			Secret:      parts[1],
			Permissions: strings.Join(v.Permissions, " "),
			ExpiresAt:   time.Now().Add(math.MaxInt64),
		}
		if _, err := data.CreateAccessKey(s.db, token); err != nil {
			return fmt.Errorf("%s create: %w", k, err)
		}
	}

	return nil
}

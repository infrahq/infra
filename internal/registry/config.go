package registry

import (
	"errors"
	"fmt"
	"os"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/secrets"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

type ConfigOkta struct {
	APIToken string `yaml:"apiToken"`
}

type ConfigIdentityProvider struct {
	Kind         string      `yaml:"kind"`
	Domain       string      `yaml:"domain"`
	ClientID     string      `yaml:"clientID"`
	ClientSecret string      `yaml:"clientSecret"`
	Config       interface{} // contains identity-provider-specific config
}

type baseConfigIdentityProvider struct {
	Kind         string `yaml:"kind"`
	Domain       string `yaml:"domain"`
	ClientID     string `yaml:"clientID"`
	ClientSecret string `yaml:"clientSecret"`
}

var _ yaml.Unmarshaler = &ConfigIdentityProvider{}

func (idp *ConfigIdentityProvider) UnmarshalYAML(unmarshal func(interface{}) error) error {
	tmp := &baseConfigIdentityProvider{}

	if err := unmarshal(&tmp); err != nil {
		return fmt.Errorf("unmarshalling secret provider: %w", err)
	}

	idp.Kind = tmp.Kind
	idp.Domain = tmp.Domain
	idp.ClientID = tmp.ClientID
	idp.ClientSecret = tmp.ClientSecret

	switch tmp.Kind {
	case ProviderKindOkta:
		o := ConfigOkta{}
		if err := unmarshal(&o); err != nil {
			return fmt.Errorf("unmarshal yaml: %w", err)
		}

		idp.Config = o
	default:
		return fmt.Errorf("%w: %q, expected %s", ErrInvalidKind, tmp.Kind, ProviderKindOkta)
	}

	return nil
}

type ConfigDestination struct {
	Name       string   `yaml:"name"`
	Namespaces []string `yaml:"namespaces"` // optional in the case of a cluster-role
}

type ConfigRoleKubernetes struct {
	Name         string              `yaml:"name"`
	Kind         string              `yaml:"kind"`
	Destinations []ConfigDestination `yaml:"destinations"`
}

type ConfigGroupMapping struct {
	Name     string                 `yaml:"name"`
	Provider string                 `yaml:"provider"`
	Roles    []ConfigRoleKubernetes `yaml:"roles"`
}

type ConfigUserMapping struct {
	Email  string                 `yaml:"email"`
	Roles  []ConfigRoleKubernetes `yaml:"roles"`
	Groups []string               `yaml:"groups"`
}

type ConfigSecretProvider struct {
	Kind   string      `yaml:"kind"`
	Name   string      `yaml:"name"` // optional
	Config interface{} // contains secret-provider-specific config
}

type simpleConfigSecretProvider struct {
	Kind string `yaml:"kind"`
	Name string `yaml:"name"`
}

// ensure ConfigSecretProvider implements yaml.Unmarshaller for the custom config field support
var _ yaml.Unmarshaler = &ConfigSecretProvider{}

func (sp *ConfigSecretProvider) UnmarshalYAML(unmarshal func(interface{}) error) error {
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

type Config struct {
	Secrets   []ConfigSecretProvider   `yaml:"secrets"`
	Providers []ConfigIdentityProvider `yaml:"providers"`
	Groups    []ConfigGroupMapping     `yaml:"groups"`
	Users     []ConfigUserMapping      `yaml:"users"`
}

// this config is loaded at start-up and re-applied when Infra's state changes (ie. a user is added)
var initialConfig Config

func ImportProviders(db *gorm.DB, providers []ConfigIdentityProvider) error {
	var idsToKeep []string

	for _, p := range providers {
		// check if we are about to override an existing provider
		var existing Provider
		if err := db.First(&existing, &Provider{Kind: p.Kind}).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// expected for new records
			} else {
				return fmt.Errorf("existing provider lookup: %w", err)
			}
		}

		if existing.Id != "" {
			logging.L.Warn("overriding existing okta provider settings with configuration settings")
		}

		if p.Domain == "" {
			return fmt.Errorf("no domain set on provider: %s", p.Kind)
		}

		if p.ClientID == "" {
			return fmt.Errorf("no clientID set on provider: %s", p.Kind)
		}

		if p.ClientSecret == "" {
			return fmt.Errorf("no clientSecret set on provider: %s", p.Kind)
		}

		var provider Provider
		if err := db.FirstOrCreate(&provider, &Provider{Kind: p.Kind}).Error; err != nil {
			return fmt.Errorf("create config provider: %w", err)
		}

		provider.ClientID = p.ClientID
		provider.Domain = p.Domain
		provider.ClientSecret = p.ClientSecret

		switch p.Kind {
		case ProviderKindOkta:
			cfg, ok := p.Config.(ConfigOkta)
			if !ok {
				return fmt.Errorf("expected provider config to be Okta, but was %t", p.Config)
			}

			if cfg.APIToken == "" {
				return fmt.Errorf("no apiToken set on provider: %s", p.Kind)
			}

			// API token and client secret will be validated to exist when they are used
			provider.APIToken = cfg.APIToken

			if err := db.Save(&provider).Error; err != nil {
				return fmt.Errorf("save provider: %w", err)
			}

			idsToKeep = append(idsToKeep, provider.Id)
		case "":
			logging.S.Errorf("skipping a provider with no kind set in configuration")
		default:
			logging.S.Errorf("skipping invalid provider kind in configuration: %s", p.Kind)
		}
	}

	if len(idsToKeep) == 0 {
		logging.L.Debug("no valid providers found in configuration, ensure the required fields are specified correctly")
		// clear the providers
		return db.Where("1 = 1").Delete(&Provider{}).Error
	}

	var toDelete []Provider
	if err := db.Not(idsToKeep).Find(&toDelete).Error; err != nil {
		return fmt.Errorf("find providers delete: %w", err)
	}

	if len(toDelete) > 0 {
		if err := db.Delete(&toDelete).Error; err != nil {
			return fmt.Errorf("delete providers: %w", err)
		}
	}

	return nil
}

func ApplyGroupMappings(db *gorm.DB, groups []ConfigGroupMapping) (modifiedRoleIDs []string, err error) {
	for _, g := range groups {
		// get the provider from the datastore that this group specifies
		var provider Provider
		// Assumes that only one kind of each provider can exist
		provReadErr := db.Where(&Provider{Kind: g.Provider}).First(&provider).Error
		if provReadErr != nil {
			if errors.Is(provReadErr, gorm.ErrRecordNotFound) {
				// skip this provider, it will need to be added in the config and re-applied
				logging.S.Debugf("skipping group '%s' with provider '%s' in config that does not exist", g.Name, g.Provider)
				continue
			}

			return nil, fmt.Errorf("group read provider: %w", provReadErr)
		}

		var group Group

		grpReadErr := db.Preload("Users").Where(&Group{Name: g.Name, ProviderId: provider.Id}).First(&group).Error
		if grpReadErr != nil {
			if errors.Is(grpReadErr, gorm.ErrRecordNotFound) {
				// skip this group, if they're created these roles will be added later
				logging.L.Debug("skipping group in config import that has not yet been provisioned")
				continue
			}

			return nil, fmt.Errorf("group read: %w", grpReadErr)
		}

		// import the roles on this group from the datastore
		var roles []Role

		var grpRoleIDs []string

		roles, grpRoleIDs, err = importRoles(db, g.Roles)
		if err != nil {
			return nil, fmt.Errorf("group import roles: %w", err)
		}

		modifiedRoleIDs = append(modifiedRoleIDs, grpRoleIDs...)

		// add the new group associations to the roles
		for i, role := range roles {
			if db.Model(&group).Where(&Role{Id: role.Id}).Association("Roles").Count() == 0 {
				if err = db.Model(&group).Where(&Role{Id: role.Id}).Association("Roles").Append(&roles[i]); err != nil {
					return nil, fmt.Errorf("group role assocations: %w", err)
				}
			}
		}
	}

	return modifiedRoleIDs, nil
}

func ApplyUserMappings(db *gorm.DB, users []ConfigUserMapping) (modifiedRoleIDs []string, err error) {
	for _, u := range users {
		var user User

		usrReadErr := db.Where(&User{Email: u.Email}).First(&user).Error
		if usrReadErr != nil {
			if errors.Is(usrReadErr, gorm.ErrRecordNotFound) {
				// skip this user, if they're created these roles will be added later
				logging.L.Debug("skipping user in config import that has not yet been provisioned")
				continue
			}

			return nil, fmt.Errorf("read user: %w", usrReadErr)
		}

		var roles []Role

		// add direct user to role mappings
		var usrRoleIDs []string

		roles, usrRoleIDs, err = importRoles(db, u.Roles)
		if err != nil {
			return nil, fmt.Errorf("import user roles: %w", err)
		}

		modifiedRoleIDs = append(modifiedRoleIDs, usrRoleIDs...)

		// for all roles attached to this user update their user associations now that we have made sure they exist
		// important: do not create the association on the user, that runs an upsert that creates a concurrent write because User.AfterCreate() calls this function
		for i, role := range roles {
			if db.Model(&user).Where(&Role{Id: role.Id}).Association("Roles").Count() == 0 {
				if err = db.Model(&user).Where(&Role{Id: role.Id}).Association("Roles").Append(&roles[i]); err != nil {
					return nil, fmt.Errorf("user role associations: %w", err)
				}
			}
		}
	}

	return modifiedRoleIDs, nil
}

// ImportRoleMappings iterates over user and group config and applies a role mapping to them
func ImportRoleMappings(db *gorm.DB, groups []ConfigGroupMapping, users []ConfigUserMapping) error {
	groupRoleIDs, err := ApplyGroupMappings(db, groups)
	if err != nil {
		return fmt.Errorf("apply group mappings: %w", err)
	}

	userRoleIDs, err := ApplyUserMappings(db, users)
	if err != nil {
		return fmt.Errorf("apply user mappings: %w", err)
	}

	var roleIDs []string
	roleIDs = append(roleIDs, groupRoleIDs...)
	roleIDs = append(roleIDs, userRoleIDs...)

	var rolesRemoved []Role
	if err := db.Not(roleIDs).Find(&rolesRemoved).Error; err != nil {
		return fmt.Errorf("find roles removed in config: %w", err)
	}

	if len(rolesRemoved) > 0 {
		if err := db.Delete(rolesRemoved).Error; err != nil {
			return fmt.Errorf("delete config removed role: %w", err)
		}
	}

	logging.S.Debugf("importing configuration removed %d role(s)", len(rolesRemoved))

	return nil
}

// importConfig tries to import all valid fields in a config file and removes old config
func (r *Registry) importConfig(bs []byte) error {
	var config Config
	if err := yaml.Unmarshal(bs, &config); err != nil {
		return err
	}

	initialConfig = config

	if err := r.configureSecrets(config); err != nil {
		return fmt.Errorf("secrets config: %w", err)
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := ImportProviders(tx, config.Providers); err != nil {
			return err
		}

		return ImportRoleMappings(tx, config.Groups, config.Users)
	})
}

// import roles creates roles specified in the config, or updates their associations
func importRoles(db *gorm.DB, roles []ConfigRoleKubernetes) (rolesImported []Role, importedRoleIDs []string, err error) {
	for _, r := range roles {
		if r.Name == "" {
			logging.L.Error("invalid role found in configuration, name is a required field")
			continue
		}

		if r.Kind == "" {
			logging.L.Error("invalid role found in configuration, kind is a required field")
			continue
		}

		if r.Kind != RoleKindKubernetesClusterRole && r.Kind != RoleKindKubernetesRole {
			logging.L.Error("only 'role' and 'cluster-role' are valid role kinds, found: " + r.Kind)
			continue
		}

		for _, destination := range r.Destinations {
			if r.Kind == RoleKindKubernetesRole && len(destination.Namespaces) == 0 {
				logging.L.Error(r.Name + " requires at least one namespace to be specified for the cluster " + destination.Name)
				continue
			}

			var dest Destination

			destErr := db.Where(&Destination{Name: destination.Name}).First(&dest).Error
			if destErr != nil {
				if errors.Is(destErr, gorm.ErrRecordNotFound) {
					// when a destination is added then the config import will be retried, skip for now
					logging.L.Debug("skipping role binding for destination in config import that has not yet been discovered")
					continue
				}

				return nil, nil, fmt.Errorf("find role destination: %w", destErr)
			}

			if len(destination.Namespaces) > 0 {
				for _, namespace := range destination.Namespaces {
					var role Role
					if err = db.FirstOrCreate(&role, &Role{Name: r.Name, Kind: r.Kind, Namespace: namespace, DestinationId: dest.Id}).Error; err != nil {
						return nil, nil, fmt.Errorf("group read provider: %w", err)
					}

					rolesImported = append(rolesImported, role)
					importedRoleIDs = append(importedRoleIDs, role.Id)
				}
			} else {
				var role Role
				if err = db.FirstOrCreate(&role, &Role{Name: r.Name, Kind: r.Kind, DestinationId: dest.Id}).Error; err != nil {
					return nil, nil, fmt.Errorf("role find create: %w", err)
				}

				rolesImported = append(rolesImported, role)
				importedRoleIDs = append(importedRoleIDs, role.Id)
			}
		}
	}

	return rolesImported, importedRoleIDs, nil
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

func (r *Registry) configureSecrets(config Config) error {
	if r.secrets == nil {
		r.secrets = map[string]secrets.SecretStorage{}
	}

	loadSecretConfig := func(secret ConfigSecretProvider) (err error) {
		name := secret.Name
		if len(name) == 0 {
			name = secret.Kind
		}

		if _, found := r.secrets[name]; found {
			return fmt.Errorf("duplicate secret configuration for %q, please provide a unique name for this secret configuration", name)
		}

		switch secret.Kind {
		case "vault":
			cfg, ok := secret.Config.(secrets.VaultConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be VaultConfig, but was %t", secret.Config)
			}

			cfg.Token, err = r.GetSecret(cfg.Token)
			if err != nil {
				return err
			}

			vault, err := secrets.NewVaultSecretProviderFromConfig(cfg)
			if err != nil {
				return fmt.Errorf("creating vault provider: %w", err)
			}

			r.secrets[name] = vault
		case "awsssm":
			cfg, ok := secret.Config.(secrets.AWSSSMConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be AWSSSMConfig, but was %t", secret.Config)
			}

			cfg.AccessKeyID, err = r.GetSecret(cfg.AccessKeyID)
			if err != nil {
				return err
			}

			cfg.SecretAccessKey, err = r.GetSecret(cfg.SecretAccessKey)
			if err != nil {
				return err
			}

			ssm, err := secrets.NewAWSSSMSecretProviderFromConfig(cfg)
			if err != nil {
				return fmt.Errorf("creating aws ssm: %w", err)
			}

			r.secrets[name] = ssm
		case "awssecretsmanager":
			cfg, ok := secret.Config.(secrets.AWSSecretsManagerConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be AWSSecretsManagerConfig, but was %t", secret.Config)
			}

			cfg.AccessKeyID, err = r.GetSecret(cfg.AccessKeyID)
			if err != nil {
				return err
			}

			cfg.SecretAccessKey, err = r.GetSecret(cfg.SecretAccessKey)
			if err != nil {
				return err
			}

			sm, err := secrets.NewAWSSecretsManagerFromConfig(cfg)
			if err != nil {
				return fmt.Errorf("creating aws sm: %w", err)
			}

			r.secrets[name] = sm
		case "kubernetes":
			cfg, ok := secret.Config.(secrets.KubernetesConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be KubernetesConfig, but was %t", secret.Config)
			}

			k8s, err := secrets.NewKubernetesSecretProviderFromConfig(cfg)
			if err != nil {
				return fmt.Errorf("creating k8s secret provider: %w", err)
			}

			r.secrets[name] = k8s
		case "env":
			cfg, ok := secret.Config.(secrets.GenericConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be GenericConfig, but was %t", secret.Config)
			}

			f := secrets.NewEnvSecretProviderFromConfig(cfg)
			r.secrets[name] = f
		case "file":
			cfg, ok := secret.Config.(secrets.FileConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be FileConfig, but was %t", secret.Config)
			}

			f := secrets.NewFileSecretProviderFromConfig(cfg)
			r.secrets[name] = f
		case "plaintext", "":
			cfg, ok := secret.Config.(secrets.GenericConfig)
			if !ok {
				return fmt.Errorf("expected secret config to be GenericConfig, but was %t", secret.Config)
			}

			f := secrets.NewPlainSecretProviderFromConfig(cfg)
			r.secrets[name] = f
		default:
			return fmt.Errorf("unknown secret provider type %q", secret.Kind)
		}

		return nil
	}

	// check all base types first
	for _, secret := range config.Secrets {
		if !isABaseSecretStorageKind(secret.Kind) {
			continue
		}

		if err := loadSecretConfig(secret); err != nil {
			return err
		}
	}

	if err := r.loadDefaultSecretConfig(); err != nil {
		return err
	}

	// now load non-base types which might depend on them.
	for _, secret := range config.Secrets {
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
func (r *Registry) loadDefaultSecretConfig() error {
	// set up the default supported types
	if _, found := r.secrets["env"]; !found {
		f := secrets.NewEnvSecretProviderFromConfig(secrets.GenericConfig{})
		r.secrets["env"] = f
	}

	if _, found := r.secrets["file"]; !found {
		f := secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{})
		r.secrets["file"] = f
	}

	if _, found := r.secrets["plaintext"]; !found {
		f := secrets.NewPlainSecretProviderFromConfig(secrets.GenericConfig{})
		r.secrets["plaintext"] = f
	}

	if _, found := r.secrets["kubernetes"]; !found {
		// only setup k8s automatically if KUBERNETES_SERVICE_HOST is defined; ie, we are in the cluster.
		if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
			k8s, err := secrets.NewKubernetesSecretProviderFromConfig(secrets.NewKubernetesConfig())
			if err != nil {
				return fmt.Errorf("creating k8s secret provider: %w", err)
			}

			r.secrets["kubernetes"] = k8s
		}
	}

	return nil
}

package registry

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/logging"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

type ConfigSource struct {
	Type         string `yaml:"type"`
	Domain       string `yaml:"domain"`
	ClientId     string `yaml:"clientId"`
	ClientSecret string `yaml:"clientSecret"`
	ApiToken     string `yaml:"apiToken"`
}

var dashAdminRemover = regexp.MustCompile(`(.*)\-admin(\.okta\.com)`)

func (s *ConfigSource) cleanupDomain() {
	s.Domain = strings.TrimSpace(s.Domain)
	s.Domain = dashAdminRemover.ReplaceAllString(s.Domain, "$1$2")
}

type ConfigMachine struct {
	Name   string `yaml:"name"`
	Kind   string `yaml:"kind"`
	APIKey string `yaml:"key"`
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
	Name   string                 `yaml:"name"`
	Source string                 `yaml:"source"`
	Roles  []ConfigRoleKubernetes `yaml:"roles"`
}

type ConfigUserMapping struct {
	Email  string                 `yaml:"email"`
	Roles  []ConfigRoleKubernetes `yaml:"roles"`
	Groups []string               `yaml:"groups"`
}

type Config struct {
	Sources  []ConfigSource       `yaml:"sources"`
	Machines []ConfigMachine      `yaml:"machines"`
	Groups   []ConfigGroupMapping `yaml:"groups"`
	Users    []ConfigUserMapping  `yaml:"users"`
}

// this config is loaded at start-up and re-applied when the registry state changes (ex: a user is added)
var initialConfig Config

func ImportSources(db *gorm.DB, sources []ConfigSource) error {
	var idsToKeep []string

	for _, s := range sources {
		switch s.Type {
		case SOURCE_TYPE_OKTA:
			// check the domain is specified
			s.cleanupDomain()
			if s.Domain == "" {
				logging.L.Info("domain not set on source \"" + s.Type + "\", import skipped")
			}
			// check if we are about to override an existing source
			var existing Source
			db.First(&existing, &Source{Type: SOURCE_TYPE_OKTA})
			if existing.Id != "" {
				logging.L.Warn("overriding existing okta source settings, only one okta source is supported")
			}
			var source Source
			err := db.FirstOrCreate(&source, &Source{Type: s.Type}).Error
			if err != nil {
				return err
			}

			source.ClientId = s.ClientId
			source.Domain = s.Domain
			// API token and client secret will be validated to exist when they are used
			source.ClientSecret = s.ClientSecret
			source.ApiToken = s.ApiToken

			err = db.Save(&source).Error
			if err != nil {
				return err
			}

			idsToKeep = append(idsToKeep, source.Id)
		default:
			logging.L.Error("skipping invalid source type in configuration: " + s.Type)
		}
	}

	if len(idsToKeep) == 0 {
		logging.L.Debug("no valid sources found in configuration, ensure the required fields are specified correctly")
	}

	if err := db.Where("1 = 1").Not(idsToKeep).Delete(&Source{}).Error; err != nil {
		return err
	}

	return nil
}

func ImportMachines(db *gorm.DB, k8s *kubernetes.Kubernetes, machines []ConfigMachine) error {
	// clear any existing machines, need to do this iteratively so that the information for clearing associated API keys is populated
	var toDelete []Machine
	if err := db.Find(&toDelete).Error; err != nil {
		return err
	}
	for _, m := range toDelete {
		if err := db.Where("1=1").Delete(&m).Error; err != nil {
			return err
		}
	}

	for _, m := range machines {
		switch strings.ToLower(m.Kind) {
		case MACHINE_KIND_API_KEY:
			if strings.ToLower(m.Name) == "default" {
				// this name is used for the default API key that engines use to connect to the registry
				logging.L.Info("cannot import machine API key with the name \"default\", this name is reserved, continuing...")
				continue
			}
			// get the secret API key value from kubernetes
			secretKey, err := k8s.GetSecret(m.APIKey)
			if err != nil {
				logging.L.Error(err.Error())
				logging.L.Info("could not retrieve secret for " + m.APIKey + ", continuing...")
				continue
			}
			if len(secretKey) != API_KEY_LEN {
				logging.L.Info("secret stored at " + m.APIKey + " does not have a valid key length, it must be exactly " + fmt.Sprint(API_KEY_LEN) + " characters")
				logging.L.Info("skipped importing machine: " + m.Name)
				continue
			}

			err = db.Transaction(func(tx *gorm.DB) error {
				var apiKey ApiKey
				tx.First(&apiKey, &ApiKey{Name: m.Name})
				if apiKey.Id != "" {
					return &ErrExistingKey{}
				}

				apiKey.Name = m.Name
				apiKey.Key = secretKey
				err := tx.Create(&apiKey).Error
				if err != nil {
					return err
				}

				apiMachine := Machine{Name: m.Name, Kind: MACHINE_KIND_API_KEY, ApiKeyId: apiKey.Id}
				err = tx.Create(&apiMachine).Error

				return err
			})
			if err != nil {
				switch err.(type) {
				case *ErrExistingKey:
					logging.L.Error(err.Error())
					logging.L.Info("skipped importing " + m.Name + " due to existing API key with the same name")
					continue
				default:
					return err
				}
			}
		}
	}
	return nil
}

func ApplyGroupMappings(db *gorm.DB, configGroups []ConfigGroupMapping) (groupIds []string, err error) {
	for _, g := range configGroups {
		// get the source from the datastore that this group specifies
		var source Source
		// Assumes that only one type of each source can exist
		srcReadErr := db.Where(&Source{Type: g.Source}).First(&source).Error
		if srcReadErr != nil {
			if errors.Is(srcReadErr, gorm.ErrRecordNotFound) {
				// skip this source, it will need to be added in the config and re-applied
				logging.L.Debug("skipping group with source in config that does not exist: " + g.Source)
				continue
			}
			err = srcReadErr
			return
		}

		// import the roles on this group from the datastore
		var roles []Role
		roles, err = importRoles(db, g.Roles)
		if err != nil {
			return
		}

		var group Group
		// Group names must be unique for mapping purposes
		err = db.FirstOrCreate(&group, &Group{Name: g.Name, SourceId: source.Id}).Error
		if err != nil {
			return
		}

		// add the new group associations to the roles
		for _, role := range roles {
			if db.Model(&group).Where(&Role{Id: role.Id}).Association("Roles").Count() == 0 {
				if err = db.Model(&group).Where(&Role{Id: role.Id}).Association("Roles").Append(&role); err != nil {
					return
				}
			}
		}
		groupIds = append(groupIds, group.Id)
	}
	return
}

func ApplyUserMapping(db *gorm.DB, users []ConfigUserMapping) error {
	for _, u := range users {
		var user User
		usrReadErr := db.Where(&User{Email: u.Email}).First(&user).Error
		if usrReadErr != nil {
			if errors.Is(usrReadErr, gorm.ErrRecordNotFound) {
				// skip this user, if they're created these roles will be added later
				logging.L.Debug("skipping user in config import that has not yet been provisioned")
				continue
			}
			return usrReadErr
		}

		// add the user to groups, these declarations can be overriden by external group syncing
		for _, gName := range u.Groups {
			// Assumes that only one group can exist with a given name, regardless of sources
			var group Group
			grpReadErr := db.Where(&Group{Name: gName}).First(&group).Error
			if grpReadErr != nil {
				if errors.Is(grpReadErr, gorm.ErrRecordNotFound) {
					logging.L.Debug("skipping unknown group \"" + gName + "\" on user")
					continue
				}
				return grpReadErr
			}
			if db.Model(&user).Where(&Group{Id: group.Id}).Association("Groups").Count() == 0 {
				if err := db.Model(&user).Where(&Group{Id: group.Id}).Association("Groups").Append(&group); err != nil {
					return err
				}
			}
		}

		// add roles to user
		roles, err := importRoles(db, u.Roles)
		if err != nil {
			return err
		}
		// for all roles attached to this user update their user associations now that we have made sure they exist
		// important: do not create the association on the user, that runs an upsert that creates a concurrent write because User.AfterCreate() calls this function
		for _, role := range roles {
			if db.Model(&user).Where(&Role{Id: role.Id}).Association("Roles").Count() == 0 {
				if err = db.Model(&user).Where(&Role{Id: role.Id}).Association("Roles").Append(&role); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ImportMappings imports the group and user role mappings and removes previously created roles if they no longer exist
func ImportMappings(db *gorm.DB, groups []ConfigGroupMapping, users []ConfigUserMapping) error {
	// gorm blocks global delete by default: https://gorm.io/docs/delete.html#Block-Global-Delete
	if err := db.Where("1 = 1").Delete(&Role{}).Error; err != nil {
		return err
	}

	grpIdsToKeep, err := ApplyGroupMappings(db, groups)
	if err != nil {
		return err
	}

	if len(grpIdsToKeep) == 0 {
		logging.L.Debug("no valid groups found in configuration")
	}

	if err := db.Where("1 = 1").Not(grpIdsToKeep).Delete(&Group{}).Error; err != nil {
		return err
	}

	return ApplyUserMapping(db, users)
}

// ImportConfig tries to import all valid fields in a config file
func ImportConfig(db *gorm.DB, k8s *kubernetes.Kubernetes, bs []byte) error {
	var config Config
	err := yaml.Unmarshal(bs, &config)
	if err != nil {
		return err
	}

	initialConfig = config

	return db.Transaction(func(tx *gorm.DB) error {
		if err = ImportSources(tx, config.Sources); err != nil {
			return err
		}
		if err = ImportMachines(tx, k8s, config.Machines); err != nil {
			return err
		}
		// Need to import of group/user mappings together because they both rely on roles
		if err = ImportMappings(tx, config.Groups, config.Users); err != nil {
			return err
		}
		return nil
	})
}

// import roles creates roles specified in the config, or updates their associations
func importRoles(db *gorm.DB, roles []ConfigRoleKubernetes) ([]Role, error) {
	var rolesImported []Role
	for _, r := range roles {
		if r.Name == "" {
			logging.L.Error("invalid role found in configuration, name is a required field")
			continue
		}
		if r.Kind == "" {
			logging.L.Error("invalid role found in configuration, kind is a required field")
			continue
		}
		if r.Kind != ROLE_KIND_K8S_CLUSTER_ROLE && r.Kind != ROLE_KIND_K8S_ROLE {
			logging.L.Error("only 'role' and 'cluster-role' are valid role kinds, found: " + r.Kind)
			continue
		}
		for _, destination := range r.Destinations {
			if r.Kind == ROLE_KIND_K8S_ROLE && len(destination.Namespaces) == 0 {
				logging.L.Error(r.Name + " requires at least one namespace to be specified for the cluster " + destination.Name)
				continue
			}
			var dest Destination
			err := db.Where(&Destination{Name: destination.Name}).First(&dest).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// when a destination is added then the config import will be retried, skip for now
					logging.L.Debug("skipping destination in config import that has not yet been discovered")
					continue
				}
				return nil, err
			}
			if len(destination.Namespaces) > 0 {
				for _, namespace := range destination.Namespaces {
					var role Role
					if err = db.FirstOrCreate(&role, &Role{Name: r.Name, Kind: r.Kind, Namespace: namespace, DestinationId: dest.Id}).Error; err != nil {
						return nil, err
					}
					rolesImported = append(rolesImported, role)
				}
			} else {
				var role Role
				if err = db.FirstOrCreate(&role, &Role{Name: r.Name, Kind: r.Kind, DestinationId: dest.Id}).Error; err != nil {
					return nil, err
				}
				rolesImported = append(rolesImported, role)
			}

		}
	}
	return rolesImported, nil
}

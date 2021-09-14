package registry

import (
	"errors"

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

type ConfigCluster struct {
	Name       string   `yaml:"name"`
	Namespaces []string `yaml:"namespaces"` // optional in the case of a cluster-role
}

type ConfigRoleKubernetes struct {
	Name     string          `yaml:"name"`
	Kind     string          `yaml:"kind"`
	Clusters []ConfigCluster `yaml:"clusters"`
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
	Sources []ConfigSource       `yaml:"sources"`
	Groups  []ConfigGroupMapping `yaml:"groups"`
	Users   []ConfigUserMapping  `yaml:"users"`
}

// this config is loaded at start-up and re-applied when the registry state changes (ex: a user is added)
var initialConfig Config

func ImportSources(db *gorm.DB, sources []ConfigSource) error {
	var idsToKeep []string

	for _, s := range sources {
		switch s.Type {
		case SOURCE_TYPE_OKTA:
			// check the domain is specified
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
			source.FromConfig = true

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
		logging.L.Info("no valid sources found in configuration, ensure the required fields are specified correctly")
	}

	if err := db.Where(&Role{FromConfig: false}).Not(idsToKeep).Delete(&Source{}).Error; err != nil {
		return err
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
	if err := db.Where(&Role{FromConfig: true}).Delete(Role{}).Error; err != nil {
		return err
	}

	grpIdsToKeep, err := ApplyGroupMappings(db, groups)
	if err != nil {
		return err
	}
	// clean up existing groups which have been removed from the config
	if len(grpIdsToKeep) > 0 {
		err = db.Not(grpIdsToKeep).Delete(Group{}).Error
		if err != nil {
			return err
		}
	} else {
		var groups []Group
		if err := db.Find(&groups).Error; err != nil {
			return err
		}
		if len(groups) > 0 {
			err = db.Delete(groups).Error
			if err != nil {
				return err
			}
		}
		// it is perfectly valid to have a config with no groups, but the user may still want to know this
		logging.L.Debug("no valid groups found in configuration")
	}

	return ApplyUserMapping(db, users)
}

// ImportConfig tries to import all valid fields in a config file
func ImportConfig(db *gorm.DB, bs []byte) error {
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
		// Need to import of group/user mappings together becuase they both rely on roles
		if err = ImportMappings(tx, config.Groups, config.Users); err != nil {
			return err
		}
		return nil
	})
}

// import roles creates roles specified in the config, or updates their assosiations
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
		for _, cluster := range r.Clusters {
			if r.Kind == ROLE_KIND_K8S_ROLE && len(cluster.Namespaces) == 0 {
				logging.L.Error(r.Name + " requires at least one namespace to be specified for the cluster " + cluster.Name)
				continue
			}
			var destination Destination
			err := db.Where(&Destination{Name: cluster.Name}).First(&destination).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// when a destination is added then the config import will be retried, skip for now
					logging.L.Debug("skipping destination in config import that has not yet been discovered")
					continue
				}
				return nil, err
			}
			if len(cluster.Namespaces) > 0 {
				for _, namespace := range cluster.Namespaces {
					var role Role
					if err = db.FirstOrCreate(&role, &Role{Name: r.Name, Kind: r.Kind, Namespace: namespace, DestinationId: destination.Id, FromConfig: true}).Error; err != nil {
						return nil, err
					}
					rolesImported = append(rolesImported, role)
				}
			} else {
				var role Role
				if err = db.FirstOrCreate(&role, &Role{Name: r.Name, Kind: r.Kind, DestinationId: destination.Id, FromConfig: true}).Error; err != nil {
					return nil, err
				}
				rolesImported = append(rolesImported, role)
			}

		}
	}
	return rolesImported, nil
}

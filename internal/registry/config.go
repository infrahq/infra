package registry

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/infrahq/infra/internal/logging"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

type ConfigSource struct {
	Kind         string `yaml:"kind"`
	Domain       string `yaml:"domain"`
	ClientId     string `yaml:"clientId"`
	ClientSecret string `yaml:"clientSecret"`
	APIToken     string `yaml:"apiToken"`
}

var dashAdminRemover = regexp.MustCompile(`(.*)\-admin(\.okta\.com)`)

func (s *ConfigSource) cleanupDomain() {
	s.Domain = strings.TrimSpace(s.Domain)
	s.Domain = dashAdminRemover.ReplaceAllString(s.Domain, "$1$2")
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
	Sources []ConfigSource       `yaml:"sources"`
	Groups  []ConfigGroupMapping `yaml:"groups"`
	Users   []ConfigUserMapping  `yaml:"users"`
}

// this config is loaded at start-up and re-applied when Infra's state changes (ie. a user is added)
var initialConfig Config

func ImportSources(db *gorm.DB, sources []ConfigSource) error {
	var idsToKeep []string

	for _, s := range sources {
		switch s.Kind {
		case SourceKindOkta:
			// check the domain is specified
			s.cleanupDomain()

			if s.Domain == "" {
				logging.S.Infof("domain not set on source \"%s\", import skipped", s.Kind)
			}

			// check if we are about to override an existing source
			var existing Source

			db.First(&existing, &Source{Kind: SourceKindOkta})

			if existing.Id != "" {
				logging.L.Warn("overriding existing okta source settings with configuration settings")
			}

			var source Source
			if err := db.FirstOrCreate(&source, &Source{Kind: SourceKindOkta}).Error; err != nil {
				return fmt.Errorf("create config source: %w", err)
			}

			source.ClientId = s.ClientId
			source.Domain = s.Domain
			// API token and client secret will be validated to exist when they are used
			source.ClientSecret = s.ClientSecret
			source.APIToken = s.APIToken

			if err := db.Save(&source).Error; err != nil {
				return fmt.Errorf("save source: %w", err)
			}

			idsToKeep = append(idsToKeep, source.Id)
		case "":
			logging.S.Errorf("skipping a source with no kind set in configuration")
		default:
			logging.S.Errorf("skipping invalid source kind in configuration: %s", s.Kind)
		}
	}

	if len(idsToKeep) == 0 {
		logging.L.Debug("no valid sources found in configuration, ensure the required fields are specified correctly")
		// clear the sources
		return db.Where("1 = 1").Delete(&Source{}).Error
	}

	return db.Not(idsToKeep).Delete(&Source{}).Error
}

func ApplyGroupMappings(db *gorm.DB, groups []ConfigGroupMapping) (modifiedRoleIDs []string, err error) {
	for _, g := range groups {
		// get the source from the datastore that this group specifies
		var source Source
		// Assumes that only one kind of each source can exist
		srcReadErr := db.Where(&Source{Kind: g.Source}).First(&source).Error
		if srcReadErr != nil {
			if errors.Is(srcReadErr, gorm.ErrRecordNotFound) {
				// skip this source, it will need to be added in the config and re-applied
				logging.S.Debugf("skipping group '%s' with source '%s' in config that does not exist", g.Name, g.Source)
				continue
			}

			return nil, fmt.Errorf("group read source: %w", srcReadErr)
		}

		var group Group

		grpReadErr := db.Preload("Users").Where(&Group{Name: g.Name, SourceId: source.Id}).First(&group).Error
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

// ImportConfig tries to import all valid fields in a config file and removes old config
func ImportConfig(db *gorm.DB, bs []byte) error {
	var config Config
	if err := yaml.Unmarshal(bs, &config); err != nil {
		return err
	}

	initialConfig = config

	return db.Transaction(func(tx *gorm.DB) error {
		if err := ImportSources(tx, config.Sources); err != nil {
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
						return nil, nil, fmt.Errorf("group read source: %w", err)
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

package registry

import (
	"errors"
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
	ApiToken     string `yaml:"apiToken"`
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

// this config is loaded at start-up and re-applied when the registry state changes (ex: a user is added)
var initialConfig Config

func ImportSources(db *gorm.DB, sources []ConfigSource) error {
	var idsToKeep []string

	for _, s := range sources {
		switch s.Kind {
		case SourceKindOkta:
			// check the domain is specified
			s.cleanupDomain()

			if s.Domain == "" {
				logging.L.Sugar().Infof("domain not set on source \"%s\", import skipped", s.Kind)
			}

			// check if we are about to override an existing source
			var existing Source

			db.First(&existing, &Source{Kind: SourceKindOkta})

			if existing.Id != "" {
				logging.L.Warn("overriding existing okta source settings with configuration settings")
			}

			var source Source
			if err := db.FirstOrCreate(&source, &Source{Kind: SourceKindOkta}).Error; err != nil {
				return err
			}

			source.ClientId = s.ClientId
			source.Domain = s.Domain
			// API token and client secret will be validated to exist when they are used
			source.ClientSecret = s.ClientSecret
			source.ApiToken = s.ApiToken

			if err := db.Save(&source).Error; err != nil {
				return err
			}

			idsToKeep = append(idsToKeep, source.Id)
		default:
			logging.L.Sugar().Errorf("skipping invalid source kind in configuration: %s" + s.Kind)
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

func ImportGroupMapping(db *gorm.DB, groups []ConfigGroupMapping) error {
	for _, g := range groups {
		// get the source from the datastore that this group specifies
		var source Source
		// Assumes that only one type of each source can exist
		srcReadErr := db.Where(&Source{Kind: g.Source}).First(&source).Error
		if srcReadErr != nil {
			if errors.Is(srcReadErr, gorm.ErrRecordNotFound) {
				// skip this source, it will need to be added in the config and re-applied
				logging.L.Sugar().Debugf("skipping group '%s' with source '%s' in config that does not exist", g.Name, g.Source)
				continue
			}

			return srcReadErr
		}

		var group Group

		grpReadErr := db.Where(&Group{Name: g.Name, SourceId: source.Id}).First(&group).Error
		if grpReadErr != nil {
			if errors.Is(grpReadErr, gorm.ErrRecordNotFound) {
				// skip this group, if they're created these roles will be added later
				logging.L.Debug("skipping group in config import that has not yet been provisioned")
				continue
			}

			return grpReadErr
		}

		// import the roles on this group from the datastore
		var roles []Role

		roles, err := importRoles(db, g.Roles)
		if err != nil {
			return err
		}

		// add the new group associations to the roles
		for i, role := range roles {
			if db.Model(&group).Where(&Role{Id: role.Id}).Association("Roles").Count() == 0 {
				if err = db.Model(&group).Where(&Role{Id: role.Id}).Association("Roles").Append(&roles[i]); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func ImportUserMapping(db *gorm.DB, users []ConfigUserMapping) error {
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

		// add direct user to role mappings
		roles, err := importRoles(db, u.Roles)
		if err != nil {
			return err
		}
		// for all roles attached to this user update their user associations now that we have made sure they exist
		// important: do not create the association on the user, that runs an upsert that creates a concurrent write because User.AfterCreate() calls this function
		for i, role := range roles {
			if db.Model(&user).Where(&Role{Id: role.Id}).Association("Roles").Count() == 0 {
				if err = db.Model(&user).Where(&Role{Id: role.Id}).Association("Roles").Append(&roles[i]); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ImportConfig tries to import all valid fields in a config file
func ImportConfig(db *gorm.DB, bs []byte) error {
	var config Config
	if err := yaml.Unmarshal(bs, &config); err != nil {
		return err
	}

	initialConfig = config

	return db.Transaction(func(tx *gorm.DB) error {
		// gorm blocks global delete by default: https://gorm.io/docs/delete.html#Block-Global-Delete
		if err := tx.Where("1 = 1").Delete(&Role{}).Error; err != nil {
			return err
		}

		var users []User
		if err := tx.Find(&users).Error; err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Delete(&users).Error; err != nil {
			return err
		}

		var groups []Group
		if err := tx.Find(&groups).Error; err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Delete(&groups).Error; err != nil {
			return err
		}

		if err := ImportSources(tx, config.Sources); err != nil {
			return err
		}

		if err := ImportGroupMapping(tx, config.Groups); err != nil {
			return err
		}

		return ImportUserMapping(tx, config.Users)
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

			err := db.Where(&Destination{Name: destination.Name}).First(&dest).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// when a destination is added then the config import will be retried, skip for now
					logging.L.Debug("skipping role binding for destination in config import that has not yet been discovered")
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

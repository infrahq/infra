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

type ConfigRoleKubernetes struct {
	Name     string   `yaml:"name"`
	Kind     string   `yaml:"kind"`
	Clusters []string `yaml:"clusters"`
}

type ConfigGroupMapping struct {
	Name    string                 `yaml:"name"`
	Sources []string               `yaml:"sources"`
	Roles   []ConfigRoleKubernetes `yaml:"roles"`
}

type ConfigUserMapping struct {
	Name   string                 `yaml:"name"`
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
			var source Source
			err := db.FirstOrCreate(&source, &Source{Type: s.Type, Domain: s.Domain}).Error
			if err != nil {
				return err
			}

			source.ClientId = s.ClientId
			// API token and client secret will be validated to exist when they are used
			source.ClientSecret = s.ClientSecret
			source.ApiToken = s.ApiToken
			source.FromConfig = true

			err = db.Save(&source).Error
			if err != nil {
				return err
			}

			idsToKeep = append(idsToKeep, source.Id)
		}
	}
	if err := db.Where(&Role{FromConfig: false}).Not(idsToKeep).Not(&Source{Type: SOURCE_TYPE_INFRA}).Delete(&Source{}).Error; err != nil {
		return err
	}
	return nil
}

func ApplyGroupMappings(db *gorm.DB, groups []ConfigGroupMapping) (groupIds []string, roleIds []string, err error) {
	for _, g := range groups {
		// get the sources from the datastore that this group specifies
		var sources []Source
		for _, src := range g.Sources {
			var source Source
			// Assumes that only one type of each source can exist
			srcReadErr := db.Where(&Source{Type: src}).First(&source).Error
			if srcReadErr != nil {
				if errors.Is(srcReadErr, gorm.ErrRecordNotFound) {
					// skip this source, it will need to be added in the config and re-applied
					logging.L.Debug("skipping source in config that does not exist: " + src)
					continue
				}
				err = srcReadErr
				return
			}
			sources = append(sources, source)
		}

		if len(sources) == 0 {
			logging.L.Debug("no valid sources found, skipping group: " + g.Name)
			continue
		}

		var group Group
		// Group names must be unique for mapping purposes
		err = db.FirstOrCreate(&group, &Group{Name: g.Name}).Error
		if err != nil {
			return
		}

		// add the identity provider associations to the group
		for _, source := range sources {
			if db.Model(&source).Where(&Group{Id: group.Id}).Association("Groups").Count() == 0 {
				if err = db.Model(&source).Where(&Group{Id: group.Id}).Association("Groups").Append(&group); err != nil {
					return
				}
			}
		}

		// import the roles on this group from the datastore
		var roles []Role
		roles, err = importRoles(db, g.Roles)
		if err != nil {
			return
		}
		// add the new group associations to the roles
		for _, role := range roles {
			if db.Model(&role).Where(&Group{Id: group.Id}).Association("Groups").Count() == 0 {
				if err = db.Model(&role).Where(&Group{Id: group.Id}).Association("Groups").Append(&group); err != nil {
					return
				}
			}
			roleIds = append(roleIds, role.Id)
		}
		groupIds = append(groupIds, group.Id)
	}
	return
}

func ApplyUserMapping(db *gorm.DB, users []ConfigUserMapping) ([]string, error) {
	var ids []string

	for _, u := range users {
		var user User
		usrReadErr := db.Where(&User{Email: u.Name}).First(&user).Error
		if usrReadErr != nil {
			if errors.Is(usrReadErr, gorm.ErrRecordNotFound) {
				// skip this user, if they're created these roles will be added later
				logging.L.Debug("skipping user in config import that has not yet been provisioned")
				continue
			}
			return nil, usrReadErr
		}

		// add the user to groups
		for _, gName := range u.Groups {
			// Assumes that only one group can exist with a given name, regardless of sources
			var group Group
			grpReadErr := db.Where(&Group{Name: gName}).First(&group).Error
			if grpReadErr != nil {
				if errors.Is(grpReadErr, gorm.ErrRecordNotFound) {
					logging.L.Debug("skipping unknown group \"" + gName + "\" on user")
					continue
				}
				return nil, grpReadErr
			}
			if db.Model(&user).Where(&Group{Id: group.Id}).Association("Groups").Count() == 0 {
				if err := db.Model(&user).Where(&Group{Id: group.Id}).Association("Groups").Append(&group); err != nil {
					return nil, err
				}
			}
		}

		// add roles to user
		roles, err := importRoles(db, u.Roles)
		if err != nil {
			return nil, err
		}
		// for all roles attached to this user update their user associations now that we have made sure they exist
		// important: do not create the association on the user, that runs an upsert that creates a concurrent write because User.AfterCreate() calls this function
		for _, role := range roles {
			if db.Model(&user).Where(&Role{Id: role.Id}).Association("Roles").Count() == 0 {
				if err = db.Model(&user).Where(&Role{Id: role.Id}).Association("Roles").Append(&role); err != nil {
					return nil, err
				}
			}
			ids = append(ids, role.Id)
		}
	}
	return ids, nil
}

// ImportMappings imports the group and user role mappings and removes previously created roles if they no longer exist
func ImportMappings(db *gorm.DB, groups []ConfigGroupMapping, users []ConfigUserMapping) error {
	grpIdsToKeep, grpRoleIdsToKeep, err := ApplyGroupMappings(db, groups)
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
		db.Find(&groups)
		if len(groups) > 0 {
			err = db.Delete(groups).Error
			if err != nil {
				return err
			}
		}
		logging.L.Info("no valid groups found in configuration, continuing...")
	}

	usrRoleIdsToKeep, err := ApplyUserMapping(db, users)
	if err != nil {
		return err
	}

	// clean up existing roles which have been removed from the config
	var roleIdsToKeep []string
	roleIdsToKeep = append(roleIdsToKeep, grpRoleIdsToKeep...)
	roleIdsToKeep = append(roleIdsToKeep, usrRoleIdsToKeep...)
	return db.Where(&Role{FromConfig: true}).Not(roleIdsToKeep).Delete(Role{}).Error
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
		switch r.Kind {
		case ROLE_KIND_K8S_ROLE:
			// TODO (brucemacd): Handle config imports of roles when we support RoleBindings
			logging.L.Info("Skipping role: " + r.Name + ", RoleBindings are not supported yet")
		case ROLE_KIND_K8S_CLUSTER_ROLE:
			for _, cName := range r.Clusters {
				var destination Destination
				err := db.Where(&Destination{Name: cName}).First(&destination).Error
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// when a destination is added then the config import will be retried, skip for now
						logging.L.Debug("skipping destination in config import that has not yet been discovered")
						continue
					}
					return nil, err
				}
				var role Role
				if err = db.FirstOrCreate(&role, &Role{Name: r.Name, Kind: r.Kind, DestinationId: destination.Id, FromConfig: true}).Error; err != nil {
					return nil, err
				}
				rolesImported = append(rolesImported, role)
			}
		default:
			logging.L.Info("Unrecognized role kind: " + r.Kind + " in infra.yaml, role skipped.")
		}
	}
	return rolesImported, nil
}

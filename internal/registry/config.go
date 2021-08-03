package registry

import (
	"errors"

	"github.com/infrahq/infra/internal/logging"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

type ConfigSource struct {
	Type             string `yaml:"type"`
	OktaDomain       string `yaml:"oktaDomain"`
	OktaClientId     string `yaml:"oktaClientId"`
	OktaClientSecret string `yaml:"oktaClientSecret"`
	OktaApiToken     string `yaml:"oktaApiToken"`
}

type ConfigRoleKubernetes struct {
	Name     string   `yaml:"name"`
	Kind     string   `yaml:"kind"`
	Clusters []string `yaml:"clusters"`
}

type ConfigUserMapping struct {
	Name  string
	Roles []ConfigRoleKubernetes
	// TODO (brucemacd): Add groups here
}
type Config struct {
	Sources []ConfigSource      `yaml:"sources"`
	Users   []ConfigUserMapping `yaml:"users"`
}

// this config is loaded at start-up and re-applied when the registry state changes (ex: a user is added)
var initialConfig Config

func ImportSources(db *gorm.DB, sources []ConfigSource) error {
	var idsToKeep []string

	for _, s := range sources {
		switch s.Type {
		case SOURCE_TYPE_OKTA:
			var source Source
			err := db.FirstOrCreate(&source, &Source{Type: s.Type, OktaDomain: s.OktaDomain}).Error
			if err != nil {
				return err
			}

			source.OktaClientId = s.OktaClientId
			source.OktaClientSecret = s.OktaClientSecret
			source.OktaApiToken = s.OktaApiToken
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

func ApplyUserMapping(db *gorm.DB, users []ConfigUserMapping) ([]string, error) {
	var ids []string

	for _, u := range users {
		var user User
		err := db.Where(&User{Email: u.Name}).First(&user).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// skip this user, if they're created these roles will be added later
				logging.L.Debug("skipping user in config import that has not yet been provisioned")
				continue
			}
			return nil, err
		}
		for _, r := range u.Roles {
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
					// if this role is not yet associated with this user, add that association now
					// important: do not create the association on the user, that runs an upsert that creates a deadlock because User.AfterCreate() calls this function
					if db.Model(&user).Where(&Role{Id: role.Id}).Association("Roles").Count() == 0 {
						if err = db.Model(&user).Where(&Role{Id: role.Id}).Association("Roles").Append(&role); err != nil {
							return nil, err
						}
					}
					ids = append(ids, role.Id)
				}
			default:
				logging.L.Info("Unrecognized role kind: " + r.Kind + " in infra.yaml, role skipped.")
			}
		}

		// TODO: add user to groups here
	}
	return ids, nil
}

func ImportUserMappings(db *gorm.DB, users []ConfigUserMapping) error {
	idsToKeep, err := ApplyUserMapping(db, users)
	if err != nil {
		return err
	}
	return db.Where(&Role{FromConfig: true}).Not(idsToKeep).Delete(Role{}).Error
}

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
		if err = ImportUserMappings(tx, config.Users); err != nil {
			return err
		}
		return nil
	})
}

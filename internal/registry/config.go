package registry

import (
	"errors"

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

type ConfigPermission struct {
	Role            string `yaml:"role"`
	UserEmail       string `yaml:"user"`
	DestinationName string `yaml:"destination"`
}

type Config struct {
	Sources     []ConfigSource     `yaml:"sources"`
	Permissions []ConfigPermission `yaml:"permissions"`
}

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

	if err := db.Where(&Permission{FromConfig: false}).Not(idsToKeep).Not(&Source{Type: SOURCE_TYPE_INFRA}).Delete(&Source{}).Error; err != nil {
		return err
	}
	return nil
}

func ApplyPermissions(db *gorm.DB, permissions []ConfigPermission) ([]string, error) {
	var ids []string
	for _, p := range permissions {
		var user User
		err := db.Where(&User{Email: p.UserEmail}).First(&user).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}

		var destination Destination
		err = db.Where(&Destination{Name: p.DestinationName}).First(&destination).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}

		var permission Permission
		err = db.FirstOrCreate(&permission, &Permission{UserId: user.Id, DestinationId: destination.Id, Role: p.Role, FromDefault: false}).Error
		if err != nil {
			return nil, err
		}

		permission.FromConfig = true

		err = db.Save(&permission).Error
		if err != nil {
			return nil, err
		}

		ids = append(ids, permission.Id)
	}
	return ids, nil
}

func ImportPermissions(db *gorm.DB, permissions []ConfigPermission) error {
	idsToKeep, err := ApplyPermissions(db, permissions)
	if err != nil {
		return err
	}
	return db.Where(&Permission{FromConfig: true}).Not(idsToKeep).Delete(Permission{}).Error
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
		if err = ImportPermissions(tx, config.Permissions); err != nil {
			return err
		}
		return nil
	})
}

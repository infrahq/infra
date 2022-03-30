package data

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
)

func migrate(db *gorm.DB) error {
	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		// rename grants.identity -> grants.subject
		{
			ID: "202203231621", // date the migration was created
			Migrate: func(tx *gorm.DB) error {
				// it's a good practice to copy any used structs inside the function,
				// so side-effects are prevented if the original struct changes

				if tx.Migrator().HasColumn(&models.Grant{}, "identity") {
					return tx.Migrator().RenameColumn(&models.Grant{}, "identity", "subject")
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().RenameColumn(&models.Grant{}, "subject", "identity")
			},
		},
		{
			ID: "202203241643", // date the migration was created
			Migrate: func(tx *gorm.DB) error {
				if tx.Migrator().HasColumn(&models.AccessKey{}, "key") {
					return tx.Migrator().RenameColumn(&models.AccessKey{}, "key", "key_id")
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().RenameColumn(&models.AccessKey{}, "key_id", "key")
			},
		},
		// next one here
	})

	m.InitSchema(func(db *gorm.DB) error {
		// TODO: can optionally remove this skip after any existing users have migrated.
		if db.Migrator().HasTable("users") {
			return nil
		}
		return automigrate(db)
	})

	if err := m.Migrate(); err != nil {
		return err
	}

	// automigrate again, so that for simple things like adding db fields we don't necessarily need to do a migration
	return automigrate(db)
}

func automigrate(db *gorm.DB) error {
	tables := []interface{}{
		&models.User{},
		&models.Machine{},
		&models.Group{},
		&models.Grant{},
		&models.Provider{},
		&models.ProviderToken{},
		&models.Destination{},
		&models.AccessKey{},
		&models.Settings{},
		&models.EncryptionKey{},
		&models.TrustedCertificate{},
		&models.RootCertificate{},
		&models.Credential{},
	}

	for _, table := range tables {
		if err := db.AutoMigrate(table); err != nil {
			return err
		}
	}

	return nil
}

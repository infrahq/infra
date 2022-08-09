package data

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data/migrator"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
)

func migrations() []*migrator.Migration {
	return []*migrator.Migration{
		// drop Settings SignupEnabled column
		{
			ID: "202204281130",
			Migrate: func(tx *gorm.DB) error {
				return tx.Migrator().DropColumn(&models.Settings{}, "signup_enabled")
			},
		},
		// #1657: get rid of identity kind
		{
			ID: "202204291613",
			Migrate: func(tx *gorm.DB) error {
				if tx.Migrator().HasColumn(&models.Identity{}, "kind") {
					if err := tx.Migrator().DropColumn(&models.Identity{}, "kind"); err != nil {
						return err
					}
				}

				return nil
			},
		},
		// drop old Groups index; new index will be created automatically
		{
			ID: "2022-06-08T10:27-fixed",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec(`DROP INDEX IF EXISTS idx_groups_name_provider_id`).Error
			},
		},
		addKindToProviders(),
		dropCertificateTables(),
		addAuthURLAndScopeToProviders(),
		setDestinationLastSeenAt(),
		deleteDuplicateGrants(),
		dropDeletedProviderUsers(),
		removeDeletedIdentitiesFromGroups(),
		addFieldsFor_0_14_3(),
		// next one here
	}
}

func initializeSchema(db *gorm.DB) error {
	tables := []interface{}{
		&models.ProviderUser{},
		&models.Group{},
		&models.Identity{},
		&models.Provider{},
		&models.Grant{},
		&models.Destination{},
		&models.AccessKey{},
		&models.Settings{},
		&models.EncryptionKey{},
		&models.Credential{},
		&models.Organization{},
		&models.PasswordResetToken{},
	}
	for _, table := range tables {
		if err := db.AutoMigrate(table); err != nil {
			return err
		}
	}

	return nil
}

// #2294: set the provider kind on existing providers
func addKindToProviders() *migrator.Migration {
	return &migrator.Migration{
		ID: "202206151027",
		Migrate: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(&models.Provider{}, "kind") {
				logging.Debugf("migrating provider table kind")
				if err := tx.Migrator().AddColumn(&models.Provider{}, "kind"); err != nil {
					return err
				}
			}

			db := tx.Begin()
			db.Table("providers").Where("kind IS NULL AND name = ?", "infra").Update("kind", models.ProviderKindInfra)
			db.Table("providers").Where("kind IS NULL").Update("kind", models.ProviderKindOkta)

			return db.Commit().Error
		},
	}
}

// #2276: drop unused certificate tables
func dropCertificateTables() *migrator.Migration {
	return &migrator.Migration{
		ID: "202206161733",
		Migrate: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable("trusted_certificates", "root_certificates")
		},
	}
}

// #2353: store auth URL and scopes to provider
func addAuthURLAndScopeToProviders() *migrator.Migration {
	return &migrator.Migration{
		ID: "202206281027",
		Migrate: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(&models.Provider{}, "scopes") {
				logging.Debugf("migrating provider table auth URL and scopes")
				if err := tx.Migrator().AddColumn(&models.Provider{}, "auth_url"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&models.Provider{}, "scopes"); err != nil {
					return err
				}

				db := tx.Begin()

				// need to select only these fields from the providers
				// we dont have the database encryption key for the client secret at this point
				var providerModels []models.Provider
				err := db.Select("id", "url", "kind").Find(&providerModels).Error
				if err != nil {
					return err
				}

				for i := range providerModels {
					// do not resolve the auth details for the infra provider
					// check infra provider name and kind just in case other migrations haven't run
					if providerModels[i].Kind == models.ProviderKindInfra || providerModels[i].Name == models.InternalInfraProviderName {
						continue
					}

					logging.Debugf("migrating %s provider", providerModels[i].Name)

					providerClient := providers.NewOIDCClient(providerModels[i], "not-used", "http://localhost:8301")
					authServerInfo, err := providerClient.AuthServerInfo(context.Background())
					if err != nil {
						if errors.Is(err, context.DeadlineExceeded) {
							return fmt.Errorf("%w: %s", internal.ErrBadGateway, err)
						}
						return fmt.Errorf("could not get provider info: %w", err)
					}

					db.Model(&providerModels[i]).Update("auth_url", authServerInfo.AuthURL)
					db.Model(&providerModels[i]).Update("scopes", strings.Join(authServerInfo.ScopesSupported, ","))
				}

				return db.Commit().Error
			}

			return nil
		},
	}
}

// #2360: delete duplicate grants (the same subject, resource, and privilege) to allow for unique constraint
func deleteDuplicateGrants() *migrator.Migration {
	return &migrator.Migration{ID: "202207081217", Migrate: func(tx *gorm.DB) error {
		stmt := `
			DELETE FROM grants
			WHERE deleted_at IS NULL
			AND id NOT IN (
				SELECT min(id)
				FROM grants
				WHERE deleted_at IS NULL
				GROUP BY subject, resource, privilege)`
		return tx.Exec(stmt).Error
	}}
}

// setDestinationLastSeenAt creates the `last_seen_at` column if it does not exist and sets it to
// the destination's `updated_at` value. No effect if the `last_seen_at` exists
func setDestinationLastSeenAt() *migrator.Migration {
	return &migrator.Migration{
		ID: "202207041724",
		Migrate: func(tx *gorm.DB) error {
			if tx.Migrator().HasColumn(&models.Destination{}, "last_seen_at") {
				return nil
			}

			if err := tx.Migrator().AddColumn(&models.Destination{}, "last_seen_at"); err != nil {
				return err
			}

			return tx.Exec("UPDATE destinations SET last_seen_at = updated_at").Error
		},
	}
}

// dropDeletedProviderUsers removes soft-deleted provider users so they do not cause conflicts
func dropDeletedProviderUsers() *migrator.Migration {
	return &migrator.Migration{
		ID: "202207270000",
		Migrate: func(tx *gorm.DB) error {
			if tx.Migrator().HasColumn(&models.ProviderUser{}, "deleted_at") {
				if err := tx.Exec("DELETE FROM provider_users WHERE deleted_at IS NOT NULL").Error; err != nil {
					return fmt.Errorf("could not remove soft deleted provider users: %w", err)
				}
				return tx.Migrator().DropColumn(&models.ProviderUser{}, "deleted_at")
			}
			return nil
		},
	}
}

func removeDeletedIdentitiesFromGroups() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-07-28T12:46",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec("DELETE FROM identities_groups WHERE identity_id in (SELECT id FROM identities WHERE deleted_at IS NOT NULL)").Error
		},
	}
}

// addFieldsFor_0_14_3 adds all migrations that were previously applied by a
// second call to gorm.AutoMigrate. In this release we're removing the
// unconditional call to gorm.AutoMigrate in favor of having explicit migrations
// for all changes.
//
// To account for all the existing migrations that were applied by AutoMigrate
// we have to call it here again on any tables that have had changes.
//
// In the future we should use ALTER TABLE sql statements instead of AutoMigrate.
//
// nolint:revive
func addFieldsFor_0_14_3() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-07-21T18:28",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&models.Provider{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&models.Settings{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&models.Organization{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&models.AccessKey{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&models.Credential{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&models.Destination{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&models.ProviderUser{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&models.Group{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&models.PasswordResetToken{}); err != nil {
				return err
			}

			if err := tx.Exec(`
ALTER TABLE credentials DROP COLUMN IF EXISTS one_time_password_used;

ALTER TABLE provider_users DROP COLUMN IF EXISTS id;
ALTER TABLE provider_users DROP COLUMN IF EXISTS created_at;
ALTER TABLE provider_users DROP COLUMN IF EXISTS updated_at;
`).Error; err != nil {
				return err
			}

			if !tx.Migrator().HasConstraint("provider_users", "provider_users_pkey") {
				if err := tx.Exec(`
ALTER TABLE ONLY provider_users
	ADD CONSTRAINT fk_provider_users_identity FOREIGN KEY (identity_id) REFERENCES identities(id);

ALTER TABLE ONLY provider_users
	ADD CONSTRAINT fk_provider_users_provider FOREIGN KEY (provider_id) REFERENCES providers(id);

ALTER TABLE provider_users ADD CONSTRAINT provider_users_pkey
	PRIMARY KEY (identity_id, provider_id);

`).Error; err != nil {
					return err
				}
			}

			if !tx.Migrator().HasIndex("grants", "idx_grant_srp") {
				stmt := `CREATE UNIQUE INDEX idx_grant_srp ON testing.grants USING btree (subject, privilege, resource) WHERE (deleted_at IS NULL);`
				if err := tx.Exec(stmt).Error; err != nil {
					return err
				}
			}

			return nil
		},
	}
}

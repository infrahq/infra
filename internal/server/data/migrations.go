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
		addOrganizations(),
		scopeUniqueIndicesToOrganization(),
		// next one here
	}
}

func initializeSchema(db *gorm.DB) error {
	tables := []interface{}{
		&models.Identity{},
		&models.Group{},
		&models.Grant{},
		&models.Provider{},
		&models.Destination{},
		&models.AccessKey{},
		&models.Settings{},
		&models.EncryptionKey{},
		&models.Credential{},
		&models.ProviderUser{},
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

func addOrganizations() *migrator.Migration {
	return &migrator.Migration{
		ID: "202207271554",
		Migrate: func(tx *gorm.DB) error {
			logging.Debugf("migrating orgs")
			mods := []interface{}{
				&models.AccessKey{},
				&models.Credential{},
				&models.Destination{},
				&models.EncryptionKey{},
				&models.Grant{},
				&models.Group{},
				&models.Identity{},
				&models.Organization{},
				&models.Provider{},
				&models.Settings{},
			}

			for _, mod := range mods {
				if tx.Migrator().HasTable(mod) {
					if !tx.Migrator().HasColumn(mod, "organization_id") {
						if err := tx.Migrator().AddColumn(mod, "organization_id"); err != nil {
							logging.Debugf("failed to add column: %q", mod)
							return err
						}
					}
				}
			}

			return nil
		},
	}
}

func scopeUniqueIndicesToOrganization() *migrator.Migration {
	return &migrator.Migration{
		ID: "202208041772",
		Migrate: func(tx *gorm.DB) error {
			queries := strings.Split(`
drop index if exists idx_access_keys_name;
drop index if exists idx_access_keys_key_id;
drop index if exists idx_credentials_identity_id;
drop index if exists idx_destinations_unique_id;
drop index if exists idx_grant_srp;
drop index if exists idx_groups_name;
drop index if exists idx_identities_name;
drop index if exists idx_providers_name;

create unique index idx_access_keys_name on access_keys (organization_id, name) where (deleted_at is null);
create unique index idx_access_keys_key_id on access_keys (organization_id, key_id) where (deleted_at is null);
create unique index idx_credentials_identity_id ON credentials ("organization_id","identity_id") where (deleted_at is null);
create unique index idx_destinations_unique_id ON destinations ("organization_id","unique_id") where (deleted_at is null);
create unique index idx_grant_srp ON grants ("organization_id","subject","privilege","resource") where (deleted_at is null);
create unique index idx_groups_name ON groups ("organization_id","name") where (deleted_at is null);
create unique index idx_identities_name ON identities ("organization_id","name") where (deleted_at is null);
create unique index idx_providers_name ON providers ("organization_id","name") where (deleted_at is null);
create unique index settings_org_id ON settings ("organization_id") where deleted_at is null;

drop table if exists identities_organizations;

alter table "settings" alter column "id" drop default;
alter table "providers" alter column "id" drop default;
alter table "organizations" alter column "id" drop default;
alter table "access_keys" alter column "id" drop default;
alter table "credentials" alter column "id" drop default;
alter table "destinations" alter column "id" drop default;
alter table "encryption_keys" alter column "id" drop default;
alter table "grants" alter column "id" drop default;
alter table "groups" alter column "id" drop default;
alter table "identities" alter column "id" drop default;

alter table provider_users DROP CONSTRAINT "fk_provider_users_provider";
alter table provider_users DROP CONSTRAINT "fk_provider_users_identity";
alter table identities_groups DROP CONSTRAINT "fk_identities_groups_identity";
alter table identities_groups DROP CONSTRAINT "fk_identities_groups_group";
alter table access_keys DROP CONSTRAINT "fk_access_keys_issued_for_identity";

drop sequence access_keys_id_seq;
drop sequence credentials_id_seq;
drop sequence destinations_id_seq;
drop sequence encryption_keys_id_seq;
drop sequence grants_id_seq;
drop sequence groups_id_seq;
drop sequence identities_id_seq;
drop sequence organizations_id_seq;
drop sequence providers_id_seq;
drop sequence settings_id_seq;
			`, ";\n")
			// note running these one line at a time makes for _much_ better errors when one line fails.
			for _, query := range queries {
				query = strings.Trim(query, "\n")
				err := tx.Exec(query).Error
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}

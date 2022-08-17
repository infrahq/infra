package data

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data/migrator"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/uid"
)

func migrations() []*migrator.Migration {
	return []*migrator.Migration{
		// drop Settings SignupEnabled column
		{
			ID: "202204281130",
			Migrate: func(tx *gorm.DB) error {
				stmt := `ALTER TABLE settings DROP COLUMN IF EXISTS signup_enabled`
				if tx.Dialector.Name() == "sqlite" {
					stmt = `ALTER TABLE settings DROP COLUMN signup_enabled`
				}
				return tx.Exec(stmt).Error
			},
		},
		// #1657: get rid of identity kind
		{
			ID: "202204291613",
			Migrate: func(tx *gorm.DB) error {
				stmt := `ALTER TABLE identities DROP COLUMN IF EXISTS kind`
				if tx.Dialector.Name() == "sqlite" {
					stmt = `ALTER TABLE identities DROP COLUMN kind`
				}
				return tx.Exec(stmt).Error
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
		addFieldsForPreviouslyImplicitMigrations(),
		addOrganizations(),
		scopeUniqueIndicesToOrganization(),
		addDefaultOrganization(),
		addOrganizationDomain(),
		dropOrganizationNameIndex(),
		// next one here
	}
}

//go:embed schema.sql
var schemaSQL string

func initializeSchema(db *gorm.DB) error {
	if db.Dialector.Name() == "sqlite" {
		return autoMigrateSchema(db)
	}

	if err := db.Exec(schemaSQL).Error; err != nil {
		return fmt.Errorf("failed to exec sql: %w", err)
	}
	return nil
}

func autoMigrateSchema(db *gorm.DB) error {
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
			stmt := `ALTER TABLE providers ADD COLUMN IF NOT EXISTS kind text`
			if tx.Dialector.Name() == "sqlite" {
				stmt = `ALTER TABLE providers ADD COLUMN kind text`
			}
			if err := tx.Exec(stmt).Error; err != nil {
				return err
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
			if err := tx.Exec(`DROP TABLE IF EXISTS trusted_certificates`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`DROP TABLE IF EXISTS root_certificates`).Error; err != nil {
				return err
			}
			return nil
		},
	}
}

// #2353: store auth URL and scopes to provider
func addAuthURLAndScopeToProviders() *migrator.Migration {
	return &migrator.Migration{
		ID: "202206281027",
		Migrate: func(tx *gorm.DB) error {
			if !migrator.HasColumn(tx, "providers", "scopes") {
				logging.Debugf("migrating provider table auth URL and scopes")
				if err := tx.Exec(`ALTER TABLE providers ADD COLUMN auth_url text`).Error; err != nil {
					return err
				}
				if err := tx.Exec(`ALTER TABLE providers ADD COLUMN scopes text`).Error; err != nil {
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
			if migrator.HasColumn(tx, "destinations", "last_seen_at") {
				return nil
			}

			stmt := `ALTER TABLE destinations ADD COLUMN last_seen_at timestamp with time zone`
			if tx.Dialector.Name() == "sqlite" {
				stmt = `ALTER TABLE destinations ADD COLUMN last_seen_at datetime`
			}
			if err := tx.Exec(stmt).Error; err != nil {
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
			if migrator.HasColumn(tx, "provider_users", "deleted_at") {
				if err := tx.Exec("DELETE FROM provider_users WHERE deleted_at IS NOT NULL").Error; err != nil {
					return fmt.Errorf("could not remove soft deleted provider users: %w", err)
				}
				return tx.Exec(`ALTER TABLE provider_users DROP COLUMN deleted_at`).Error
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

// addFieldsForPreviouslyImplicitMigrations adds all migrations that were previously applied by a
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
func addFieldsForPreviouslyImplicitMigrations() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-07-21T18:28",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Exec(`
ALTER TABLE settings ADD COLUMN IF NOT EXISTS lowercase_min bigint DEFAULT 0;
ALTER TABLE settings ADD COLUMN IF NOT EXISTS uppercase_min bigint DEFAULT 0;
ALTER TABLE settings ADD COLUMN IF NOT EXISTS number_min bigint DEFAULT 0;
ALTER TABLE settings ADD COLUMN IF NOT EXISTS symbol_min bigint DEFAULT 0;
ALTER TABLE settings ADD COLUMN IF NOT EXISTS length_min bigint DEFAULT 8;
			`).Error; err != nil {
				return err
			}
			if !migrator.HasTable(tx, "organizations") {
				if err := tx.Exec(`
CREATE TABLE organizations (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    created_by bigint
);
CREATE SEQUENCE organizations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE organizations_id_seq OWNED BY organizations.id;
ALTER TABLE ONLY organizations ALTER COLUMN id SET DEFAULT nextval('organizations_id_seq'::regclass);
ALTER TABLE ONLY organizations ADD CONSTRAINT organizations_pkey PRIMARY KEY (id);
CREATE UNIQUE INDEX idx_organizations_name ON organizations USING btree (name) WHERE (deleted_at IS NULL);
				`).Error; err != nil {
					return err
				}
			}

			if err := tx.Exec(`
ALTER TABLE providers ADD COLUMN IF NOT EXISTS private_key text;
ALTER TABLE providers ADD COLUMN IF NOT EXISTS client_email text;
ALTER TABLE providers ADD COLUMN IF NOT EXISTS domain_admin_email text;
			`).Error; err != nil {
				return err
			}

			if err := tx.Exec(`
ALTER TABLE access_keys ADD COLUMN IF NOT EXISTS scopes text;
			`).Error; err != nil {
				return err
			}

			if err := tx.Exec(`
ALTER TABLE destinations ADD COLUMN IF NOT EXISTS version text;
ALTER TABLE destinations ADD COLUMN IF NOT EXISTS resources text;
ALTER TABLE destinations ADD COLUMN IF NOT EXISTS roles text;
			`).Error; err != nil {
				return err
			}

			if err := tx.Exec(`
ALTER TABLE groups ADD COLUMN IF NOT EXISTS created_by_provider bigint;
CREATE UNIQUE INDEX IF NOT EXISTS idx_groups_name ON groups USING btree (name) WHERE (deleted_at IS NULL);
			`).Error; err != nil {
				return err
			}
			if !migrator.HasTable(tx, "password_reset_tokens") {
				err := tx.Exec(`
CREATE TABLE password_reset_tokens (
    id bigint NOT NULL,
    token text,
    identity_id bigint,
    expires_at timestamp with time zone
);
CREATE SEQUENCE password_reset_tokens_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE password_reset_tokens_id_seq OWNED BY password_reset_tokens.id;
ALTER TABLE ONLY password_reset_tokens ALTER COLUMN id SET DEFAULT nextval('password_reset_tokens_id_seq'::regclass);
ALTER TABLE ONLY password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (id);
CREATE UNIQUE INDEX idx_password_reset_tokens_token ON password_reset_tokens USING btree (token);
`).Error
				if err != nil {
					return err
				}
			}

			if err := tx.Exec(`
ALTER TABLE credentials DROP COLUMN IF EXISTS one_time_password_used;

ALTER TABLE provider_users DROP COLUMN IF EXISTS id;
ALTER TABLE provider_users DROP COLUMN IF EXISTS created_at;
ALTER TABLE provider_users DROP COLUMN IF EXISTS updated_at;
`).Error; err != nil {
				return err
			}

			if !migrator.HasConstraint(tx, "provider_users", "provider_users_pkey") {
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

			if err := tx.Exec(`
CREATE UNIQUE INDEX IF NOT EXISTS idx_grant_srp ON grants USING btree (subject, privilege, resource) WHERE (deleted_at IS NULL);
			`).Error; err != nil {
				return err
			}

			return nil
		},
	}
}

func addOrganizations() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-07-27T15:54",
		Migrate: func(tx *gorm.DB) error {
			logging.Debugf("migrating orgs")

			stmt := `
ALTER TABLE IF EXISTS access_keys ADD COLUMN IF NOT EXISTS organization_id bigint;
ALTER TABLE IF EXISTS credentials ADD COLUMN IF NOT EXISTS organization_id bigint;
ALTER TABLE IF EXISTS destinations ADD COLUMN IF NOT EXISTS organization_id bigint;
ALTER TABLE IF EXISTS grants ADD COLUMN IF NOT EXISTS organization_id bigint;
ALTER TABLE IF EXISTS groups ADD COLUMN IF NOT EXISTS organization_id bigint;
ALTER TABLE IF EXISTS identities ADD COLUMN IF NOT EXISTS organization_id bigint;
ALTER TABLE IF EXISTS providers ADD COLUMN IF NOT EXISTS organization_id bigint;
ALTER TABLE IF EXISTS settings ADD COLUMN IF NOT EXISTS organization_id bigint;
ALTER TABLE IF EXISTS password_reset_tokens ADD COLUMN IF NOT EXISTS organization_id bigint;
`
			return tx.Exec(stmt).Error
		},
	}
}

func scopeUniqueIndicesToOrganization() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-08-04T17:72",
		Migrate: func(tx *gorm.DB) error {
			stmt := `
DROP INDEX IF EXISTS idx_access_keys_name;
DROP INDEX IF EXISTS idx_credentials_identity_id;
DROP INDEX IF EXISTS idx_destinations_unique_id;
DROP INDEX IF EXISTS idx_grant_srp;
DROP INDEX IF EXISTS idx_groups_name;
DROP INDEX IF EXISTS idx_identities_name;
DROP INDEX IF EXISTS idx_providers_name;
`
			if err := tx.Exec(stmt).Error; err != nil {
				return err
			}

			stmt = `
CREATE UNIQUE INDEX idx_access_keys_name on access_keys (organization_id, name) where (deleted_at is null);
CREATE UNIQUE INDEX idx_credentials_identity_id ON credentials (organization_id,identity_id) where (deleted_at is null);
CREATE UNIQUE INDEX idx_destinations_unique_id ON destinations (organization_id,unique_id) where (deleted_at is null);
CREATE UNIQUE INDEX idx_grant_srp ON grants (organization_id,subject,privilege,resource) where (deleted_at is null);
CREATE UNIQUE INDEX idx_groups_name ON groups (organization_id,name) where (deleted_at is null);
CREATE UNIQUE INDEX idx_identities_name ON identities (organization_id,name) where (deleted_at is null);
CREATE UNIQUE INDEX idx_providers_name ON providers (organization_id,name) where (deleted_at is null);
CREATE UNIQUE INDEX IF NOT EXISTS settings_org_id ON settings (organization_id) where deleted_at is null;
`
			if err := tx.Exec(stmt).Error; err != nil {
				return err
			}

			stmt = `
ALTER TABLE provider_users DROP CONSTRAINT IF EXISTS fk_provider_users_provider;
ALTER TABLE provider_users DROP CONSTRAINT IF EXISTS fk_provider_users_identity;
ALTER TABLE identities_groups DROP CONSTRAINT IF EXISTS fk_identities_groups_identity;
ALTER TABLE identities_groups DROP CONSTRAINT IF EXISTS fk_identities_groups_group;
ALTER TABLE access_keys DROP CONSTRAINT IF EXISTS fk_access_keys_issued_for_identity;
`
			if err := tx.Exec(stmt).Error; err != nil {
				return err
			}

			stmt = `
DROP SEQUENCE IF EXISTS access_keys_id_seq CASCADE;
DROP SEQUENCE IF EXISTS credentials_id_seq CASCADE;
DROP SEQUENCE IF EXISTS destinations_id_seq CASCADE;
DROP SEQUENCE IF EXISTS encryption_keys_id_seq CASCADE;
DROP SEQUENCE IF EXISTS grants_id_seq CASCADE;
DROP SEQUENCE IF EXISTS groups_id_seq CASCADE;
DROP SEQUENCE IF EXISTS identities_id_seq CASCADE;
DROP SEQUENCE IF EXISTS organizations_id_seq CASCADE;
DROP SEQUENCE IF EXISTS providers_id_seq CASCADE;
DROP SEQUENCE IF EXISTS settings_id_seq CASCADE;
DROP SEQUENCE IF EXISTS password_reset_tokens_id_seq CASCADE;
`
			return tx.Exec(stmt).Error
		},
	}
}

func addDefaultOrganization() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-08-10T13:35",
		Migrate: func(tx *gorm.DB) error {
			stmt := `
INSERT INTO organizations(id, name, created_at, updated_at)
VALUES (?, ?, ?, ?);
`
			orgID := uid.New()
			now := time.Now()
			if err := tx.Exec(stmt, orgID, "Default", now, now).Error; err != nil {
				return err
			}

			// postgres only allows a single statement when using parameters
			for _, stmt := range []string{
				`UPDATE access_keys SET organization_id = ?;`,
				`UPDATE credentials SET organization_id = ?;`,
				`UPDATE destinations SET organization_id = ?;`,
				`UPDATE grants SET organization_id = ?;`,
				`UPDATE groups SET organization_id = ?;`,
				`UPDATE identities SET organization_id = ?;`,
				`UPDATE providers SET organization_id = ?;`,
				`UPDATE settings SET organization_id = ?;`,
				`UPDATE password_reset_tokens SET organization_id = ?;`,
			} {
				if err := tx.Exec(stmt, orgID).Error; err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func addOrganizationDomain() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-08-11T11:52",
		Migrate: func(db *gorm.DB) error {
			stmt := `
ALTER TABLE IF EXISTS organizations ADD COLUMN IF NOT EXISTS domain text;
CREATE UNIQUE INDEX IF NOT EXISTS idx_organizations_domain ON organizations USING btree (domain) WHERE (deleted_at IS NULL);
`
			return db.Exec(stmt).Error
		},
	}
}

func dropOrganizationNameIndex() *migrator.Migration {
	return &migrator.Migration{
		ID: "202208121105",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec(`DROP INDEX IF EXISTS idx_organizations_name`).Error
		},
	}
}

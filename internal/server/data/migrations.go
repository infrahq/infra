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
			Migrate: func(tx migrator.DB) error {
				stmt := `ALTER TABLE settings DROP COLUMN IF EXISTS signup_enabled`
				if tx.DriverName() == "sqlite" {
					stmt = `ALTER TABLE settings DROP COLUMN signup_enabled`
				}
				_, err := tx.Exec(stmt)
				return err
			},
		},
		// #1657: get rid of identity kind
		{
			ID: "202204291613",
			Migrate: func(tx migrator.DB) error {
				stmt := `ALTER TABLE identities DROP COLUMN IF EXISTS kind`
				if tx.DriverName() == "sqlite" {
					stmt = `ALTER TABLE identities DROP COLUMN kind`
				}
				_, err := tx.Exec(stmt)
				return err
			},
		},
		// drop old Groups index; new index will be created automatically
		{
			ID: "2022-06-08T10:27-fixed",
			Migrate: func(tx migrator.DB) error {
				_, err := tx.Exec(`DROP INDEX IF EXISTS idx_groups_name_provider_id`)
				return err
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
		sqlFunctionsMigration(),
		setDefaultOrgID(),
		addIdentityVerifiedFields(),
		cleanCrossOrgGroupMemberships(),
		fixProviderUserIndex(),
		removeDotFromDestinationName(),
		// next one here
	}
}

//go:embed schema.sql
var schemaSQL string

func initializeSchema(db migrator.DB) error {
	if db.DriverName() == "sqlite" {
		dataDB, ok := db.(*DB)
		if !ok {
			panic("unexpected DB type, remove this with gorm")
		}
		return autoMigrateSchema(dataDB.DB)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
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
		Migrate: func(tx migrator.DB) error {
			stmt := `ALTER TABLE providers ADD COLUMN IF NOT EXISTS kind text`
			if tx.DriverName() == "sqlite" {
				stmt = `ALTER TABLE providers ADD COLUMN kind text`
			}
			if _, err := tx.Exec(stmt); err != nil {
				return err
			}

			stmt = `UPDATE providers SET kind = ? WHERE kind IS NULL AND name = ?`
			if _, err := tx.Exec(stmt, models.ProviderKindInfra, "infra"); err != nil {
				return err
			}
			stmt = `UPDATE providers SET kind = ? WHERE kind IS NULL`
			if _, err := tx.Exec(stmt, models.ProviderKindOkta); err != nil {
				return err
			}
			return nil
		},
	}
}

// #2276: drop unused certificate tables
func dropCertificateTables() *migrator.Migration {
	return &migrator.Migration{
		ID: "202206161733",
		Migrate: func(tx migrator.DB) error {
			if _, err := tx.Exec(`DROP TABLE IF EXISTS trusted_certificates`); err != nil {
				return err
			}
			if _, err := tx.Exec(`DROP TABLE IF EXISTS root_certificates`); err != nil {
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
		Migrate: func(tx migrator.DB) error {
			if !migrator.HasColumn(tx, "providers", "scopes") {
				logging.Debugf("migrating provider table auth URL and scopes")
				if _, err := tx.Exec(`ALTER TABLE providers ADD COLUMN auth_url text`); err != nil {
					return err
				}
				if _, err := tx.Exec(`ALTER TABLE providers ADD COLUMN scopes text`); err != nil {
					return err
				}

				stmt := `SELECT id, url, kind FROM providers`
				rows, err := tx.Query(stmt)
				if err != nil {
					return err
				}

				for rows.Next() {
					var provider models.Provider
					if err := rows.Scan(&provider.ID, &provider.URL, &provider.Kind); err != nil {
						return err
					}

					// do not resolve the auth details for the infra provider
					// check infra provider name and kind just in case other migrations haven't run
					if provider.Kind == models.ProviderKindInfra || provider.Name == models.InternalInfraProviderName {
						continue
					}

					logging.Debugf("migrating %s provider", provider.Name)

					providerClient := providers.NewOIDCClient(provider, "not-used", "http://localhost:8301")
					authServerInfo, err := providerClient.AuthServerInfo(context.Background())
					if err != nil {
						if errors.Is(err, context.DeadlineExceeded) {
							return fmt.Errorf("%w: %s", internal.ErrBadGateway, err)
						}
						return fmt.Errorf("could not get provider info: %w", err)
					}

					scopes := models.CommaSeparatedStrings(authServerInfo.ScopesSupported)
					stmt := `UPDATE providers SET auth_url = ?, scopes = ? WHERE id = ?`
					_, err = tx.Exec(stmt, authServerInfo.AuthURL, scopes, provider.ID)
					if err != nil {
						return err
					}
				}
				return rows.Close()
			}

			return nil
		},
	}
}

// #2360: delete duplicate grants (the same subject, resource, and privilege) to allow for unique constraint
func deleteDuplicateGrants() *migrator.Migration {
	return &migrator.Migration{ID: "202207081217", Migrate: func(tx migrator.DB) error {
		stmt := `
			DELETE FROM grants
			WHERE deleted_at IS NULL
			AND id NOT IN (
				SELECT min(id)
				FROM grants
				WHERE deleted_at IS NULL
				GROUP BY subject, resource, privilege)`
		_, err := tx.Exec(stmt)
		return err
	}}
}

// setDestinationLastSeenAt creates the `last_seen_at` column if it does not exist and sets it to
// the destination's `updated_at` value. No effect if the `last_seen_at` exists
func setDestinationLastSeenAt() *migrator.Migration {
	return &migrator.Migration{
		ID: "202207041724",
		Migrate: func(tx migrator.DB) error {
			if migrator.HasColumn(tx, "destinations", "last_seen_at") {
				return nil
			}

			stmt := `ALTER TABLE destinations ADD COLUMN last_seen_at timestamp with time zone`
			if tx.DriverName() == "sqlite" {
				stmt = `ALTER TABLE destinations ADD COLUMN last_seen_at datetime`
			}
			if _, err := tx.Exec(stmt); err != nil {
				return err
			}
			_, err := tx.Exec("UPDATE destinations SET last_seen_at = updated_at")
			return err
		},
	}
}

// dropDeletedProviderUsers removes soft-deleted provider users so they do not cause conflicts
func dropDeletedProviderUsers() *migrator.Migration {
	return &migrator.Migration{
		ID: "202207270000",
		Migrate: func(tx migrator.DB) error {
			if migrator.HasColumn(tx, "provider_users", "deleted_at") {
				if _, err := tx.Exec("DELETE FROM provider_users WHERE deleted_at IS NOT NULL"); err != nil {
					return fmt.Errorf("could not remove soft deleted provider users: %w", err)
				}
				_, err := tx.Exec(`ALTER TABLE provider_users DROP COLUMN deleted_at`)
				return err
			}
			return nil
		},
	}
}

func removeDeletedIdentitiesFromGroups() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-07-28T12:46",
		Migrate: func(tx migrator.DB) error {
			_, err := tx.Exec("DELETE FROM identities_groups WHERE identity_id in (SELECT id FROM identities WHERE deleted_at IS NOT NULL)")
			return err
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
		Migrate: func(tx migrator.DB) error {
			if _, err := tx.Exec(`
ALTER TABLE settings ADD COLUMN IF NOT EXISTS lowercase_min bigint DEFAULT 0;
ALTER TABLE settings ADD COLUMN IF NOT EXISTS uppercase_min bigint DEFAULT 0;
ALTER TABLE settings ADD COLUMN IF NOT EXISTS number_min bigint DEFAULT 0;
ALTER TABLE settings ADD COLUMN IF NOT EXISTS symbol_min bigint DEFAULT 0;
ALTER TABLE settings ADD COLUMN IF NOT EXISTS length_min bigint DEFAULT 8;
			`); err != nil {
				return err
			}
			if !migrator.HasTable(tx, "organizations") {
				if _, err := tx.Exec(`
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
				`); err != nil {
					return err
				}
			}

			if _, err := tx.Exec(`
ALTER TABLE providers ADD COLUMN IF NOT EXISTS private_key text;
ALTER TABLE providers ADD COLUMN IF NOT EXISTS client_email text;
ALTER TABLE providers ADD COLUMN IF NOT EXISTS domain_admin_email text;
			`); err != nil {
				return err
			}

			if _, err := tx.Exec(`
ALTER TABLE access_keys ADD COLUMN IF NOT EXISTS scopes text;
			`); err != nil {
				return err
			}

			if _, err := tx.Exec(`
ALTER TABLE destinations ADD COLUMN IF NOT EXISTS version text;
ALTER TABLE destinations ADD COLUMN IF NOT EXISTS resources text;
ALTER TABLE destinations ADD COLUMN IF NOT EXISTS roles text;
			`); err != nil {
				return err
			}

			if _, err := tx.Exec(`
ALTER TABLE groups ADD COLUMN IF NOT EXISTS created_by_provider bigint;
CREATE UNIQUE INDEX IF NOT EXISTS idx_groups_name ON groups USING btree (name) WHERE (deleted_at IS NULL);
			`); err != nil {
				return err
			}
			if !migrator.HasTable(tx, "password_reset_tokens") {
				_, err := tx.Exec(`
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
`)
				if err != nil {
					return err
				}
			}

			if _, err := tx.Exec(`
ALTER TABLE credentials DROP COLUMN IF EXISTS one_time_password_used;

ALTER TABLE provider_users DROP COLUMN IF EXISTS id;
ALTER TABLE provider_users DROP COLUMN IF EXISTS created_at;
ALTER TABLE provider_users DROP COLUMN IF EXISTS updated_at;
`); err != nil {
				return err
			}

			if !migrator.HasConstraint(tx, "provider_users", "provider_users_pkey") {
				if _, err := tx.Exec(`
ALTER TABLE ONLY provider_users
	ADD CONSTRAINT fk_provider_users_identity FOREIGN KEY (identity_id) REFERENCES identities(id);

ALTER TABLE ONLY provider_users
	ADD CONSTRAINT fk_provider_users_provider FOREIGN KEY (provider_id) REFERENCES providers(id);

ALTER TABLE provider_users ADD CONSTRAINT provider_users_pkey
	PRIMARY KEY (identity_id, provider_id);

`); err != nil {
					return err
				}
			}

			if _, err := tx.Exec(`
CREATE UNIQUE INDEX IF NOT EXISTS idx_grant_srp ON grants USING btree (subject, privilege, resource) WHERE (deleted_at IS NULL);
			`); err != nil {
				return err
			}

			return nil
		},
	}
}

func addOrganizations() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-07-27T15:54",
		Migrate: func(tx migrator.DB) error {
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
			_, err := tx.Exec(stmt)
			return err
		},
	}
}

func scopeUniqueIndicesToOrganization() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-08-04T17:72",
		Migrate: func(tx migrator.DB) error {
			stmt := `
DROP INDEX IF EXISTS idx_access_keys_name;
DROP INDEX IF EXISTS idx_credentials_identity_id;
DROP INDEX IF EXISTS idx_destinations_unique_id;
DROP INDEX IF EXISTS idx_grant_srp;
DROP INDEX IF EXISTS idx_groups_name;
DROP INDEX IF EXISTS idx_identities_name;
DROP INDEX IF EXISTS idx_providers_name;
`
			if _, err := tx.Exec(stmt); err != nil {
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
			if _, err := tx.Exec(stmt); err != nil {
				return err
			}

			stmt = `
ALTER TABLE provider_users DROP CONSTRAINT IF EXISTS fk_provider_users_provider;
ALTER TABLE provider_users DROP CONSTRAINT IF EXISTS fk_provider_users_identity;
ALTER TABLE identities_groups DROP CONSTRAINT IF EXISTS fk_identities_groups_identity;
ALTER TABLE identities_groups DROP CONSTRAINT IF EXISTS fk_identities_groups_group;
ALTER TABLE access_keys DROP CONSTRAINT IF EXISTS fk_access_keys_issued_for_identity;
`
			if _, err := tx.Exec(stmt); err != nil {
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
			_, err := tx.Exec(stmt)
			return err
		},
	}
}

func addDefaultOrganization() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-08-10T13:35",
		Migrate: func(tx migrator.DB) error {
			row := tx.QueryRow(`SELECT count(id) from organizations where name='Default'`)
			var count int
			if err := row.Scan(&count); err != nil {
				return err
			}
			if count > 0 {
				return nil
			}

			stmt := `
INSERT INTO organizations(id, name, created_at, updated_at)
VALUES (?, ?, ?, ?);
`
			orgID := uid.New()
			now := time.Now()
			if _, err := tx.Exec(stmt, orgID, "Default", now, now); err != nil {
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
				if _, err := tx.Exec(stmt, orgID); err != nil {
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
		Migrate: func(tx migrator.DB) error {
			stmt := `
ALTER TABLE IF EXISTS organizations ADD COLUMN IF NOT EXISTS domain text;
CREATE UNIQUE INDEX IF NOT EXISTS idx_organizations_domain ON organizations USING btree (domain) WHERE (deleted_at IS NULL);
`
			_, err := tx.Exec(stmt)
			return err
		},
	}
}

func dropOrganizationNameIndex() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-08-12T11:05",
		Migrate: func(tx migrator.DB) error {
			_, err := tx.Exec(`DROP INDEX IF EXISTS idx_organizations_name`)
			return err
		},
	}
}

func setDefaultOrgID() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-08-30T11:45",
		Migrate: func(tx migrator.DB) error {
			var originalOrgID uid.ID
			err := tx.QueryRow(`SELECT id from organizations where name='Default'`).Scan(&originalOrgID)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`UPDATE organizations set id=? WHERE id=?`, defaultOrganizationID, originalOrgID)
			if err != nil {
				return err
			}

			// postgres only allows a single statement when using parameters
			for _, stmt := range []string{
				`UPDATE access_keys SET organization_id = ? WHERE organization_id = ?;`,
				`UPDATE credentials SET organization_id = ? WHERE organization_id = ?;`,
				`UPDATE destinations SET organization_id = ? WHERE organization_id = ?;`,
				`UPDATE grants SET organization_id = ? WHERE organization_id = ?;`,
				`UPDATE groups SET organization_id = ? WHERE organization_id = ?;`,
				`UPDATE identities SET organization_id = ? WHERE organization_id = ?;`,
				`UPDATE providers SET organization_id = ? WHERE organization_id = ?;`,
				`UPDATE settings SET organization_id = ? WHERE organization_id = ?;`,
				`UPDATE password_reset_tokens SET organization_id = ?  WHERE organization_id = ?;`,
			} {
				if _, err := tx.Exec(stmt, defaultOrganizationID, originalOrgID); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func addIdentityVerifiedFields() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-09-01T15:00",
		Migrate: func(tx migrator.DB) error {
			stmt := `
ALTER TABLE identities
	ADD COLUMN IF NOT EXISTS verified boolean NOT NULL DEFAULT false,
	ADD COLUMN IF NOT EXISTS verification_token text NOT NULL DEFAULT substr(replace(translate(encode(decode(MD5(random()::text), 'hex'),'base64'),'/+','=='),'=',''), 1, 10);

CREATE UNIQUE INDEX IF NOT EXISTS idx_identities_verified ON identities (organization_id, verification_token) WHERE (deleted_at IS NULL);`
			_, err := tx.Exec(stmt)
			return err
		},
	}
}

func cleanCrossOrgGroupMemberships() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-09-22T11:00",
		Migrate: func(tx migrator.DB) error {
			// go through all the group members and make sure they belong to the same org as the group
			stmt := `SELECT identity_id, group_id FROM identities_groups`
			rows, err := tx.Query(stmt)
			if err != nil {
				return fmt.Errorf("select all identity groups: %w", err)
			}

			type identityGroup struct {
				IdentityID uid.ID
				GroupID    uid.ID
			}

			var idGroups []identityGroup
			for rows.Next() {
				var idGroup identityGroup
				if err := rows.Scan(&idGroup.IdentityID, &idGroup.GroupID); err != nil {
					return fmt.Errorf("scan identity and group: %w", err)
				}

				idGroups = append(idGroups, idGroup)
			}

			if err := rows.Close(); err != nil {
				return fmt.Errorf("close read identity group rows: %w", err)
			}

			for _, idGroup := range idGroups {
				var identityOrgID uid.ID
				err := tx.QueryRow(`SELECT organization_id FROM identities WHERE id = ?`, idGroup.IdentityID).Scan(&identityOrgID)
				if err != nil {
					return fmt.Errorf("select identity id: %w", err)
				}

				var groupOrgID uid.ID
				err = tx.QueryRow(`SELECT organization_id FROM groups WHERE id = ?`, idGroup.GroupID).Scan(&groupOrgID)
				if err != nil {
					return fmt.Errorf("select group id: %w", err)
				}

				if identityOrgID != groupOrgID {
					_, err := tx.Exec(`DELETE FROM identities_groups WHERE identity_id = ? AND group_id = ?`, idGroup.IdentityID, idGroup.GroupID)
					if err != nil {
						return fmt.Errorf("delete bad relation: %w", err)
					}
				}
			}

			return nil
		},
	}
}

func fixProviderUserIndex() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-09-22T13:00:00",
		Migrate: func(tx migrator.DB) error {
			stmt := `
ALTER TABLE provider_users DROP CONSTRAINT IF EXISTS provider_users_pkey;
ALTER TABLE provider_users ADD CONSTRAINT
    provider_users_pkey PRIMARY KEY (provider_id, identity_id);
`
			_, err := tx.Exec(stmt)
			return err
		},
	}
}

func removeDotFromDestinationName() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-10-04T11:44",
		Migrate: func(tx migrator.DB) error {
			type idName struct {
				id   uid.ID
				name string
			}

			rows, err := tx.Query(`SELECT id, name FROM destinations WHERE name LIKE '%.%'`)
			if err != nil {
				return err
			}
			defer rows.Close()
			var toRename []idName
			for rows.Next() {
				pair := idName{}
				if err := rows.Scan(&pair.id, &pair.name); err != nil {
					return err
				}
				toRename = append(toRename, pair)
			}
			if err := rows.Err(); err != nil {
				return err
			}
			for _, item := range toRename {
				item.name = strings.ReplaceAll(item.name, ".", "_")
				_, err := tx.Exec(`UPDATE destinations SET name = ? WHERE id = ?`,
					item.name, item.id)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
}

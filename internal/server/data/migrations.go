package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/uid"
)

func migrate(db *gorm.DB) error {
	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		// rename grants.identity -> grants.subject
		{
			ID: "202203231621", // date the migration was created
			Migrate: func(tx *gorm.DB) error {
				logging.Infof("running migration 202203231621")
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
				logging.Infof("running migration 202203241643")
				if tx.Migrator().HasColumn(&models.AccessKey{}, "key") {
					return tx.Migrator().RenameColumn(&models.AccessKey{}, "key", "key_id")
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().RenameColumn(&models.AccessKey{}, "key_id", "key")
			},
		},
		// Migration for unifying user and groups as identities
		// #1284: change access key "issued_for" to a direct ID reference
		{
			ID: "202203301642",
			Migrate: func(tx *gorm.DB) error {
				if tx.Migrator().HasColumn(&models.AccessKey{}, "provider_id") {
					return nil
				}

				logging.Infof("running migration 202203301642")
				type AccessKey struct {
					models.Model
					Name      string `gorm:"uniqueIndex:,where:deleted_at is NULL"`
					IssuedFor string

					ExpiresAt         time.Time
					Extension         time.Duration
					ExtensionDeadline time.Time

					KeyID          string `gorm:"<-;uniqueIndex:,where:deleted_at is NULL"`
					Secret         string `gorm:"-"`
					SecretChecksum []byte
				}

				var keys []AccessKey

				err := db.Find(&keys).Error
				if err != nil {
					return fmt.Errorf("migrating access key identities: %w", err)
				}

				// need to manually override this table
				if err := tx.Migrator().DropTable("access_keys"); err != nil {
					return err
				}

				// new table with the updated types
				if err := tx.Migrator().CreateTable(&models.AccessKey{}); err != nil {
					return err
				}

				for _, key := range keys {
					key.IssuedFor = strings.TrimPrefix(key.IssuedFor, "u:")
					key.IssuedFor = strings.TrimPrefix(key.IssuedFor, "m:")

					iss, err := uid.Parse([]byte(key.IssuedFor))
					if err != nil {
						return fmt.Errorf("converting access keys: %w", err)
					}

					convertedKey := &models.AccessKey{
						Model:             key.Model,
						Name:              key.Name,
						IssuedFor:         iss,
						ExpiresAt:         key.ExpiresAt,
						Extension:         key.Extension,
						ExtensionDeadline: key.ExtensionDeadline,
						KeyID:             key.KeyID,
						// key secret not needed
						SecretChecksum: key.SecretChecksum,
					}

					if err := SaveAccessKey(db, convertedKey); err != nil {
						return fmt.Errorf("save converted key: %w", err)
					}
				}

				return nil
			},
			// unable to rollback, context is lost
		},
		// #1284: change credential "identity" to a direct ID reference
		{
			ID: "202203301652",
			Migrate: func(tx *gorm.DB) error {
				logging.Infof("running migration 202203301652")
				type Credential struct {
					models.Model

					Identity        string `gorm:"<-;uniqueIndex:,where:deleted_at is NULL"`
					PasswordHash    []byte `validate:"required"`
					OneTimePassword bool
				}

				var creds []Credential

				if err := db.Find(&creds).Error; err != nil {
					return fmt.Errorf("migrating creds: %w", err)
				}

				// need to manually override this table
				if err := tx.Migrator().DropTable("credentials"); err != nil {
					return err
				}

				// new table with the updated type
				if err := tx.Migrator().CreateTable(&models.Credential{}); err != nil {
					return err
				}

				for _, cred := range creds {
					cred.Identity = strings.TrimPrefix(cred.Identity, "u:")
					cred.Identity = strings.TrimPrefix(cred.Identity, "m:")

					identityID, err := uid.Parse([]byte(cred.Identity))
					if err != nil {
						return fmt.Errorf("converting credential identity: %w", err)
					}

					convertedCred := &models.Credential{
						Model:           cred.Model,
						IdentityID:      identityID,
						PasswordHash:    cred.PasswordHash,
						OneTimePassword: cred.OneTimePassword,
					}

					if err := CreateCredential(db, convertedCred); err != nil {
						return fmt.Errorf("create converted cred: %w", err)
					}
				}

				return nil
			},
			// unable to rollback, context is lost
		},
		// #1284: change users to identities
		{
			ID: "202203301643",
			Migrate: func(tx *gorm.DB) error {
				logging.Infof("running migration 202203301643")
				if tx.Migrator().HasTable("users") {
					return tx.Migrator().RenameTable("users", "identities")
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().RenameTable("identities", "users")
			},
		},
		// #1284: set identity user from email
		{
			ID: "202203301645",
			Migrate: func(tx *gorm.DB) error {
				logging.Infof("running migration 202203301645")
				if tx.Migrator().HasColumn(&models.Identity{}, "email") {
					return tx.Migrator().RenameColumn(&models.Identity{}, "email", "name")
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().RenameColumn(&models.Identity{}, "name", "email")
			},
		},
		// #1284: migrate machines to the identities table
		{
			ID: "202203301646",
			Migrate: func(tx *gorm.DB) error {
				logging.Infof("running migration 202203301646")
				if tx.Migrator().HasTable("machines") {
					type Machine struct {
						models.Model

						Name        string `gorm:"uniqueIndex:,where:deleted_at is NULL"`
						Description string
						LastSeenAt  time.Time
					}

					var machines []Machine

					if err := db.Find(&machines).Error; err != nil {
						return fmt.Errorf("migrating machine identities: %w", err)
					}

					for _, machine := range machines {
						identity := &models.Identity{
							Model:      machine.Model,
							Name:       machine.Name,
							LastSeenAt: machine.LastSeenAt,
						}

						if err := SaveIdentity(db, identity); err != nil {
							return fmt.Errorf("saving migrated machine identity: %w", err)
						}
					}
				}

				return nil
			},
			// unable to rollback, context is lost
		},
		// #1284: migrate identity grants
		{
			ID: "202203301647",
			Migrate: func(tx *gorm.DB) error {
				logging.Infof("running migration 202203301647")
				if tx.Migrator().HasTable("machines") {
					grants, err := ListGrants(db)
					if err != nil {
						return err
					}

					for i := range grants {
						identitySubject := grants[i].Subject.String()
						if strings.HasPrefix(identitySubject, "u:") || strings.HasPrefix(identitySubject, "m:") {
							identitySubject = strings.TrimPrefix(identitySubject, "u:")
							identitySubject = strings.TrimPrefix(identitySubject, "m:")
							grants[i].Subject = uid.PolymorphicID(fmt.Sprintf("%s:%s", "i", identitySubject))
							if err := save(db, &grants[i]); err != nil {
								return fmt.Errorf("update identity grant: %w", err)
							}
						}
					}
				}
				return nil
			},
			// unable to rollback, context is lost
		},
		// #1284: migrate machines to the identities
		{
			ID: "202203301648",
			Migrate: func(tx *gorm.DB) error {
				logging.Infof("running migration 202203301648")
				if tx.Migrator().HasTable("machines") {
					if err := tx.Migrator().DropTable("machines"); err != nil {
						return err
					}
				}
				return nil
			},
			// this change cannot be rolled back
		},
		// #1449: access key name can't have whitespace
		{
			ID: "202204061643",
			Migrate: func(tx *gorm.DB) error {
				logging.Infof("running migration 202204061643")
				if tx.Migrator().HasTable("access_keys") {
					keys, err := ListAccessKeys(db)
					if err != nil {
						return err
					}

					for i := range keys {
						if strings.Contains(keys[i].Name, " ") {
							keys[i].Name = strings.ReplaceAll(keys[i].Name, " ", "-")
							err := SaveAccessKey(db, &keys[i])
							if err != nil {
								return err
							}
						}
					}
				}

				return nil
			},
			// context lost, cannot roll back
		},
		// unify users
		{
			ID: "202204111503",
			Migrate: func(tx *gorm.DB) error {
				logging.Infof("running migration 202204111503")

				type Identity struct {
					models.Model

					ProviderID uid.ID
					Name       string `gorm:"uniqueIndex:idx_identities_name_provider_id,where:deleted_at is NULL"`
				}
				identityTable := &Identity{}
				logging.Debugf("starting migration")

				logging.Debugf("checking provider_id column")
				if tx.Migrator().HasColumn(identityTable, "provider_id") {
					logging.Debugf("has provider_id column")

					// need to select only these fields from the providers
					// we dont have the database encryption key for the client secret at this point
					var providers []models.Provider
					err := tx.Select("id", "name").Find(&providers).Error
					if err != nil {
						return err
					}

					providerIDs := make(map[string]uid.ID)
					for _, provider := range providers {
						providerIDs[provider.Name] = provider.ID
					}

					infraProviderID := providerIDs["infra"]

					users, err := ListIdentities(db, func(db *gorm.DB) *gorm.DB {
						return db.Where("provider_id != ?", infraProviderID)
					})
					if err != nil {
						return err
					}

					for _, user := range users {
						logging.Debugf("migrating user %s", user.ID.String())
						newUser, err := GetIdentity(db, ByName(user.Name), func(db *gorm.DB) *gorm.DB {
							return db.Where("provider_id = ?", infraProviderID)
						})
						if err != nil {
							if errors.Is(err, internal.ErrNotFound) {
								logging.Debugf("skipping user migration for user not in infra provider")
								continue
							}
							return err
						}

						logging.Debugf("updating grants for user %s", user.ID.String())
						// update all grants to point to the new user
						err = tx.Exec("update grants set subject = ? where subject = ?", newUser.PolyID(), user.PolyID()).Error
						if err != nil {
							return err
						}

						logging.Debugf("deleting user %s", user.ID.String())
						// delete the duplicate user
						err = tx.Exec("delete from identities where id = ?", user.ID).Error
						if err != nil {
							return err
						}
					}

					// remove provider_id field
					logging.Debugf("removing provider_id foreign key")
					err = tx.Migrator().DropConstraint(identityTable, "fk_providers_users")
					if err != nil {
						return err
					}

					logging.Debugf("removing provider_id field")
					err = tx.Migrator().DropColumn(identityTable, "provider_id")
					if err != nil {
						return err
					}
				}

				if tx.Migrator().HasIndex(identityTable, "idx_identities_name_provider_id") {
					logging.Debugf("has idx_identities_name_provider_id index")
					err := tx.Migrator().DropIndex(identityTable, "idx_identities_name_provider_id")
					if err != nil {
						return fmt.Errorf("migrate identity index: %w", err)
					}
				}

				return nil
			},
			// context lost, cannot roll back
		},
		// #1518: rename models.Settings.SetupRequired to models.Settings.SignupRequired
		{
			ID: "202204181613",
			Migrate: func(tx *gorm.DB) error {
				return tx.Migrator().RenameColumn(&models.Settings{}, "setup_required", "signup_required")
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().RenameColumn(&models.Settings{}, "signup_required", "setup_required")
			},
		},
		// make Settings use EncryptedAtRest fields
		{
			ID: "202204211705",
			Migrate: func(tx *gorm.DB) error {
				type Settings struct {
					models.Model
					PrivateJWK models.EncryptedAtRestBytes
				}

				// to convert the plaintext field to encrypted, load it with SkipSymmetricKey, then save without it.
				models.SkipSymmetricKey = true
				settings := []Settings{}
				err := db.Model(Settings{}).Find(&settings).Error
				if err != nil {
					models.SkipSymmetricKey = false
					return err
				}

				models.SkipSymmetricKey = false
				for _, setting := range settings {
					if err := db.Save(setting).Error; err != nil {
						return err
					}
				}

				return nil
			},
		},
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
		// drop old Groups constraint; new constraint will be created automatically
		{
			ID: "202206081027",
			Migrate: func(tx *gorm.DB) error {
				_ = tx.Migrator().DropConstraint(&models.Group{}, "idx_groups_name_provider_id")
				return nil
			},
		},
		addKindToProviders(),
		dropCertificateTables(),
		addAuthURLAndScopeToProviders(),
		// next one here
	})

	// TODO: why? isn't this already called by NewDB?
	m.InitSchema(preMigrate)

	if err := m.Migrate(); err != nil {
		return err
	}

	// initializeSchema so that for simple things like adding db fields
	// we don't necessarily need to do a migration.
	// TODO: why not do this before migrate in all cases?
	return initializeSchema(db)
}

func preMigrate(db *gorm.DB) error {
	if db.Migrator().HasTable("providers") {
		// don't initialize the schema if tables already exist.
		return nil
	}

	return initializeSchema(db)
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
	}

	for _, table := range tables {
		if err := db.AutoMigrate(table); err != nil {
			return err
		}
	}

	return nil
}

// #2294: set the provider kind on existing providers
func addKindToProviders() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202206151027",
		Migrate: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(&models.Provider{}, "kind") {
				logging.Debugf("migrating provider table kind")
				if err := tx.Migrator().AddColumn(&models.Provider{}, "kind"); err != nil {
					return err
				}
			}

			db := tx.Begin()
			db.Table("providers").Where("kind IS NULL AND name = ?", "infra").Update("kind", models.InfraKind)
			db.Table("providers").Where("kind IS NULL").Update("kind", models.OktaKind)

			return db.Commit().Error
		},
	}
}

// #2276: drop unused certificate tables
func dropCertificateTables() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202206161733",
		Migrate: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable("trusted_certificates", "root_certificates")
		},
	}
}

// #2353: store auth URL and scopes to provider
func addAuthURLAndScopeToProviders() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202206281027",
		Migrate: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(&models.Provider{}, "scopes") {
				logging.S.Debug("migrating provider table auth URL and scopes")
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
					if providerModels[i].Kind == models.InfraKind || providerModels[i].Name == models.InternalInfraProviderName {
						continue
					}

					logging.S.Debugf("migrating %s provider", providerModels[i].Name)

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

package data

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
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
		// Migration for unifying user and groups as identities
		// #1284: change access key "issued_for" to a direct ID reference
		{
			ID: "202203301642",
			Migrate: func(tx *gorm.DB) error {
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

					iss, err := uid.ParseString(key.IssuedFor)
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
				type Credential struct {
					models.Model

					Identity            string `gorm:"<-;uniqueIndex:,where:deleted_at is NULL"`
					PasswordHash        []byte `validate:"required"`
					OneTimePassword     bool
					OneTimePasswordUsed bool
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

					identityID, err := uid.ParseString(cred.Identity)
					if err != nil {
						return fmt.Errorf("converting credential identity: %w", err)
					}

					convertedCred := &models.Credential{
						Model:               cred.Model,
						IdentityID:          identityID,
						PasswordHash:        cred.PasswordHash,
						OneTimePassword:     cred.OneTimePassword,
						OneTimePasswordUsed: cred.OneTimePasswordUsed,
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
				if tx.Migrator().HasTable("users") {
					return tx.Migrator().RenameTable("users", "identities")
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().RenameTable("identities", "users")
			},
		},
		// #1284: set identity user kind
		{
			ID: "202203301644",
			Migrate: func(tx *gorm.DB) error {
				if tx.Migrator().HasColumn(&models.Identity{}, "email") {
					if err := tx.Migrator().AddColumn(&models.Identity{}, "kind"); err != nil {
						return err
					}
					// everything in the identity table should be a user at this point
					return db.Model(&models.Identity{}).Where("1 = 1").Update("kind", "user").Error
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropColumn(&models.Identity{}, "kind")
			},
		},
		// #1284: set identity user from email
		{
			ID: "202203301645",
			Migrate: func(tx *gorm.DB) error {
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
							Kind:       models.MachineKind,
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
			Rollback: func(tx *gorm.DB) error {
				return DeleteIdentities(db, ByIdentityKind(models.MachineKind))
			},
		},
		// #1284: migrate identity grants
		{
			ID: "202203301647",
			Migrate: func(tx *gorm.DB) error {
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
		// next one here
	})

	m.InitSchema(func(db *gorm.DB) error {
		// TODO: can optionally remove this skip after any existing users have migrated.
		if db.Migrator().HasTable("providers") {
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
		&models.Identity{},
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

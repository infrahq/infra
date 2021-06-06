package server

import (
	"errors"
	"log"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/infrahq/infra/internal/generate"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type User struct {
	ID          string       `gorm:"primaryKey"`
	Created     int64        `json:"created" gorm:"autoCreateTime"`
	Updated     int64        `json:"updated" gorm:"autoUpdateTime"`
	Email       string       `json:"email" gorm:"unique"`
	Password    []byte       `json:"-"`
	Providers   []Provider   `json:"providers" gorm:"many2many:users_providers"`
	Permissions []Permission `json:"permissions" gorm:"foreignKey:UserEmail;references:Email"`
}

type Permission struct {
	ID        string `gorm:"primaryKey"`
	Created   int64  `json:"created" yaml:"-" gorm:"autoCreateTime"`
	Updated   int64  `json:"updated" yaml:"-" gorm:"autoUpdateTime"`
	UserEmail string `json:"-" yaml:"user"`
	User      User   `json:"user" yaml:"-" gorm:"foreignKey:UserEmail;references:Email"`
	RoleName  string `json:"-" yaml:"role"`
	Role      Role   `json:"role" yaml:"-" gorm:"foreignKey:RoleName;references:Name"`
}

type Provider struct {
	ID           string  `gorm:"primaryKey"`
	Created      int64   `json:"created" yaml:"-" gorm:"autoCreateTime"`
	Updated      int64   `json:"updated" yaml:"-" gorm:"autoUpdateTime"`
	Kind         string  `json:"kind" yaml:"kind"`
	Domain       string  `json:"domain" yaml:"domain,omitempty"`
	ClientID     string  `json:"clientID" yaml:"clientID,omitempty"`
	ClientSecret string  `json:"-" yaml:"clientSecret,omitempty"`
	ApiToken     string  `json:"-" yaml:"apiToken,omitempty"`
	Users        []*User `json:"-" yaml:"-" gorm:"many2many:users_providers"`
}

type Role struct {
	ID             string `gorm:"primaryKey"`
	Created        int64  `json:"created" yaml:"-" gorm:"autoCreateTime"`
	Updated        int64  `json:"updated" yaml:"-" gorm:"autoUpdateTime"`
	Name           string `json:"name" yaml:"name" gorm:"unique"`
	Description    string `json:"description" yaml:"description"`
	KubernetesRole string `json:"kubernetesRole" yaml:"kubernetesRole"`
}

type Settings struct {
	ID            string `gorm:"primaryKey"`
	Created       int64  `json:"-" yaml:"-" gorm:"autoCreateTime"`
	Updated       int64  `json:"-" yaml:"-" gorm:"autoUpdateTime"`
	Domain        string `json:"-" yaml:"domain,omitempty"`
	TokenSecret   string `json:"-" yaml:"tokenSecret,omitempty"`
	DisableSignup bool   `json:"disableSignup" yaml:"disableSignup,omitempty"`
}

var (
	DefaultRoleView  = "view"
	DefaultRoleEdit  = "edit"
	DefaultRoleAdmin = "admin"

	DefaultInfraProviderKind = "infra"
)

var DefaultRoles = []Role{
	{
		Name:           "view",
		Description:    "Read most resources",
		KubernetesRole: "view",
	},
	{
		Name:           "edit",
		Description:    "Read & write most resources",
		KubernetesRole: "edit",
	},
	{
		Name:           "admin",
		Description:    "Read & write all resources",
		KubernetesRole: "admin",
	},
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return
}

func (u *User) BeforeDelete(tx *gorm.DB) error {
	return tx.Model(u).Association("Providers").Clear()
}

func (p *Permission) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return
}

func (p *Provider) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return
}

func (p *Provider) BeforeDelete(tx *gorm.DB) error {
	var users []User
	if err := tx.Model(p).Association("Users").Find(&users); err != nil {
		return err
	}

	for _, u := range users {
		p.DeleteUser(tx, &u)
	}

	return nil
}

// CreateUser will create a user and associate them with the provider
// If the user already exists, they will not be created, instead an association
// will be added instead
func (p *Provider) CreateUser(db *gorm.DB, email string, password string) error {
	var user User
	var hashedPassword []byte
	var err error

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.FirstOrCreate(&user, &User{Email: email}).Error; err != nil {
			return err
		}

		if password != "" {
			hashedPassword, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return errors.New("could not create user")
			}

			user.Password = hashedPassword

			if err := tx.Save(&user).Error; err != nil {
				return err
			}
		}

		if tx.Model(&user).Where(&Provider{ID: p.ID}).Association("Providers").Count() == 0 {
			tx.Model(&user).Where(&Provider{ID: p.ID}).Association("Providers").Append(p)
		}

		return nil
	})
}

// Delete will delete a user's association with a provider
// If this is their only provider, then the user will be deleted entirely
// TODO (jmorganca): wrap this in a transaction or at least find out why
// there seems to cause a bug when used in a nested transaction
func (p *Provider) DeleteUser(db *gorm.DB, u *User) error {
	var user User

	if err := db.Where(&User{Email: u.Email}).First(&user).Error; err != nil {
		return err
	}

	if err := db.Model(&user).Association("Providers").Delete(p); err != nil {
		return err
	}

	if db.Model(&user).Association("Providers").Count() == 0 {
		if err := db.Delete(&user).Error; err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) SyncUsers(db *gorm.DB, emails []string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Create users in provider
		for _, email := range emails {
			if err := p.CreateUser(tx, email, ""); err != nil {
				return err
			}
		}

		// Delete users not in provider
		var toDelete []User
		if err := tx.Not("email IN ?", emails).Find(&toDelete).Error; err != nil {
			return err
		}

		for _, td := range toDelete {
			p.DeleteUser(tx, &td)
		}

		return nil
	})
}

func (r *Role) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return
}

func (s *Settings) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return
}

func (s *Settings) BeforeSave(tx *gorm.DB) error {
	if s.TokenSecret == "" {
		s.TokenSecret = generate.RandString(32)
	}
	return nil
}

type Config struct {
	Providers   []Provider   `yaml:"providers"`
	Permissions []Permission `yaml:"permissions"`
}

func ImportProviders(db *gorm.DB, providers []Provider) error {
	var idsToKeep []string
	for _, p := range providers {
		err := db.FirstOrCreate(&p, &Provider{Kind: p.Kind, Domain: p.Domain}).Error
		if err != nil {
			return err
		}

		idsToKeep = append(idsToKeep, p.ID)
	}

	var toDelete []Provider
	if err := db.Not(idsToKeep).Not(&Provider{Kind: DefaultInfraProviderKind}).Find(&toDelete).Error; err != nil {
		return err
	}

	for _, td := range toDelete {
		if err := db.Delete(&td).Error; err != nil {
			return err
		}
	}
	return nil
}

func ImportPermissions(db *gorm.DB, permissions []Permission) error {
	// Create permissions that don't exist
	var idsToKeep []string
	for _, p := range permissions {
		err := db.FirstOrCreate(&p, &p).Error
		if err != nil {
			return err
		}

		idsToKeep = append(idsToKeep, p.ID)
	}

	return db.Not(idsToKeep).Delete(Permission{}).Error
}

func ImportConfig(db *gorm.DB, bs []byte) error {
	var config Config
	err := yaml.Unmarshal(bs, &config)
	if err != nil {
		return err
	}

	var raw map[string]interface{}
	err = yaml.Unmarshal(bs, &raw)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if _, ok := raw["providers"]; ok {
			if err = ImportProviders(tx, config.Providers); err != nil {
				return err
			}
		}

		if _, ok := raw["permissions"]; ok {
			if err = ImportPermissions(tx, config.Permissions); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetConfig serializes a config from the database
func ExportConfig(db *gorm.DB) ([]byte, error) {
	var config Config

	err := db.Transaction(func(tx *gorm.DB) error {
		err := db.Find(&config.Providers).Error
		if err != nil {
			return err
		}

		err = db.Find(&config.Permissions).Error
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	bs, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}

	return bs, nil
}

func NewDB(dbpath string) (*gorm.DB, error) {
	if err := os.MkdirAll(path.Dir(dbpath), os.ModePerm); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dbpath), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Error,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	})

	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&User{})
	db.AutoMigrate(&Provider{})
	db.AutoMigrate(&Permission{})
	db.AutoMigrate(&Role{})
	db.AutoMigrate(&Settings{})

	// Add default roles
	for _, p := range DefaultRoles {
		err := db.Where(&Role{Name: p.Name}).First(&p).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		db.Save(&p)
	}

	// Add default provider
	infraProvider := Provider{Kind: DefaultInfraProviderKind}
	err = db.FirstOrCreate(&infraProvider, Provider{Kind: DefaultInfraProviderKind}).Error
	if err != nil {
		return nil, err
	}

	// Add default settings
	err = db.FirstOrCreate(&Settings{}, &Settings{}).Error
	if err != nil {
		return nil, err
	}

	return db, nil
}

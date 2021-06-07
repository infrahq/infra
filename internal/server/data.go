package server

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"log"
	"os"
	"path"
	"time"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/okta"
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
	Providers   []Provider   `json:"providers,omitempty" gorm:"many2many:users_providers"`
	Permissions []Permission `json:"permissions,omitempty" gorm:"foreignKey:UserEmail;references:Email"`
}

type Permission struct {
	ID        string `gorm:"primaryKey"`
	Created   int64  `json:"created" yaml:"-" gorm:"autoCreateTime"`
	Updated   int64  `json:"updated" yaml:"-" gorm:"autoUpdateTime"`
	UserEmail string `json:"-" yaml:"user"`
	User      User   `json:"user,omitempty" yaml:"-" gorm:"foreignKey:UserEmail;references:Email"`
	RoleName  string `json:"-" yaml:"role"`
	Role      Role   `json:"role,omitempty" yaml:"-" gorm:"foreignKey:RoleName;references:Name"`
}

type Provider struct {
	ID           string `gorm:"primaryKey"`
	Created      int64  `json:"created" yaml:"-" gorm:"autoCreateTime"`
	Updated      int64  `json:"updated" yaml:"-" gorm:"autoUpdateTime"`
	Kind         string `json:"kind" yaml:"kind"`
	Domain       string `json:"domain" yaml:"domain,omitempty" gorm:"unique"`
	ClientID     string `json:"clientID" yaml:"clientID,omitempty"`
	ClientSecret string `json:"-" yaml:"clientSecret,omitempty"`
	ApiToken     string `json:"-" yaml:"apiToken,omitempty"`
	Users        []User `json:"users,omitempty" yaml:"-" gorm:"many2many:users_providers"`
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
	JWTSecret     string `json:"-" yaml:"jwtSecret,omitempty"`
	DisableSignup bool   `json:"disableSignup" yaml:"disableSignup,omitempty"`
}

type Token struct {
	ID      string `gorm:"primaryKey"`
	Created int64  `json:"created" gorm:"autoCreateTime"`
	Updated int64  `json:"updated" gorm:"autoUpdateTime"`
	Expires int64  `json:"expires"`
	Secret  []byte `json:"-" gorm:"autoCreateTime"`

	UserID string
	User   User `json:"-"`
}

var (
	DefaultRoleView  = "view"
	DefaultRoleEdit  = "edit"
	DefaultRoleAdmin = "admin"

	DefaultInfraProviderKind = "infra"
)

var DefaultRoles = []Role{
	{
		Name:           "readonly",
		Description:    "Read most resources",
		KubernetesRole: "view",
	},
	{
		Name:           "editor",
		Description:    "Read & write most resources",
		KubernetesRole: "edit",
	},
	{
		Name:           "superadmin",
		Description:    "Full access to all resources",
		KubernetesRole: "admin",
	},
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = generate.RandString(12)
	}

	return
}

// TODO (jmorganca): use foreign constraints instead?
func (u *User) BeforeDelete(tx *gorm.DB) error {
	err := tx.Model(u).Association("Providers").Clear()
	if err != nil {
		return err
	}

	return tx.Where(&Token{UserID: u.ID}).Delete(&Token{}).Error
}

func (p *Permission) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		p.ID = generate.RandString(12)
	}
	return
}

func (p *Provider) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		p.ID = generate.RandString(12)
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
func (p *Provider) CreateUser(db *gorm.DB, user *User, email string, password string) error {
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
		if err := db.Select("Tokens").Delete(&user).Error; err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) SyncUsers(db *gorm.DB) error {
	var emails []string
	var err error

	switch p.Kind {
	case "okta":
		emails, err = okta.Emails(p.Domain, p.ClientID, p.ApiToken)
		if err != nil {
			return err
		}
	case "infra":
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// Create users in provider
		for _, email := range emails {
			if err := p.CreateUser(tx, &User{}, email, ""); err != nil {
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
		r.ID = generate.RandString(12)
	}
	return
}

func (s *Settings) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == "" {
		s.ID = generate.RandString(12)
	}
	return
}

func (s *Settings) BeforeSave(tx *gorm.DB) error {
	if s.JWTSecret == "" {
		s.JWTSecret = generate.RandString(32)
	}
	return nil
}

func (t *Token) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == "" {
		t.ID = generate.RandString(12)
	}

	// TODO (jmorganca): 24 hours may be too long or too short for some teams
	// this should be customizable in settings or limited by the provider
	if t.Expires == 0 {
		t.Expires = time.Now().Add(time.Hour * 24).Unix()
	}
	return
}

func (t *Token) CheckSecret(secret string) (err error) {
	h := sha256.New()
	h.Write([]byte(secret))
	if !bytes.Equal(t.Secret, h.Sum(nil)) {
		return errors.New("could not verify token secret")
	}

	return nil
}

func NewToken(db *gorm.DB, userID string, token *Token) (secret string, err error) {
	secret = generate.RandString(24)

	h := sha256.New()
	h.Write([]byte(secret))
	token.UserID = userID
	token.Secret = h.Sum(nil)

	err = db.Create(token).Error
	if err != nil {
		return "", err
	}

	return
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
	db.AutoMigrate(&Token{})

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

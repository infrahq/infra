package server

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"path"
	"time"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/okta"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/yaml.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var initialConfig Config

type Resource struct {
	ID                 string `gorm:"primaryKey"`
	Created            int64  `json:"created" gorm:"autoCreateTime"`
	Updated            int64  `json:"updated" gorm:"autoUpdateTime"`
	Kind               string `json:"kind"`
	Name               string `json:"name"`
	KubernetesCA       string `json:"kubernetesCA"`
	KubernetesEndpoint string `json:"kubernetesEndpoint"`
}

type Grant struct {
	ID           string   `gorm:"primaryKey"`
	Created      int64    `json:"created" yaml:"-" gorm:"autoCreateTime"`
	Updated      int64    `json:"updated" yaml:"-" gorm:"autoUpdateTime"`
	UserEmail    string   `json:"-" yaml:"user"`
	User         User     `json:"user,omitempty" yaml:"-" gorm:"foreignKey:UserEmail;references:Email"`
	RoleName     string   `json:"-" yaml:"role"`
	Role         Role     `json:"role,omitempty" yaml:"-" gorm:"foreignKey:RoleName;references:Name"`
	ResourceName string   `json:"-" yaml:"resource"`
	Resource     Resource `json:"resource,omitempty" yaml:"-" gorm:"foreignKey:ResourceName;references:Name"`
}

type Role struct {
	ID             string `gorm:"primaryKey"`
	Created        int64  `json:"created" yaml:"-" gorm:"autoCreateTime"`
	Updated        int64  `json:"updated" yaml:"-" gorm:"autoUpdateTime"`
	Name           string `json:"name" yaml:"name" gorm:"unique"`
	Description    string `json:"description" yaml:"description"`
	Default        bool   `json:"default"`
	KubernetesRole string `json:"kubernetesRole" yaml:"kubernetesRole"`
}

type User struct {
	ID        string     `gorm:"primaryKey"`
	Created   int64      `json:"created" gorm:"autoCreateTime"`
	Updated   int64      `json:"updated" gorm:"autoUpdateTime"`
	Email     string     `json:"email" gorm:"unique"`
	Password  []byte     `json:"-"`
	Providers []Provider `json:"providers,omitempty" gorm:"many2many:users_providers"`
	Grants    []Grant    `json:"grants,omitempty" gorm:"foreignKey:UserEmail;references:Email"`
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
	Users        []User `json:"-" yaml:"-" gorm:"many2many:users_providers"`
}

type Settings struct {
	ID            string `gorm:"primaryKey"`
	Created       int64  `json:"-" yaml:"-" gorm:"autoCreateTime"`
	Updated       int64  `json:"-" yaml:"-" gorm:"autoUpdateTime"`
	DisableSignup bool   `json:"disableSignup" yaml:"disableSignup,omitempty"`
	PrivateJWK    []byte
	PublicJWK     []byte
	AddToken      string
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

type APIKey struct {
	ID      string `gorm:"primaryKey"`
	Created int64  `json:"created" gorm:"autoCreateTime"`
	Updated int64  `json:"updated" gorm:"autoUpdateTime"`
	Key     string `json:"key"`
}

var (
	DefaultRoleView  = "view"
	DefaultRoleEdit  = "edit"
	DefaultRoleAdmin = "admin"

	DefaultInfraProviderKind = "infra"
)

// TODO (jmorganca): encode actual rbac rules here
var Roles = []Role{
	{
		Name:           "kubernetes.viewer",
		Description:    "Read most resources",
		KubernetesRole: "view",
		Default:        true,
	},
	{
		Name:           "kubernetes.editor",
		Description:    "Read & write most resources",
		KubernetesRole: "edit",
	},
	{
		Name:           "kubernetes.admin",
		Description:    "Full access to all resources",
		KubernetesRole: "cluster-admin",
	},
	{
		Name:        "infra.member",
		Description: "Read-only access to Infra server. Ability to log in and connect resources Infra.",
		Default:     true,
	},
	{
		Name:        "infra.owner",
		Description: "Take any action in Infra",
	},
}

func (r *Resource) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = generate.RandString(12)
	}

	return
}

func (u *Resource) AfterCreate(tx *gorm.DB) (err error) {
	_, err = ApplyGrants(tx, initialConfig.Grants)
	return
}

func (r *Resource) BeforeDelete(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = generate.RandString(12)
	}

	return tx.Where(&Grant{ResourceName: r.Name}).Delete(&Grant{}).Error
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = generate.RandString(12)
	}

	// Add default member grant
	// TODO (merge me with larger default grant logic for POST /v1/grants handler)
	count := tx.Model(u).Where("role_name LIKE ?", "infra.%").Association("Grants").Count()
	if count == 0 {
		tx.Create(&Grant{
			UserEmail:    u.Email,
			RoleName:     "infra.member",
			ResourceName: "infra",
		})
	}

	return
}

func (u *User) AfterCreate(tx *gorm.DB) (err error) {
	_, err = ApplyGrants(tx, initialConfig.Grants)
	return
}

// TODO (jmorganca): use foreign constraints instead?
func (u *User) BeforeDelete(tx *gorm.DB) error {
	// Delete from provider
	err := tx.Model(u).Association("Providers").Clear()
	if err != nil {
		return err
	}

	// Delete user tokens
	err = tx.Where(&Token{UserID: u.ID}).Delete(&Token{}).Error
	if err != nil {
		return err
	}

	// Delete grants
	return tx.Where(&Grant{UserEmail: u.Email}).Delete(&Grant{}).Error
}

func (g *Grant) BeforeCreate(tx *gorm.DB) (err error) {
	if g.ID == "" {
		g.ID = generate.RandString(12)
	}

	// Choose a default role if not specified
	if g.RoleName == "" {
		if g.ResourceName == "infra" {
			g.RoleName = "infra.member"
			return
		}

		g.RoleName = "kubernetes.viewer"
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
func (p *Provider) CreateUser(db *gorm.DB, user *User, email string, password string, role string) error {
	var hashedPassword []byte
	var err error

	return db.Transaction(func(tx *gorm.DB) error {
		grant := Grant{UserEmail: email, RoleName: role, ResourceName: "infra"}
		if err := tx.Create(&grant).Error; err != nil {
			return err
		}

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
			if err := p.CreateUser(tx, &User{}, email, "", "infra.member"); err != nil {
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
	if len(s.PublicJWK) == 0 || len(s.PrivateJWK) == 0 {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return err
		}

		priv := jose.JSONWebKey{Key: key, KeyID: "", Algorithm: string(jose.RS256), Use: "sig"}
		thumb, err := priv.Thumbprint(crypto.SHA256)
		if err != nil {
			return err
		}
		kid := base64.URLEncoding.EncodeToString(thumb)
		priv.KeyID = kid
		pub := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: string(jose.RS256), Use: "sig"}

		privJS, err := priv.MarshalJSON()
		if err != nil {
			return err
		}

		pubJS, err := pub.MarshalJSON()
		if err != nil {
			return err
		}

		s.PrivateJWK = privJS
		s.PublicJWK = pubJS
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

func (a *APIKey) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == "" {
		a.ID = generate.RandString(12)
	}

	if a.Key == "" {
		a.Key = generate.RandString(24)
	}
	return
}

type Config struct {
	Providers []Provider `yaml:"providers"`
	Grants    []Grant    `yaml:"grants"`
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

func ApplyGrants(db *gorm.DB, grants []Grant) ([]string, error) {
	var ids []string
	for _, g := range grants {
		err := db.FirstOrCreate(&g, &g).Error
		if err != nil {
			return nil, err
		}

		ids = append(ids, g.ID)
	}

	return ids, nil
}

func ImportGrants(db *gorm.DB, grants []Grant) error {
	// Create grants that don't exist
	idsToKeep, err := ApplyGrants(db, grants)
	if err != nil {
		return err
	}

	return db.Not(idsToKeep).Delete(Grant{}).Error
}

func ImportConfig(db *gorm.DB, bs []byte) error {
	var config Config
	err := yaml.Unmarshal(bs, &config)
	if err != nil {
		return err
	}

	initialConfig = config

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

		if _, ok := raw["grants"]; ok {
			if err = ImportGrants(tx, config.Grants); err != nil {
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

		err = db.Find(&config.Grants).Error
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

	db.AutoMigrate(&Resource{})
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Provider{})
	db.AutoMigrate(&Grant{})
	db.AutoMigrate(&Role{})
	db.AutoMigrate(&Settings{})
	db.AutoMigrate(&Token{})
	db.AutoMigrate(&APIKey{})

	// Add default roles
	for _, p := range Roles {
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

	// Add default resource (infra)
	err = db.FirstOrCreate(&Resource{}, &Resource{Name: "infra"}).Error
	if err != nil {
		return nil, err
	}

	// Add default settings
	err = db.FirstOrCreate(&Settings{}, &Settings{}).Error
	if err != nil {
		return nil, err
	}

	// Add default api key
	err = db.FirstOrCreate(&APIKey{}, &APIKey{}).Error
	if err != nil {
		return nil, err
	}

	return db, nil
}

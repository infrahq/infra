package registry

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

type User struct {
	ID       string `gorm:"primaryKey"`
	Created  int64  `json:"created" gorm:"autoCreateTime"`
	Updated  int64  `json:"updated" gorm:"autoUpdateTime"`
	Email    string `json:"email" gorm:"unique"`
	Password []byte `json:"-"`
	Admin    bool   `json:"admin"`

	Sources     []Source     `json:"sources,omitempty" gorm:"many2many:users_sources"`
	Permissions []Permission `json:"permissions,omitempty" gorm:"foreignKey:UserEmail;references:Email"`
}

type Source struct {
	ID           string `gorm:"primaryKey"`
	Created      int64  `json:"created" yaml:"-" gorm:"autoCreateTime"`
	Updated      int64  `json:"updated" yaml:"-" gorm:"autoUpdateTime"`
	Kind         string `json:"kind" yaml:"kind"`
	Domain       string `json:"domain" yaml:"domain,omitempty" gorm:"unique"`
	ClientID     string `json:"clientID" yaml:"clientID,omitempty"`
	ClientSecret string `json:"-" yaml:"clientSecret,omitempty"`
	ApiToken     string `json:"-" yaml:"apiToken,omitempty"`
	Users        []User `json:"-" yaml:"-" gorm:"many2many:users_sources"`
}

type Destination struct {
	ID                 string `gorm:"primaryKey"`
	Created            int64  `json:"created" gorm:"autoCreateTime"`
	Updated            int64  `json:"updated" gorm:"autoUpdateTime"`
	Kind               string `json:"kind"`
	Name               string `json:"name"`
	KubernetesCA       string `json:"kubernetesCA"`
	KubernetesEndpoint string `json:"kubernetesEndpoint"`
}

type Permission struct {
	ID              string      `gorm:"primaryKey"`
	Created         int64       `json:"created" yaml:"-" gorm:"autoCreateTime"`
	Updated         int64       `json:"updated" yaml:"-" gorm:"autoUpdateTime"`
	UserEmail       string      `json:"-" yaml:"user"`
	User            User        `json:"user,omitempty" yaml:"-" gorm:"foreignKey:UserEmail;references:Email"`
	DestinationName string      `json:"-" yaml:"destination"`
	Destination     Destination `json:"destination,omitempty" yaml:"-" gorm:"foreignKey:DestinationName;references:Name"`
	Role            string      `json:"role" yaml:"role"`
	FromConfig      bool        `json:"-" yaml:"-"`
}

type Settings struct {
	ID            string `gorm:"primaryKey"`
	Created       int64  `json:"-" yaml:"-" gorm:"autoCreateTime"`
	Updated       int64  `json:"-" yaml:"-" gorm:"autoUpdateTime"`
	DisableSignup bool   `json:"disableSignup" yaml:"disableSignup,omitempty"`
	PrivateJWK    []byte
	PublicJWK     []byte
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
	Name    string `json:"name" gorm:"unique"`
	Key     string `json:"key"`
}

var (
	DefaultRoleView  = "view"
	DefaultRoleEdit  = "edit"
	DefaultRoleAdmin = "admin"

	DefaultInfraSourceKind = "infra"
)

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = generate.RandString(12)
	}

	return
}

// TODO (jmorganca): use foreign constraints instead?
func (u *User) BeforeDelete(tx *gorm.DB) error {
	err := tx.Model(u).Association("Sources").Clear()
	if err != nil {
		return err
	}

	err = tx.Where(&Token{UserID: u.ID}).Delete(&Token{}).Error
	if err != nil {
		return err
	}

	return tx.Where(&Permission{UserEmail: u.Email}).Delete(&Permission{}).Error
}

func (u *User) AfterCreate(tx *gorm.DB) (err error) {
	_, err = ApplyPermissions(tx, initialConfig.Permissions)
	if err != nil {
		return err
	}

	// if user is admin, provision admin permission
	var destinations []Destination
	err = tx.Find(&destinations).Error
	if err != nil {
		return err
	}

	role := "view"
	if u.Admin {
		role = "cluster-admin"
	}

	for _, d := range destinations {
		var permission Permission
		err := tx.FirstOrCreate(&permission, &Permission{UserEmail: u.Email, DestinationName: d.Name, Role: role}).Error
		if err != nil {
			return err
		}
	}

	return
}

func (r *Destination) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = generate.RandString(12)
	}

	return
}

func (d *Destination) AfterCreate(tx *gorm.DB) (err error) {
	// Apply default permissions from config
	_, err = ApplyPermissions(tx, initialConfig.Permissions)
	if err != nil {
		return err
	}

	// if user is admin, provision admin permission
	var users []User
	err = tx.Find(&users).Error
	if err != nil {
		return err
	}

	for _, u := range users {
		role := "view"
		if u.Admin {
			role = "cluster-admin"
		}

		var permission Permission
		err := tx.FirstOrCreate(&permission, &Permission{UserEmail: u.Email, DestinationName: d.Name, Role: role}).Error
		if err != nil {
			return err
		}
	}

	return
}

func (d *Destination) BeforeDelete(tx *gorm.DB) (err error) {
	if d.ID == "" {
		d.ID = generate.RandString(12)
	}

	return tx.Where(&Permission{DestinationName: d.Name}).Delete(&Permission{}).Error
}

func (g *Permission) BeforeCreate(tx *gorm.DB) (err error) {
	if g.ID == "" {
		g.ID = generate.RandString(12)
	}

	return
}

func (s *Source) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == "" {
		s.ID = generate.RandString(12)
	}
	return
}

func (s *Source) BeforeDelete(tx *gorm.DB) error {
	var users []User
	if err := tx.Model(s).Association("Users").Find(&users); err != nil {
		return err
	}

	for _, u := range users {
		s.DeleteUser(tx, &u)
	}

	return nil
}

// CreateUser will create a user and associate them with the source
// If the user already exists, they will not be created, instead an association
// will be added instead
func (s *Source) CreateUser(db *gorm.DB, user *User, email string, password string) error {
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

		if tx.Model(&user).Where(&Source{ID: s.ID}).Association("Sources").Count() == 0 {
			tx.Model(&user).Where(&Source{ID: s.ID}).Association("Sources").Append(s)
		}

		return nil
	})
}

// Delete will delete a user's association with a source
// If this is their only source, then the user will be deleted entirely
// TODO (jmorganca): wrap this in a transaction or at least find out why
// there seems to cause a bug when used in a nested transaction
func (s *Source) DeleteUser(db *gorm.DB, u *User) error {
	var user User

	if err := db.Where(&User{Email: u.Email}).First(&user).Error; err != nil {
		return err
	}

	if err := db.Model(&user).Association("Sources").Delete(s); err != nil {
		return err
	}

	if db.Model(&user).Association("Sources").Count() == 0 {
		if err := db.Delete(&user).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *Source) SyncUsers(db *gorm.DB) error {
	var emails []string
	var err error

	switch s.Kind {
	case "okta":
		emails, err = okta.Emails(s.Domain, s.ClientID, s.ApiToken)
		if err != nil {
			return err
		}
	case "infra":
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// Create users in source
		for _, email := range emails {
			if err := s.CreateUser(tx, &User{}, email, ""); err != nil {
				return err
			}
		}

		// Delete users not in source
		var toDelete []User
		if err := tx.Not("email IN ?", emails).Find(&toDelete).Error; err != nil {
			return err
		}

		for _, td := range toDelete {
			s.DeleteUser(tx, &td)
		}

		return nil
	})
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
	// this should be customizable in settings or limited by the source's
	// policy (e.g. Okta is often 1-3 hours)
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
	Sources     []Source     `yaml:"sources"`
	Permissions []Permission `yaml:"permissions"`
}

func ImportSources(db *gorm.DB, sources []Source) error {
	var idsToKeep []string
	for _, s := range sources {
		err := db.FirstOrCreate(&s, &Source{Kind: s.Kind, Domain: s.Domain}).Error
		if err != nil {
			return err
		}

		idsToKeep = append(idsToKeep, s.ID)
	}

	var toDelete []Source
	if err := db.Not(idsToKeep).Not(&Source{Kind: DefaultInfraSourceKind}).Find(&toDelete).Error; err != nil {
		return err
	}

	for _, td := range toDelete {
		if err := db.Delete(&td).Error; err != nil {
			return err
		}
	}
	return nil
}

func ApplyPermissions(db *gorm.DB, permissions []Permission) ([]string, error) {
	var ids []string
	for _, p := range permissions {
		var user User
		err := db.Where(&User{Email: p.UserEmail}).First(&user).Error
		if err != nil {
			continue
		}

		var destination Destination
		err = db.Where(&Destination{Name: p.DestinationName}).First(&destination).Error
		if err != nil {
			continue
		}

		permission := p
		p.FromConfig = true

		err = db.FirstOrCreate(&permission, &permission).Error
		if err != nil {
			return nil, err
		}

		ids = append(ids, p.ID)
	}

	return ids, nil
}

func ImportPermissions(db *gorm.DB, permissions []Permission) error {
	idsToKeep, err := ApplyPermissions(db, permissions)
	if err != nil {
		return err
	}

	return db.Not(idsToKeep).Not(&Permission{FromConfig: false}).Delete(Permission{}).Error
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
		if _, ok := raw["sources"]; ok {
			if err = ImportSources(tx, config.Sources); err != nil {
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
		err := db.Find(&config.Sources).Error
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
	db.AutoMigrate(&Source{})
	db.AutoMigrate(&Destination{})
	db.AutoMigrate(&Permission{})
	db.AutoMigrate(&Settings{})
	db.AutoMigrate(&Token{})
	db.AutoMigrate(&APIKey{})

	// Add default source
	infraSource := Source{Kind: DefaultInfraSourceKind}
	err = db.FirstOrCreate(&infraSource, Source{Kind: DefaultInfraSourceKind}).Error
	if err != nil {
		return nil, err
	}

	// Add default settings
	err = db.FirstOrCreate(&Settings{}, &Settings{}).Error
	if err != nil {
		return nil, err
	}

	// Add default api key
	err = db.FirstOrCreate(&APIKey{}, &APIKey{Name: "default"}).Error
	if err != nil {
		return nil, err
	}

	return db, nil
}

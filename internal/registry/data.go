package registry

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
	"gopkg.in/square/go-jose.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var IdLen = 12

type User struct {
	Id      string `gorm:"primaryKey"`
	Created int64  `gorm:"autoCreateTime"`
	Updated int64  `gorm:"autoUpdateTime"`
	Email   string `gorm:"unique"`

	Providers []Provider `gorm:"many2many:users_providers"`
	Roles     []Role     `gorm:"many2many:users_roles"`
	Groups    []Group    `gorm:"many2many:groups_users"`
}

var ProviderKindOkta = "okta"

type Provider struct {
	Id      string `gorm:"primaryKey"`
	Created int64  `gorm:"autoCreateTime"`
	Updated int64  `gorm:"autoUpdateTime"`
	Kind    string `yaml:"kind"`

	Domain       string
	ClientID     string
	ClientSecret string

	// used for okta sync
	APIToken string

	Users []User `gorm:"many2many:users_providers"`
}

type Group struct {
	Id         string `gorm:"primaryKey"`
	Created    int64  `gorm:"autoCreateTime"`
	Updated    int64  `gorm:"autoUpdateTime"`
	Name       string
	ProviderId string
	Provider   Provider `gorm:"foreignKey:ProviderId;references:Id"`

	Roles []Role `gorm:"many2many:groups_roles"`
	Users []User `gorm:"many2many:groups_users"`
}

var DestinationKindKubernetes = "kubernetes"

type Destination struct {
	Id      string `gorm:"primaryKey"`
	Created int64  `gorm:"autoCreateTime"`
	Updated int64  `gorm:"autoUpdateTime"`
	Name    string `gorm:"unique"`
	Kind    string

	KubernetesCa       string
	KubernetesEndpoint string

	Labels []Label `gorm:"many2many:destinations_labels"`
}

type Label struct {
	ID    string `gorm:"primaryKey"`
	Value string
}

type Role struct {
	Id            string `gorm:"primaryKey"`
	Created       int64  `gorm:"autoCreateTime"`
	Updated       int64  `gorm:"autoUpdateTime"`
	Name          string
	Kind          string
	Namespace     string
	DestinationId string
	Destination   Destination `gorm:"foreignKey:DestinationId;references:Id"`
	Groups        []Group     `gorm:"many2many:groups_roles"`
	Users         []User      `gorm:"many2many:users_roles"`
}

var (
	RoleKindKubernetesRole        = "role"
	RoleKindKubernetesClusterRole = "cluster-role"
)

type Settings struct {
	Id         string `gorm:"primaryKey"`
	Created    int64  `gorm:"autoCreateTime"`
	Updated    int64  `gorm:"autoUpdateTime"`
	PrivateJWK []byte
	PublicJWK  []byte
}

var (
	TokenSecretLen = 24
	TokenLen       = IdLen + TokenSecretLen
)

type Token struct {
	Id      string `gorm:"primaryKey"`
	Created int64  `gorm:"autoCreateTime"`
	Updated int64  `gorm:"autoUpdateTime"`
	Expires int64
	Secret  []byte

	UserId string
	User   User `gorm:"foreignKey:UserId;references:Id;"`
}

var APIKeyLen = 24

type APIKey struct {
	Id          string `gorm:"primaryKey"`
	Created     int64  `gorm:"autoCreateTime"`
	Updated     int64  `gorm:"autoUpdateTime"`
	Name        string `gorm:"unique"`
	Key         string
	Permissions string // space separated list of permissions/scopes that a token can perform
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.Id == "" {
		u.Id = generate.MathRandString(IdLen)
	}

	return nil
}

func (u *User) AfterCreate(tx *gorm.DB) error {
	if _, err := ApplyUserMappings(tx, initialConfig.Users); err != nil {
		return fmt.Errorf("after create user mapping: %w", err)
	}

	return nil
}

// TODO (jmorganca): use foreign constraints instead?
func (u *User) BeforeDelete(tx *gorm.DB) error {
	if err := tx.Model(u).Association("Providers").Clear(); err != nil {
		return fmt.Errorf("user associations before delete: %w", err)
	}

	if err := tx.Where(&Token{UserId: u.Id}).Delete(&Token{}).Error; err != nil {
		return fmt.Errorf("delete user tokens before user: %w", err)
	}

	logging.S.Debugf("deleting user: %s", u.Id)

	var roles []Role
	if err := tx.Model(u).Association("Roles").Find(&roles); err != nil {
		return fmt.Errorf("find user roles before delete: %w", err)
	}

	if err := tx.Model(u).Association("Roles").Clear(); err != nil {
		return fmt.Errorf("clear user roles before delete: %w", err)
	}

	return cleanUnassociatedRoles(tx, roles)
}

func (d *Destination) BeforeCreate(tx *gorm.DB) (err error) {
	if d.Id == "" {
		d.Id = generate.MathRandString(IdLen)
	}

	return nil
}

func (d *Destination) AfterCreate(tx *gorm.DB) error {
	if err := tx.Model(&d).Association("Labels").Replace(d.Labels); err != nil {
		return err
	}

	if _, err := ApplyGroupMappings(tx, initialConfig.Groups); err != nil {
		return fmt.Errorf("group apply after destination create: %w", err)
	}

	if _, err := ApplyUserMappings(tx, initialConfig.Users); err != nil {
		return fmt.Errorf("user apply after destination create: %w", err)
	}

	return nil
}

// TODO (jmorganca): use foreign constraints instead?
func (d *Destination) BeforeDelete(tx *gorm.DB) (err error) {
	if err := tx.Model(d).Association("Labels").Clear(); err != nil {
		return fmt.Errorf("before delete destination mapping: %w", err)
	}

	return tx.Where(&Role{DestinationId: d.Id}).Delete(&Role{}).Error
}

func (l *Label) BeforeCreate(tx *gorm.DB) (err error) {
	if l.ID == "" {
		// TODO (#570): use some other form of randomly generated identifier as the ID
		l.ID = generate.MathRandString(IdLen)
	}

	return nil
}

func (r *Role) BeforeCreate(tx *gorm.DB) (err error) {
	if r.Id == "" {
		r.Id = generate.MathRandString(IdLen)
	}

	return nil
}

func (g *Group) BeforeCreate(tx *gorm.DB) (err error) {
	if g.Id == "" {
		g.Id = generate.MathRandString(IdLen)
	}

	return nil
}

func (g *Group) AfterCreate(tx *gorm.DB) error {
	if _, err := ApplyGroupMappings(tx, initialConfig.Groups); err != nil {
		return fmt.Errorf("after create group mapping: %w", err)
	}

	return nil
}

func (g *Group) BeforeDelete(tx *gorm.DB) error {
	if err := tx.Model(g).Association("Users").Clear(); err != nil {
		return fmt.Errorf("clear group users before delete: %w", err)
	}

	logging.S.Debugf("deleting group: %s", g.Id)

	var roles []Role
	if err := tx.Model(g).Association("Roles").Find(&roles); err != nil {
		return fmt.Errorf("find group roles before delete: %w", err)
	}

	if err := tx.Model(g).Association("Roles").Clear(); err != nil {
		return fmt.Errorf("clear group roles before delete: %w", err)
	}

	return cleanUnassociatedRoles(tx, roles)
}

// cleanUnassociatedRoles deletes roles with no users/groups
func cleanUnassociatedRoles(tx *gorm.DB, roles []Role) error {
	for _, r := range roles {
		usrCount := tx.Model(r).Association("Users").Count()
		grpCount := tx.Model(r).Association("Groups").Count()

		if usrCount == 0 && grpCount == 0 {
			logging.S.Debugf("deleting role with no associations: %s at %s", r.Id, r.DestinationId)

			if err := tx.Delete(r).Error; err != nil {
				return fmt.Errorf("delete unassociated role: %w", err)
			}
		}
	}

	return nil
}

func (p *Provider) BeforeCreate(tx *gorm.DB) (err error) {
	if p.Id == "" {
		p.Id = generate.MathRandString(IdLen)
	}

	return nil
}

func (p *Provider) BeforeDelete(tx *gorm.DB) error {
	var users []User
	if err := tx.Model(p).Association("Users").Find(&users); err != nil {
		return err
	}

	for _, u := range users {
		if err := p.DeleteUser(tx, u); err != nil {
			return err
		}
	}

	return nil
}

// CreateUser will create a user and associate them with the provider
// If the user already exists, they will not be created, instead an association
// will be added instead
func (p *Provider) CreateUser(db *gorm.DB, user *User, email string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(&User{Email: email}).FirstOrCreate(&user).Error; err != nil {
			return err
		}

		if tx.Model(&user).Where(&Provider{Id: p.Id}).Association("Providers").Count() == 0 {
			if err := tx.Model(&user).Where(&Provider{Id: p.Id}).Association("Providers").Append(p); err != nil {
				return err
			}
		}

		return nil
	})
}

// Delete will delete a user's association with a provider
// If this is their only provider, then the user will be deleted entirely
// TODO (jmorganca): wrap this in a transaction or at least find out why
// there seems to cause a bug when used in a nested transaction
func (p *Provider) DeleteUser(db *gorm.DB, u User) error {
	if err := db.Model(&u).Association("Providers").Delete(p); err != nil {
		return err
	}

	if db.Model(&u).Association("Providers").Count() == 0 {
		if err := db.Delete(&u).Error; err != nil {
			return err
		}
	}

	return nil
}

// Validate checks that an Okta provider is valid
func (p *Provider) Validate(r *Registry) error {
	switch p.Kind {
	case ProviderKindOkta:
		apiToken, err := r.GetSecret(p.APIToken)
		if err != nil {
			// this logs the expected secret object location, not the actual secret
			return fmt.Errorf("could not retrieve okta API token from kubernetes secret %v: %w", p.APIToken, err)
		}

		if _, err := r.GetSecret(p.ClientSecret); err != nil {
			return fmt.Errorf("could not retrieve okta client secret %v: %w", p.ClientSecret, err)
		}

		return r.okta.ValidateOktaConnection(p.Domain, p.ClientID, apiToken)
	default:
		return nil
	}
}

func (p *Provider) SyncUsers(r *Registry) error {
	var emails []string

	switch p.Kind {
	case ProviderKindOkta:
		apiToken, err := r.GetSecret(p.APIToken)
		if err != nil {
			return fmt.Errorf("sync okta users api token: %w", err)
		}

		emails, err = r.okta.Emails(p.Domain, p.ClientID, apiToken)
		if err != nil {
			return fmt.Errorf("sync okta emails: %w", err)
		}
	default:
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create users in provider
		for _, email := range emails {
			if err := p.CreateUser(tx, &User{}, email); err != nil {
				return fmt.Errorf("create user from okta: %w", err)
			}
		}

		// Remove users from provider that no longer exist in the identity provider
		var toDelete []User
		if err := tx.Not("email IN ?", emails).Find(&toDelete).Error; err != nil {
			return fmt.Errorf("sync okta delete emails: %w", err)
		}

		for _, td := range toDelete {
			if err := p.DeleteUser(tx, td); err != nil {
				return fmt.Errorf("sync okta delete users: %w", err)
			}
		}

		return nil
	})
}

func (p *Provider) SyncGroups(r *Registry) error {
	var groupEmails map[string][]string

	switch p.Kind {
	case ProviderKindOkta:
		apiToken, err := r.GetSecret(p.APIToken)
		if err != nil {
			return fmt.Errorf("sync okta groups api secret: %w", err)
		}

		groupEmails, err = r.okta.Groups(p.Domain, p.ClientID, apiToken)
		if err != nil {
			return fmt.Errorf("sync okta groups: %w", err)
		}
	default:
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		var activeIDs []string
		for groupName, emails := range groupEmails {
			var group Group
			if err := tx.FirstOrCreate(&group, Group{Name: groupName, ProviderId: p.Id}).Error; err != nil {
				logging.S.Debug("could not find or create okta group: " + groupName)
				return fmt.Errorf("sync create okta group: %w", err)
			}
			var users []User
			if err := tx.Where("email IN ?", emails).Find(&users).Error; err != nil {
				return fmt.Errorf("sync okta group emails: %w", err)
			}

			if err := tx.Model(&group).Association("Users").Replace(users); err != nil {
				return fmt.Errorf("sync okta replace with %d group users: %w", len(users), err)
			}
			activeIDs = append(activeIDs, group.Id)
		}

		// Delete provider groups not in response
		var toDelete []Group
		if err := tx.Where(&Group{ProviderId: p.Id}).Not(activeIDs).Find(&toDelete).Error; err != nil {
			return fmt.Errorf("sync okta find inactive not in %d active: %w", len(activeIDs), err)
		}

		for i := range toDelete {
			if err := tx.Delete(&toDelete[i]).Error; err != nil {
				return fmt.Errorf("sync okta delete user: %w", err)
			}
		}

		return nil
	})
}

func (s *Settings) BeforeCreate(tx *gorm.DB) (err error) {
	if s.Id == "" {
		s.Id = generate.MathRandString(IdLen)
	}

	return nil
}

func (s *Settings) BeforeSave(tx *gorm.DB) error {
	if len(s.PublicJWK) == 0 || len(s.PrivateJWK) == 0 {
		pubkey, key, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return err
		}

		priv := jose.JSONWebKey{Key: key, KeyID: "", Algorithm: string(jose.ED25519), Use: "sig"}

		thumb, err := priv.Thumbprint(crypto.SHA256)
		if err != nil {
			return err
		}

		kid := base64.URLEncoding.EncodeToString(thumb)
		priv.KeyID = kid
		pub := jose.JSONWebKey{Key: pubkey, KeyID: kid, Algorithm: string(jose.ED25519), Use: "sig"}

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
	if t.Id == "" {
		t.Id = generate.MathRandString(IdLen)
	}

	if t.Expires == 0 {
		return fmt.Errorf("token expiry not set")
	}

	return nil
}

func (t *Token) CheckExpired() (err error) {
	if time.Now().After(time.Unix(t.Expires, 0)) {
		return errors.New("could not verify expired token")
	}

	return nil
}

func (t *Token) CheckSecret(secret string) (err error) {
	h := sha256.New()
	h.Write([]byte(secret))

	if subtle.ConstantTimeCompare(t.Secret, h.Sum(nil)) != 1 {
		return errors.New("could not verify token secret")
	}

	return nil
}

func NewToken(db *gorm.DB, userId string, sessionDuration time.Duration, token *Token) (secret string, err error) {
	secret, err = generate.RandString(TokenSecretLen)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write([]byte(secret))

	token.UserId = userId
	token.Secret = h.Sum(nil)
	token.Expires = time.Now().Add(sessionDuration).Unix()

	err = db.Create(token).Error
	if err != nil {
		return "", err
	}

	return secret, nil
}

func ValidateAndGetToken(db *gorm.DB, in string) (*Token, error) {
	if len(in) != TokenLen {
		return nil, errors.New("invalid token length")
	}

	id := in[0:IdLen]
	secret := in[IdLen:TokenLen]

	var token Token
	if err := db.Preload("User").First(&token, &Token{Id: id}).Error; err != nil {
		return nil, errors.New("could not retrieve token – it may not exist")
	}

	if err := token.CheckExpired(); err != nil {
		return nil, errors.New("token expired")
	}

	if err := token.CheckSecret(secret); err != nil {
		return nil, errors.New("invalid token secret")
	}

	return &token, nil
}

func (a *APIKey) BeforeCreate(tx *gorm.DB) (err error) {
	if a.Id == "" {
		a.Id = generate.MathRandString(IdLen)
	}

	if a.Key == "" {
		a.Key, err = generate.RandString(APIKeyLen)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewDB(dbpath string) (*gorm.DB, error) {
	if !strings.HasPrefix(dbpath, "file::memory") {
		if err := os.MkdirAll(path.Dir(dbpath), os.ModePerm); err != nil {
			return nil, err
		}
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

	if err := db.AutoMigrate(&User{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Provider{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Role{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Group{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Destination{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Label{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Settings{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Token{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&APIKey{}); err != nil {
		return nil, err
	}

	// Add default settings
	err = db.FirstOrCreate(&Settings{}).Error
	if err != nil {
		return nil, err
	}

	return db, nil
}

package server

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"testing"

	"go.uber.org/zap/zaptest"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/secrets"
	"github.com/infrahq/infra/uid"
)

func setupLogging(t *testing.T) {
	origL := logging.L
	logging.L = zaptest.NewLogger(t)
	logging.S = logging.L.Sugar()
	t.Cleanup(func() {
		logging.L = origL
		logging.S = logging.L.Sugar()
	})
}

func TestGetPostgresConnectionURL(t *testing.T) {
	setupLogging(t)

	r := &Server{options: Options{}, secrets: make(map[string]secrets.SecretStorage)}

	f := secrets.NewPlainSecretProviderFromConfig(secrets.GenericConfig{})
	r.secrets["plaintext"] = f

	url, err := r.getPostgresConnectionString()
	assert.NilError(t, err)

	assert.Assert(t, is.Len(url, 0))

	r.options.DBHost = "localhost"

	url, err = r.getPostgresConnectionString()
	assert.NilError(t, err)

	assert.Equal(t, "host=localhost", url)

	r.options.DBPort = 5432

	url, err = r.getPostgresConnectionString()
	assert.NilError(t, err)
	assert.Equal(t, "host=localhost port=5432", url)

	r.options.DBUser = "user"

	url, err = r.getPostgresConnectionString()
	assert.NilError(t, err)

	assert.Equal(t, "host=localhost user=user port=5432", url)

	r.options.DBPassword = "plaintext:secret"

	url, err = r.getPostgresConnectionString()
	assert.NilError(t, err)

	assert.Equal(t, "host=localhost user=user password=secret port=5432", url)

	r.options.DBName = "postgres"

	url, err = r.getPostgresConnectionString()
	assert.NilError(t, err)

	assert.Equal(t, "host=localhost user=user password=secret port=5432 dbname=postgres", url)
}

func TestSetupRequired(t *testing.T) {
	db := setupDB(t)

	s := Server{db: db}

	// cases where setup is enabled
	cases := map[string]Options{
		"EnableSetup": {
			EnableSetup: true,
		},
		"NoImportProviders": {
			EnableSetup: true,
			Config: Config{
				Providers: []Provider{},
			},
		},
		"NoImportGrants": {
			EnableSetup: true,
			Config: Config{
				Grants: []Grant{},
			},
		},
	}

	for name, options := range cases {
		t.Run(name, func(t *testing.T) {
			s.options = options
			assert.Assert(t, s.setupRequired())
		})
	}

	// cases where setup is disabled through configs
	cases = map[string]Options{
		"DisableSetup": {
			EnableSetup: false,
		},
		"AdminAccessKey": {
			EnableSetup:    true,
			AdminAccessKey: "admin-access-key",
		},
		"AccessKey": {
			EnableSetup: true,
			AccessKey:   "access-key",
		},
		"ImportProviders": {
			EnableSetup: true,
			Config: Config{
				Providers: []Provider{
					{
						Name: "provider",
					},
				},
			},
		},
		"ImportGrants": {
			EnableSetup: true,
			Config: Config{
				Grants: []Grant{
					{
						Role: "admin",
					},
				},
			},
		},
	}

	for name, options := range cases {
		t.Run(name, func(t *testing.T) {
			s.options = options
			assert.Assert(t, !s.setupRequired())
		})
	}

	// reset options
	s.options = Options{
		EnableSetup: true,
	}

	err := db.Create(&models.Identity{Name: "non-admin"}).Error
	assert.NilError(t, err)

	assert.Assert(t, s.setupRequired())

	err = db.Create(&models.Identity{Name: "admin"}).Error
	assert.NilError(t, err)

	assert.Assert(t, !s.setupRequired())
}

func TestLoadConfigEmpty(t *testing.T) {
	db := setupDB(t)

	err := data.CreateGrant(db, &models.Grant{Subject: uid.PolymorphicID("i:1234"), Privilege: "view", Resource: "kubernetes.config-test"})
	assert.NilError(t, err)

	err = loadConfig(db, Config{})
	assert.NilError(t, err)

	var providers, grants int64

	err = db.Model(&models.Provider{}).Count(&providers).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), providers) // internal infra provider only

	err = db.Model(&models.Grant{}).Count(&grants).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), grants)
}

func TestLoadConfigInvalid(t *testing.T) {
	cases := map[string]Config{
		"MissingProviderName": {
			Providers: []Provider{
				{
					URL:          "demo.okta.com",
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				},
			},
		},
		"MissingProviderURL": {
			Providers: []Provider{
				{
					Name:         "okta",
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				},
			},
		},
		"MissingProviderClientID": {
			Providers: []Provider{
				{
					Name:         "okta",
					URL:          "demo.okta.com",
					ClientSecret: "client-secret",
				},
			},
		},
		"MissingProviderClientSecret": {
			Providers: []Provider{
				{
					Name:     "okta",
					URL:      "demo.okta.com",
					ClientID: "client-id",
				},
			},
		},
		"MissingGrantIdentity": {
			Grants: []Grant{
				{
					Role:     "admin",
					Resource: "kubernetes.test-cluster",
				},
			},
		},
		"MissingGrantRole": {
			Grants: []Grant{
				{
					Machine:  "T-1000",
					Resource: "kubernetes.test-cluster",
				},
			},
		},
		"MissingGrantResource": {
			Grants: []Grant{
				{
					Machine: "T-1000",
					Role:    "admin",
				},
			},
		},
		"UserGrantWithMultipleProviders": {
			Providers: []Provider{
				{
					Name:         "okta",
					URL:          "demo.okta.com",
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				},
				{
					Name:         "atko",
					URL:          "demo.atko.com",
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				},
			},
			Grants: []Grant{
				{
					User:     "test@example.com",
					Role:     "admin",
					Resource: "kubernetes.test-cluster",
				},
			},
		},
		"GroupGrantWithMultipleProviders": {
			Providers: []Provider{
				{
					Name:         "okta",
					URL:          "demo.okta.com",
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				},
				{
					Name:         "atko",
					URL:          "demo.atko.com",
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				},
			},
			Grants: []Grant{
				{
					Group:    "Everyone",
					Role:     "admin",
					Resource: "kubernetes.test-cluster",
				},
			},
		},
	}

	for name, config := range cases {
		t.Run(name, func(t *testing.T) {
			db := setupDB(t)

			err := loadConfig(db, config)
			// TODO: add expectedErr for each case
			assert.ErrorContains(t, err, "") // could be any error
		})
	}
}

func TestLoadConfigWithProviders(t *testing.T) {
	db := setupDB(t)

	config := Config{
		Providers: []Provider{
			{
				Name:         "okta",
				URL:          "demo.okta.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
		},
	}

	err := loadConfig(db, config)
	assert.NilError(t, err)

	var provider models.Provider
	err = db.Where("name = ?", "okta").First(&provider).Error
	assert.NilError(t, err)
	assert.Equal(t, "okta", provider.Name)
	assert.Equal(t, "demo.okta.com", provider.URL)
	assert.Equal(t, "client-id", provider.ClientID)
	assert.Equal(t, models.EncryptedAtRest("client-secret"), provider.ClientSecret)
}

func TestLoadConfigWithUserGrantsImplicitProvider(t *testing.T) {
	db := setupDB(t)

	config := Config{
		Grants: []Grant{
			{
				User:     "test@example.com",
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
			},
		},
	}

	err := loadConfig(db, config)
	assert.NilError(t, err)

	var provider models.Provider
	err = db.Where("name = ?", models.InternalInfraProviderName).First(&provider).Error
	assert.NilError(t, err)

	var user models.Identity
	err = db.Where("name = ?", "test@example.com").First(&user).Error
	assert.NilError(t, err)
	assert.Equal(t, provider.ID, user.ProviderID)

	var grant models.Grant
	err = db.Where("subject = ?", uid.NewIdentityPolymorphicID(user.ID)).First(&grant).Error
	assert.NilError(t, err)
	assert.Equal(t, "admin", grant.Privilege)
	assert.Equal(t, "kubernetes.test-cluster", grant.Resource)
}

func TestLoadConfigWithUserGrantsExplicitProvider(t *testing.T) {
	db := setupDB(t)

	config := Config{
		Providers: []Provider{
			{
				Name:         "okta",
				URL:          "demo.okta.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
			{
				Name:         "atko",
				URL:          "demo.atko.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
		},
		Grants: []Grant{
			{
				User:     "test@example.com",
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
				Provider: "atko",
			},
		},
	}

	err := loadConfig(db, config)
	assert.NilError(t, err)

	var provider models.Provider
	err = db.Where("name = ?", "atko").First(&provider).Error
	assert.NilError(t, err)

	var user models.Identity
	err = db.Where("name = ?", "test@example.com").First(&user).Error
	assert.NilError(t, err)
	assert.Equal(t, provider.ID, user.ProviderID)

	var grant models.Grant
	err = db.Where("subject = ?", uid.NewIdentityPolymorphicID(user.ID)).First(&grant).Error
	assert.NilError(t, err)
	assert.Equal(t, "admin", grant.Privilege)
	assert.Equal(t, "kubernetes.test-cluster", grant.Resource)
}

func TestLoadConfigWithGroupGrantsImplicitProvider(t *testing.T) {
	db := setupDB(t)

	config := Config{
		Grants: []Grant{
			{
				Group:    "Everyone",
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
			},
		},
	}

	err := loadConfig(db, config)
	assert.NilError(t, err)

	var provider models.Provider
	err = db.Where("name = ?", models.InternalInfraProviderName).First(&provider).Error
	assert.NilError(t, err)

	var group models.Group
	err = db.Where("name = ?", "Everyone").First(&group).Error
	assert.NilError(t, err)
	assert.Equal(t, provider.ID, group.ProviderID)

	var grant models.Grant
	err = db.Where("subject = ?", uid.NewGroupPolymorphicID(group.ID)).First(&grant).Error
	assert.NilError(t, err)
	assert.Equal(t, "admin", grant.Privilege)
	assert.Equal(t, "kubernetes.test-cluster", grant.Resource)
}

func TestLoadConfigWithGroupGrantsExplicitProvider(t *testing.T) {
	db := setupDB(t)

	config := Config{
		Providers: []Provider{
			{
				Name:         "okta",
				URL:          "demo.okta.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
			{
				Name:         "atko",
				URL:          "demo.atko.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
		},
		Grants: []Grant{
			{
				Group:    "Everyone",
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
				Provider: "atko",
			},
		},
	}

	err := loadConfig(db, config)
	assert.NilError(t, err)

	var provider models.Provider
	err = db.Where("name = ?", "atko").First(&provider).Error
	assert.NilError(t, err)

	var group models.Group
	err = db.Where("name = ?", "Everyone").First(&group).Error
	assert.NilError(t, err)
	assert.Equal(t, provider.ID, group.ProviderID)

	var grant models.Grant
	err = db.Where("subject = ?", uid.NewGroupPolymorphicID(group.ID)).First(&grant).Error
	assert.NilError(t, err)
	assert.Equal(t, "admin", grant.Privilege)
	assert.Equal(t, "kubernetes.test-cluster", grant.Resource)
}

func TestLoadConfigWithMachineGrants(t *testing.T) {
	db := setupDB(t)

	config := Config{
		Grants: []Grant{
			{
				Machine:  "T-1000",
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
			},
		},
	}

	err := loadConfig(db, config)
	assert.NilError(t, err)

	var machine models.Identity
	err = db.Where("name = ?", "T-1000").First(&machine).Error
	assert.NilError(t, err)

	var grant models.Grant
	err = db.Where("subject = ?", uid.NewIdentityPolymorphicID(machine.ID)).First(&grant).Error
	assert.NilError(t, err)
	assert.Equal(t, "admin", grant.Privilege)
	assert.Equal(t, "kubernetes.test-cluster", grant.Resource)
}

func TestLoadConfigPruneConfig(t *testing.T) {
	db := setupDB(t)

	config := Config{
		Providers: []Provider{
			{
				Name:         "okta",
				URL:          "demo.okta.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
		},
		Grants: []Grant{
			{
				User:     "test@example.com",
				Provider: "okta",
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
			},
			{
				Group:    "Everyone",
				Provider: "okta",
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
			},
			{
				Machine:  "T-1000",
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
			},
		},
	}

	err := loadConfig(db, config)
	assert.NilError(t, err)

	var providers, grants, users, groups, machines int64

	err = db.Model(&models.Provider{}).Count(&providers).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(2), providers) // okta and infra providers

	err = db.Model(&models.Grant{}).Count(&grants).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(3), grants)

	err = db.Model(&models.Identity{}).Where(models.Identity{Kind: models.UserKind}).Count(&users).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), users)

	err = db.Model(&models.Group{}).Count(&groups).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), groups)

	err = db.Model(&models.Identity{}).Where(models.Identity{Kind: models.MachineKind}).Count(&machines).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), machines)

	// previous config is cleared on new config application
	newConfig := Config{
		Providers: []Provider{
			{
				Name:         "okta",
				URL:          "new-demo.okta.com",
				ClientID:     "new-client-id",
				ClientSecret: "new-client-secret",
			},
		},
	}

	err = loadConfig(db, newConfig)
	assert.NilError(t, err)

	err = db.Model(&models.Provider{}).Count(&providers).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(2), providers) // infra and new okta

	err = db.Model(&models.Grant{}).Count(&grants).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(0), grants)
}

func TestLoadConfigPruneGrants(t *testing.T) {
	db := setupDB(t)

	config := Config{
		Providers: []Provider{
			{
				Name:         "okta",
				URL:          "demo.okta.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
		},
		Grants: []Grant{
			{
				User:     "test@example.com",
				Provider: "okta",
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
			},
			{
				Group:    "Everyone",
				Provider: "okta",
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
			},
			{
				Machine:  "T-1000",
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
			},
		},
	}

	err := loadConfig(db, config)
	assert.NilError(t, err)

	var providers, grants, users, groups, machines int64

	err = db.Model(&models.Provider{}).Count(&providers).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(2), providers) // infra and okta

	err = db.Model(&models.Grant{}).Count(&grants).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(3), grants)

	err = db.Model(&models.Identity{}).Where(models.Identity{Kind: models.UserKind}).Count(&users).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), users)

	err = db.Model(&models.Group{}).Count(&groups).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), groups)

	err = db.Model(&models.Identity{}).Where(models.Identity{Kind: models.MachineKind}).Count(&machines).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), machines)

	providersOnly := Config{
		Providers: []Provider{
			{
				Name:         "okta",
				URL:          "demo.okta.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
		},
	}

	err = loadConfig(db, providersOnly)
	assert.NilError(t, err)

	err = db.Model(&models.Provider{}).Count(&providers).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(2), providers) // infra and okta

	err = db.Model(&models.Grant{}).Count(&grants).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(0), grants)

	err = db.Model(&models.Identity{}).Where(models.Identity{Kind: models.UserKind}).Count(&users).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), users)

	err = db.Model(&models.Group{}).Count(&groups).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), groups)

	err = db.Model(&models.Identity{}).Where(models.Identity{Kind: models.MachineKind}).Count(&machines).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), machines)
}

func TestLoadConfigUpdate(t *testing.T) {
	db := setupDB(t)

	config := Config{
		Providers: []Provider{
			{
				Name:         "okta",
				URL:          "demo.okta.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
		},
		Grants: []Grant{
			{
				User:     "test@example.com",
				Provider: "okta",
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
			},
			{
				Group:    "Everyone",
				Provider: "okta",
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
			},
			{
				Machine:  "T-1000",
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
			},
		},
	}

	err := loadConfig(db, config)
	assert.NilError(t, err)

	var providers, users, groups, machines int64

	err = db.Model(&models.Provider{}).Count(&providers).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(2), providers) // infra and okta

	grants := make([]models.Grant, 0)
	err = db.Find(&grants).Error
	assert.NilError(t, err)
	assert.Assert(t, is.Len(grants, 3))
	assert.Equal(t, "admin", grants[0].Privilege)
	assert.Equal(t, "admin", grants[1].Privilege)
	assert.Equal(t, "admin", grants[2].Privilege)

	err = db.Model(&models.Identity{}).Where(models.Identity{Kind: models.UserKind}).Count(&users).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), users)

	err = db.Model(&models.Group{}).Count(&groups).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), groups)

	err = db.Model(&models.Identity{}).Where(models.Identity{Kind: models.MachineKind}).Count(&machines).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), machines)

	updatedConfig := Config{
		Providers: []Provider{
			{
				Name:         "atko",
				URL:          "demo.atko.com",
				ClientID:     "client-id-2",
				ClientSecret: "client-secret-2",
			},
		},
		Grants: []Grant{
			{
				User:     "test@example.com",
				Provider: "atko",
				Role:     "view",
				Resource: "kubernetes.test-cluster",
			},
			{
				Group:    "Everyone",
				Provider: "atko",
				Role:     "view",
				Resource: "kubernetes.test-cluster",
			},
			{
				Machine:  "T-1000",
				Role:     "view",
				Resource: "kubernetes.test-cluster",
			},
		},
	}

	err = loadConfig(db, updatedConfig)
	assert.NilError(t, err)

	err = db.Model(&models.Provider{}).Count(&providers).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(2), providers) // infra and atko

	var provider models.Provider
	err = db.Where("name = ?", "atko").First(&provider).Error
	assert.NilError(t, err)
	assert.Equal(t, "atko", provider.Name)
	assert.Equal(t, "demo.atko.com", provider.URL)
	assert.Equal(t, "client-id-2", provider.ClientID)
	assert.Equal(t, models.EncryptedAtRest("client-secret-2"), provider.ClientSecret)

	grants = make([]models.Grant, 0)
	err = db.Find(&grants).Error
	assert.NilError(t, err)
	assert.Assert(t, is.Len(grants, 3))
	assert.Equal(t, "view", grants[0].Privilege)
	assert.Equal(t, "view", grants[1].Privilege)
	assert.Equal(t, "view", grants[2].Privilege)

	err = db.Model(&models.Identity{}).Where(models.Identity{Kind: models.UserKind}).Count(&users).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), users)

	var user models.Identity
	err = db.Where("name = ?", "test@example.com").First(&user).Error
	assert.NilError(t, err)
	assert.Equal(t, provider.ID, user.ProviderID)

	err = db.Model(&models.Group{}).Count(&groups).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), groups)

	var group models.Group
	err = db.Where("name = ?", "Everyone").First(&group).Error
	assert.NilError(t, err)
	assert.Equal(t, provider.ID, group.ProviderID)

	err = db.Model(&models.Identity{}).Where(models.Identity{Kind: models.MachineKind}).Count(&machines).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), machines)

	var machine models.Identity
	err = db.Where("name = ?", "T-1000").First(&machine).Error
	assert.NilError(t, err)
}

func TestImportAccessKeys(t *testing.T) {
	db := setupDB(t)

	s := Server{db: db}

	s.options = Options{
		AdminAccessKey: "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb",
		AccessKey:      "tuogTmCFSk.FzoWHhNonnRztyRChPUiMqDx",
	}

	err := s.importSecrets()
	assert.NilError(t, err)

	err = s.importAccessKeys()
	assert.NilError(t, err)
}

func TestImportAccessKeysUpdate(t *testing.T) {
	db := setupDB(t)

	s := Server{db: db}

	s.options = Options{
		AdminAccessKey: "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb",
		AccessKey:      "tuogTmCFSk.FzoWHhNonnRztyRChPUiMqDx",
	}

	err := s.importSecrets()
	assert.NilError(t, err)

	err = s.importAccessKeys()
	assert.NilError(t, err)

	s.options = Options{
		AdminAccessKey: "EKoHADINYX.NfhgLRqggYgdQiQXoxrNwgOe",
	}

	err = s.importAccessKeys()
	assert.NilError(t, err)

	accessKey, err := data.GetAccessKey(s.db, data.ByName("default admin access key"))
	assert.NilError(t, err)
	assert.Equal(t, accessKey.KeyID, "EKoHADINYX")
}

func TestServer_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	dir := t.TempDir()
	opts := Options{
		DBEncryptionKeyProvider: "native",
		DBEncryptionKey:         filepath.Join(dir, "sqlite3.db.key"),
		TLSCache:                filepath.Join(dir, "tlscache"),
		DBFile:                  filepath.Join(dir, "sqlite3.db"),
	}
	srv, err := New(opts)
	assert.NilError(t, err)

	go func() {
		if err := srv.Run(ctx); err != nil {
			t.Errorf("server errored: %v", err)
		}
	}()

	t.Run("metrics server started", func(t *testing.T) {
		resp, err := http.Get("http://" + srv.Addrs.Metrics.String() + "/metrics")
		assert.NilError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := ioutil.ReadAll(resp.Body)
		assert.NilError(t, err)
		assert.Assert(t, is.Contains(string(body), "# HELP"))
		assert.Assert(t, is.Contains(string(body), "# TYPE"))
	})

	t.Run("http server started", func(t *testing.T) {
		resp, err := http.Get("http://" + srv.Addrs.HTTP.String() + "/healthz")
		assert.NilError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("https server started", func(t *testing.T) {
		tr := &http.Transport{}
		tr.TLSClientConfig = &tls.Config{
			// TODO: use the actual certs when that is possible
			//nolint:gosec
			InsecureSkipVerify: true,
		}
		client := &http.Client{Transport: tr}

		url := "https://" + srv.Addrs.HTTPS.String() + "/healthz"
		req, err := http.NewRequest(http.MethodGet, url, nil)
		assert.NilError(t, err)

		resp, err := client.Do(req)
		assert.NilError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

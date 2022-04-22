package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/infrahq/secrets"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap/zaptest"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func setupServer(t *testing.T, ops ...func(*testing.T, *Options)) *Server {
	t.Helper()
	options := Options{}
	for _, op := range ops {
		op(t, &options)
	}
	s := newServer(options)
	s.db = setupDB(t)

	// TODO: share more of this with Server.New
	err := loadDefaultSecretConfig(s.secrets)
	assert.NilError(t, err)

	err = s.setupInternalInfraIdentityProvider()
	assert.NilError(t, err)

	err = s.importAccessKeys()
	assert.NilError(t, err)

	return s
}

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

	r := newServer(Options{})

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

func TestSignupEnabled(t *testing.T) {
	db := setupDB(t)

	s := Server{db: db}

	// cases where setup is enabled
	cases := map[string]Options{
		"EnableSignup": {
			EnableSignup: true,
		},
		"NoImportProviders": {
			EnableSignup: true,
			Config: Config{
				Providers: []Provider{},
			},
		},
		"NoImportGrants": {
			EnableSignup: true,
			Config: Config{
				Grants: []Grant{},
			},
		},
	}

	for name, options := range cases {
		t.Run(name, func(t *testing.T) {
			s.options = options
			assert.Assert(t, s.signupEnabled())
		})
	}

	// cases where setup is disabled through configs
	cases = map[string]Options{
		"DisableSetup": {
			EnableSignup: false,
		},
		"AdminAccessKey": {
			EnableSignup:   true,
			AdminAccessKey: "admin-access-key",
		},
		"AccessKey": {
			EnableSignup: true,
			AccessKey:    "access-key",
		},
		"ImportProviders": {
			EnableSignup: true,
			Config: Config{
				Providers: []Provider{
					{
						Name: "provider",
					},
				},
			},
		},
		"ImportGrants": {
			EnableSignup: true,
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
			assert.Assert(t, !s.signupEnabled())
		})
	}

	// reset options
	s.options = Options{
		EnableSignup: true,
	}

	err := db.Create(&models.Identity{Name: "non-admin"}).Error
	assert.NilError(t, err)

	assert.Assert(t, s.signupEnabled())

	id := uid.New()
	err = db.Create(&models.Identity{Model: models.Model{ID: id}, Name: "admin"}).Error
	assert.NilError(t, err)

	err = db.Create(&models.AccessKey{Name: "admin", IssuedFor: id, ExpiresAt: time.Now()}).Error
	assert.NilError(t, err)

	assert.Assert(t, !s.signupEnabled())
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
			},
		},
	}

	err := loadConfig(db, config)
	assert.NilError(t, err)

	var user models.Identity
	err = db.Where("name = ?", "test@example.com").First(&user).Error
	assert.NilError(t, err)

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

	var group models.Group
	err = db.Where("name = ?", "Everyone").First(&group).Error
	assert.NilError(t, err)

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
			},
		},
	}

	err := loadConfig(db, config)
	assert.NilError(t, err)

	var group models.Group
	err = db.Where("name = ?", "Everyone").First(&group).Error
	assert.NilError(t, err)

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
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
			},
			{
				Group:    "Everyone",
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

	// removing provider also removes ProviderUser
	var count int64
	err = db.Model(&models.ProviderUser{}).Count(&count).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(0), count)
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
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
			},
			{
				Group:    "Everyone",
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
				Role:     "admin",
				Resource: "kubernetes.test-cluster",
			},
			{
				Group:    "Everyone",
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
				Role:     "view",
				Resource: "kubernetes.test-cluster",
			},
			{
				Group:    "Everyone",
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

	err = db.Model(&models.Group{}).Count(&groups).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), groups)

	var group models.Group
	err = db.Where("name = ?", "Everyone").First(&group).Error
	assert.NilError(t, err)

	err = db.Model(&models.Identity{}).Where(models.Identity{Kind: models.MachineKind}).Count(&machines).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), machines)

	var machine models.Identity
	err = db.Where("name = ?", "T-1000").First(&machine).Error
	assert.NilError(t, err)
}

func TestImportAccessKeys(t *testing.T) {
	s := setupServer(t)

	s.options = Options{
		AdminAccessKey: "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb",
		AccessKey:      "tuogTmCFSk.FzoWHhNonnRztyRChPUiMqDx",
	}

	err := s.importAccessKeys()
	assert.NilError(t, err)
}

func TestImportAccessKeysUpdate(t *testing.T) {
	s := setupServer(t)

	s.options = Options{
		AdminAccessKey: "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb",
		AccessKey:      "tuogTmCFSk.FzoWHhNonnRztyRChPUiMqDx",
	}

	err := s.importAccessKeys()
	assert.NilError(t, err)

	s.options = Options{
		AdminAccessKey: "EKoHADINYX.NfhgLRqggYgdQiQXoxrNwgOe",
	}

	err = s.importAccessKeys()
	assert.NilError(t, err)

	accessKey, err := data.GetAccessKey(s.db, data.ByName("default-admin-access-key"))
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
		// the infra http request metric
		assert.Assert(t, is.Contains(string(body), "# HELP http_request_duration_seconds"))
		// standard go metrics
		assert.Assert(t, is.Contains(string(body), "# HELP go_threads"))
		// standard process metrics
		if runtime.GOOS == "linux" {
			assert.Assert(t, is.Contains(string(body), "# HELP process_open_fds"))
		}
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

func TestServer_Run_UIProxy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	message := `message through the proxy`
	uiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(message))
	}))
	t.Cleanup(uiSrv.Close)

	dir := t.TempDir()
	opts := Options{
		DBEncryptionKeyProvider: "native",
		DBEncryptionKey:         filepath.Join(dir, "sqlite3.db.key"),
		TLSCache:                filepath.Join(dir, "tlscache"),
		DBFile:                  filepath.Join(dir, "sqlite3.db"),
		UI:                      UIOptions{Enabled: true},
		EnableSignup:            true,
	}
	assert.NilError(t, opts.UI.ProxyURL.Set(uiSrv.URL))

	srv, err := New(opts)
	assert.NilError(t, err)

	go func() {
		if err := srv.Run(ctx); err != nil {
			t.Errorf("server errored: %v", err)
		}
	}()

	t.Run("requests are proxied", func(t *testing.T) {
		resp, err := http.Get("http://" + srv.Addrs.HTTP.String() + "/any-path")
		assert.NilError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := ioutil.ReadAll(resp.Body)
		assert.NilError(t, err)
		assert.Equal(t, message, string(body))
	})

	t.Run("api routes are available", func(t *testing.T) {
		resp, err := http.Get("http://" + srv.Addrs.HTTP.String() + "/v1/signup")
		assert.NilError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body api.SignupEnabledResponse
		err = json.NewDecoder(resp.Body).Decode(&body)
		assert.NilError(t, err)

		assert.Assert(t, body.Enabled)
	})
}

func TestServer_GenerateRoutes_NoRoute(t *testing.T) {
	type testCase struct {
		name     string
		path     string
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	s := &Server{options: Options{UI: UIOptions{Enabled: true}}}
	router := s.GenerateRoutes(prometheus.NewRegistry())

	run := func(t *testing.T, tc testCase) {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, resp.Code, http.StatusNotFound)
		if tc.expected != nil {
			tc.expected(t, resp)
		}
	}

	testCases := []testCase{
		{
			name: "/v1 path prefix",
			path: "/v1/not/found",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				contentType := resp.Header().Get("Content-Type")
				expected := "application/json; charset=utf-8"
				assert.Equal(t, contentType, expected)
			},
		},
		{
			name: "ui path",
			path: "/not/found",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				// response should have an html body
				title := "<title>404: This page could not be found</title>"
				assert.Assert(t, is.Contains(resp.Body.String(), title))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestServer_GenerateRoutes_UI(t *testing.T) {
	type testCase struct {
		name         string
		path         string
		expectedCode int
		expected     func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	s := &Server{options: Options{UI: UIOptions{Enabled: true}}}
	router := s.GenerateRoutes(prometheus.NewRegistry())

	run := func(t *testing.T, tc testCase) {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Check(t, is.Equal(resp.Code, tc.expectedCode))
		if tc.expected != nil {
			tc.expected(t, resp)
		}
	}

	testCases := []testCase{
		{
			name:         "default index",
			path:         "/",
			expectedCode: http.StatusOK,
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				actual := resp.Header().Get("Content-Type")
				assert.Equal(t, actual, "text/html; charset=utf-8")
			},
		},
		{
			name:         "index page redirects",
			path:         "/index.html",
			expectedCode: http.StatusMovedPermanently,
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				actual := resp.Header().Get("Location")
				assert.Equal(t, actual, "./")
			},
		},
		{
			name:         "page with a path",
			path:         "/providers/add/admins.html",
			expectedCode: http.StatusOK,
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				actual := resp.Header().Get("Content-Type")
				assert.Equal(t, actual, "text/html; charset=utf-8")
			},
		},
		{
			name:         "image",
			path:         "/icon.svg",
			expectedCode: http.StatusOK,
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				actual := resp.Header().Get("Content-Type")
				assert.Equal(t, actual, "image/svg+xml")
			},
		},
		{
			name:         "page without .html suffix",
			path:         "/providers/add/admins",
			expectedCode: http.StatusOK,
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				actual := resp.Header().Get("Content-Type")
				assert.Equal(t, actual, "text/html; charset=utf-8")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

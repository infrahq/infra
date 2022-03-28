package server

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/secrets"
	"github.com/infrahq/infra/uid"
)

func setupLogging(t *testing.T) {
	logging.L = zaptest.NewLogger(t)
	logging.S = logging.L.Sugar()
}

func TestGetPostgresConnectionURL(t *testing.T) {
	setupLogging(t)

	r := &Server{options: Options{}, secrets: make(map[string]secrets.SecretStorage)}

	f := secrets.NewPlainSecretProviderFromConfig(secrets.GenericConfig{})
	r.secrets["plaintext"] = f

	url, err := r.getPostgresConnectionString()
	require.NoError(t, err)

	require.Empty(t, url)

	r.options.DBHost = "localhost"

	url, err = r.getPostgresConnectionString()
	require.NoError(t, err)

	require.Equal(t, "host=localhost", url)

	r.options.DBPort = 5432

	url, err = r.getPostgresConnectionString()
	require.NoError(t, err)
	require.Equal(t, "host=localhost port=5432", url)

	r.options.DBUser = "user"

	url, err = r.getPostgresConnectionString()
	require.NoError(t, err)

	require.Equal(t, "host=localhost user=user port=5432", url)

	r.options.DBPassword = "plaintext:secret"

	url, err = r.getPostgresConnectionString()
	require.NoError(t, err)

	require.Equal(t, "host=localhost user=user password=secret port=5432", url)

	r.options.DBName = "postgres"

	url, err = r.getPostgresConnectionString()
	require.NoError(t, err)

	require.Equal(t, "host=localhost user=user password=secret port=5432 dbname=postgres", url)
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
			require.True(t, s.setupRequired())
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
			require.False(t, s.setupRequired())
		})
	}

	// reset options
	s.options = Options{
		EnableSetup: true,
	}

	err := db.Create(&models.Machine{Name: "non-admin"}).Error
	require.NoError(t, err)

	require.True(t, s.setupRequired())

	err = db.Create(&models.Machine{Name: "admin"}).Error
	require.NoError(t, err)

	require.False(t, s.setupRequired())
}

func TestLoadConfigEmpty(t *testing.T) {
	db := setupDB(t)

	err := loadConfig(db, Config{})
	require.NoError(t, err)

	var providers, grants int64

	err = db.Model(&models.Provider{}).Count(&providers).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), providers) // just the infra provider

	err = db.Model(&models.Grant{}).Count(&grants).Error
	require.NoError(t, err)
	require.Equal(t, int64(0), grants)
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
			require.Error(t, err)
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
	require.NoError(t, err)

	var provider models.Provider
	err = db.Where("name = ?", "okta").First(&provider).Error
	require.NoError(t, err)
	require.Equal(t, "okta", provider.Name)
	require.Equal(t, "demo.okta.com", provider.URL)
	require.Equal(t, "client-id", provider.ClientID)
	require.Equal(t, models.EncryptedAtRest("client-secret"), provider.ClientSecret)
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
	require.NoError(t, err)

	var provider models.Provider
	err = db.Where("name = ?", models.InternalInfraProviderName).First(&provider).Error
	require.NoError(t, err)

	var user models.User
	err = db.Where("email = ?", "test@example.com").First(&user).Error
	require.NoError(t, err)
	require.Equal(t, provider.ID, user.ProviderID)

	var grant models.Grant
	err = db.Where("subject = ?", uid.NewUserPolymorphicID(user.ID)).First(&grant).Error
	require.NoError(t, err)
	require.Equal(t, "admin", grant.Privilege)
	require.Equal(t, "kubernetes.test-cluster", grant.Resource)
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
	require.NoError(t, err)

	var provider models.Provider
	err = db.Where("name = ?", "atko").First(&provider).Error
	require.NoError(t, err)

	var user models.User
	err = db.Where("email = ?", "test@example.com").First(&user).Error
	require.NoError(t, err)
	require.Equal(t, provider.ID, user.ProviderID)

	var grant models.Grant
	err = db.Where("subject = ?", uid.NewUserPolymorphicID(user.ID)).First(&grant).Error
	require.NoError(t, err)
	require.Equal(t, "admin", grant.Privilege)
	require.Equal(t, "kubernetes.test-cluster", grant.Resource)
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
	require.NoError(t, err)

	var provider models.Provider
	err = db.Where("name = ?", models.InternalInfraProviderName).First(&provider).Error
	require.NoError(t, err)

	var group models.Group
	err = db.Where("name = ?", "Everyone").First(&group).Error
	require.NoError(t, err)
	require.Equal(t, provider.ID, group.ProviderID)

	var grant models.Grant
	err = db.Where("subject = ?", uid.NewGroupPolymorphicID(group.ID)).First(&grant).Error
	require.NoError(t, err)
	require.Equal(t, "admin", grant.Privilege)
	require.Equal(t, "kubernetes.test-cluster", grant.Resource)
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
	require.NoError(t, err)

	var provider models.Provider
	err = db.Where("name = ?", "atko").First(&provider).Error
	require.NoError(t, err)

	var group models.Group
	err = db.Where("name = ?", "Everyone").First(&group).Error
	require.NoError(t, err)
	require.Equal(t, provider.ID, group.ProviderID)

	var grant models.Grant
	err = db.Where("subject = ?", uid.NewGroupPolymorphicID(group.ID)).First(&grant).Error
	require.NoError(t, err)
	require.Equal(t, "admin", grant.Privilege)
	require.Equal(t, "kubernetes.test-cluster", grant.Resource)
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
	require.NoError(t, err)

	var machine models.Machine
	err = db.Where("name = ?", "T-1000").First(&machine).Error
	require.NoError(t, err)

	var grant models.Grant
	err = db.Where("subject = ?", uid.NewMachinePolymorphicID(machine.ID)).First(&grant).Error
	require.NoError(t, err)
	require.Equal(t, "admin", grant.Privilege)
	require.Equal(t, "kubernetes.test-cluster", grant.Resource)
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
	require.NoError(t, err)

	var providers, grants, users, groups, machines int64

	err = db.Model(&models.Provider{}).Count(&providers).Error
	require.NoError(t, err)
	require.Equal(t, int64(2), providers) // okta and infra providers

	err = db.Model(&models.Grant{}).Count(&grants).Error
	require.NoError(t, err)
	require.Equal(t, int64(3), grants)

	err = db.Model(&models.User{}).Count(&users).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), users)

	err = db.Model(&models.Group{}).Count(&groups).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), groups)

	err = db.Model(&models.Machine{}).Count(&machines).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), machines)

	err = loadConfig(db, Config{})
	require.NoError(t, err)

	err = db.Model(&models.Provider{}).Count(&providers).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), providers) // infra provider only

	err = db.Model(&models.Grant{}).Count(&grants).Error
	require.NoError(t, err)
	require.Equal(t, int64(0), grants)

	// removing provider also removes users/groups
	err = db.Model(&models.User{}).Count(&users).Error
	require.NoError(t, err)
	require.Equal(t, int64(0), users)

	err = db.Model(&models.Group{}).Count(&groups).Error
	require.NoError(t, err)
	require.Equal(t, int64(0), groups)

	// but not machines
	err = db.Model(&models.Machine{}).Count(&machines).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), machines)
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
	require.NoError(t, err)

	var providers, grants, users, groups, machines int64

	err = db.Model(&models.Provider{}).Count(&providers).Error
	require.NoError(t, err)
	require.Equal(t, int64(2), providers) // infra and okta

	err = db.Model(&models.Grant{}).Count(&grants).Error
	require.NoError(t, err)
	require.Equal(t, int64(3), grants)

	err = db.Model(&models.User{}).Count(&users).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), users)

	err = db.Model(&models.Group{}).Count(&groups).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), groups)

	err = db.Model(&models.Machine{}).Count(&machines).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), machines)

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
	require.NoError(t, err)

	err = db.Model(&models.Provider{}).Count(&providers).Error
	require.NoError(t, err)
	require.Equal(t, int64(2), providers) // infra and okta

	err = db.Model(&models.Grant{}).Count(&grants).Error
	require.NoError(t, err)
	require.Equal(t, int64(0), grants)

	err = db.Model(&models.User{}).Count(&users).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), users)

	err = db.Model(&models.Group{}).Count(&groups).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), groups)

	err = db.Model(&models.Machine{}).Count(&machines).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), machines)
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
	require.NoError(t, err)

	var providers, users, groups, machines int64

	err = db.Model(&models.Provider{}).Count(&providers).Error
	require.NoError(t, err)
	require.Equal(t, int64(2), providers) // infra and okta

	grants := make([]models.Grant, 0)
	err = db.Find(&grants).Error
	require.NoError(t, err)
	require.Len(t, grants, 3)
	require.Equal(t, "admin", grants[0].Privilege)
	require.Equal(t, "admin", grants[1].Privilege)
	require.Equal(t, "admin", grants[2].Privilege)

	err = db.Model(&models.User{}).Count(&users).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), users)

	err = db.Model(&models.Group{}).Count(&groups).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), groups)

	err = db.Model(&models.Machine{}).Count(&machines).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), machines)

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
	require.NoError(t, err)

	err = db.Model(&models.Provider{}).Count(&providers).Error
	require.NoError(t, err)
	require.Equal(t, int64(2), providers) // infra and atko

	var provider models.Provider
	err = db.Where("name = ?", "atko").First(&provider).Error
	require.NoError(t, err)
	require.Equal(t, "atko", provider.Name)
	require.Equal(t, "demo.atko.com", provider.URL)
	require.Equal(t, "client-id-2", provider.ClientID)
	require.Equal(t, models.EncryptedAtRest("client-secret-2"), provider.ClientSecret)

	grants = make([]models.Grant, 0)
	err = db.Find(&grants).Error
	require.NoError(t, err)
	require.Len(t, grants, 3)
	require.Equal(t, "view", grants[0].Privilege)
	require.Equal(t, "view", grants[1].Privilege)
	require.Equal(t, "view", grants[2].Privilege)

	err = db.Model(&models.User{}).Count(&users).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), users)

	var user models.User
	err = db.Where("email = ?", "test@example.com").First(&user).Error
	require.NoError(t, err)
	require.Equal(t, provider.ID, user.ProviderID)

	err = db.Model(&models.Group{}).Count(&groups).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), groups)

	var group models.Group
	err = db.Where("name = ?", "Everyone").First(&group).Error
	require.NoError(t, err)
	require.Equal(t, provider.ID, group.ProviderID)

	err = db.Model(&models.Machine{}).Count(&machines).Error
	require.NoError(t, err)
	require.Equal(t, int64(1), machines)

	var machine models.Machine
	err = db.Where("name = ?", "T-1000").First(&machine).Error
	require.NoError(t, err)
}

func TestImportAccessKeys(t *testing.T) {
	db := setupDB(t)

	s := Server{db: db}

	s.options = Options{
		AdminAccessKey: "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb",
		AccessKey:      "tuogTmCFSk.FzoWHhNonnRztyRChPUiMqDx",
	}

	err := s.importSecrets()
	require.NoError(t, err)

	err = s.importAccessKeys()
	require.NoError(t, err)
}

func TestImportAccessKeysUpdate(t *testing.T) {
	db := setupDB(t)

	s := Server{db: db}

	s.options = Options{
		AdminAccessKey: "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb",
		AccessKey:      "tuogTmCFSk.FzoWHhNonnRztyRChPUiMqDx",
	}

	err := s.importSecrets()
	require.NoError(t, err)

	err = s.importAccessKeys()
	require.NoError(t, err)

	s.options = Options{
		AdminAccessKey: "EKoHADINYX.NfhgLRqggYgdQiQXoxrNwgOe",
	}

	err = s.importAccessKeys()
	require.NoError(t, err)

	accessKey, err := data.GetAccessKey(s.db, data.ByName("default admin access key"))
	require.NoError(t, err)
	require.Equal(t, accessKey.KeyID, "EKoHADINYX")
}

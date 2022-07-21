package server

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/assert/opt"
	"k8s.io/utils/strings/slices"

	"github.com/infrahq/infra/internal/cmd/cliopts"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestKeyProvider_PrepareForDecode_IntegrationWithDecode_FullConfig(t *testing.T) {
	content := `
keys:
  - kind: vault
    config:
      token: the-token
      namespace: the-namespace
      secretMount: secret-mount
      address: https://vault:12345
  - kind: awskms
    config:
      encryptionAlgorithm: aes_512
      endpoint: /endpoint
      region: the-region
      accessKeyID: the-key-id
      secretAccessKey: the-secret
  - kind: native
    config:
      secretProvider: the-storage
`
	raw := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(content), &raw)
	assert.NilError(t, err)

	actual := Options{}
	err = decodeConfig(&actual, raw)
	assert.NilError(t, err)

	expected := Options{
		Keys: []KeyProvider{
			{
				Kind: "vault",
				Config: VaultConfig{
					TransitMount: "",
					SecretMount:  "secret-mount",
					Token:        "the-token",
					Namespace:    "the-namespace",
					Address:      "https://vault:12345",
				},
			},
			{
				Kind: "awskms",
				Config: AWSKMSConfig{
					AWSConfig: AWSConfig{
						Endpoint:        "/endpoint",
						Region:          "the-region",
						AccessKeyID:     "the-key-id",
						SecretAccessKey: "the-secret",
					},
					EncryptionAlgorithm: "aes_512",
				},
			},
			{
				Kind: "native",
				Config: nativeKeyProviderConfig{
					SecretProvider: "the-storage",
				},
			},
		},
	}
	assert.DeepEqual(t, expected, actual)
}

func TestSecretProvider_PrepareForDecode_IntegrationWithDecode_FullConfig(t *testing.T) {
	content := `
secrets:

  - name: the-vault
    kind: vault
    config:
      transitMount: /some-mount
      token: the-token
      namespace: the-namespace
      secretMount: secret-mount
      address: https://vault:12345

  - name: the-aws
    kind: awsssm
    config:
      keyID: the-key-id
      endpoint: the-endpoint
      region: the-region
      accessKeyID: the-access-key
      secretAccessKey: the-secret-key

  - name: aws-2
    kind: awssecretsmanager
    config:
      endpoint: the-endpoint-2
      region: the-region-2
      accessKeyID: the-access-key-2
      secretAccessKey: the-secret-key-2

  - name: the-kubes
    kind: kubernetes
    config:
      namespace: the-namespace

  - name: the-env
    kind: env
    config:
      base64: true
      base64UrlEncoded: true
      base64Raw: true

  - name: the-file
    kind: file
    config:
      path: /the-path
      base64: true

  - name: the-plaintext
    kind: plaintext
    config:
      base64Raw: true
`
	raw := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(content), &raw)
	assert.NilError(t, err)

	actual := Options{}
	err = decodeConfig(&actual, raw)
	assert.NilError(t, err)

	expected := Options{
		Secrets: []SecretProvider{
			{
				Kind: "vault",
				Name: "the-vault",
				Config: VaultConfig{
					TransitMount: "/some-mount",
					SecretMount:  "secret-mount",
					Token:        "the-token",
					Namespace:    "the-namespace",
					Address:      "https://vault:12345",
				},
			},
			{
				Kind: "awsssm",
				Name: "the-aws",
				Config: AWSSSMConfig{
					KeyID: "the-key-id",
					AWSConfig: AWSConfig{
						Endpoint:        "the-endpoint",
						Region:          "the-region",
						AccessKeyID:     "the-access-key",
						SecretAccessKey: "the-secret-key",
					},
				},
			},
			{
				Kind: "awssecretsmanager",
				Name: "aws-2",
				Config: AWSSecretsManagerConfig{
					AWSConfig: AWSConfig{
						Endpoint:        "the-endpoint-2",
						Region:          "the-region-2",
						AccessKeyID:     "the-access-key-2",
						SecretAccessKey: "the-secret-key-2",
					},
				},
			},
			{
				Kind: "kubernetes",
				Name: "the-kubes",
				Config: KubernetesConfig{
					Namespace: "the-namespace",
				},
			},
			{
				Kind: "env",
				Name: "the-env",
				Config: GenericConfig{
					Base64:           true,
					Base64URLEncoded: true,
					Base64Raw:        true,
				},
			},
			{
				Kind: "file",
				Name: "the-file",
				Config: FileConfig{
					Path: "/the-path",
					GenericConfig: GenericConfig{
						Base64: true,
					},
				},
			},
			{
				Kind: "plaintext",
				Name: "the-plaintext",
				Config: GenericConfig{
					Base64Raw: true,
				},
			},
		},
	}
	assert.DeepEqual(t, expected, actual)
}

func decodeConfig(target interface{}, source interface{}) error {
	cfg := cliopts.DecodeConfig(target)
	decoder, err := mapstructure.NewDecoder(&cfg)
	if err != nil {
		return err
	}
	return decoder.Decode(source)
}

type rawConfig map[interface{}]interface{}

func TestSecretProvider_PrepareForDecode_IntegrationWithDecode(t *testing.T) {
	type testCase struct {
		name        string
		source      rawConfig
		expectedErr string
		expected    SecretProvider
	}

	run := func(t *testing.T, tc testCase) {
		actual := SecretProvider{}
		err := decodeConfig(&actual, tc.source)
		if tc.expectedErr != "" {
			assert.ErrorContains(t, err, tc.expectedErr)
			return
		}

		assert.NilError(t, err)
		assert.DeepEqual(t, actual, tc.expected)
	}

	testCases := []testCase{
		{
			name: "minimal config",
			expected: SecretProvider{
				Kind:   "plaintext",
				Config: GenericConfig{},
			},
		},
		{
			name:   "missing kind",
			source: rawConfig{"name": "custom"},
			expected: SecretProvider{
				Kind:   "plaintext",
				Name:   "custom",
				Config: GenericConfig{},
			},
		},
		{
			name:        "wrong type for name",
			source:      rawConfig{"name": map[string]int{}},
			expectedErr: `'name' expected type 'string'`,
		},
		{
			name:        "wrong type for kind",
			source:      rawConfig{"kind": map[string]int{}},
			expectedErr: `'kind' expected type 'string'`,
		},
		{
			name:        "wrong type for config",
			source:      rawConfig{"config": true},
			expectedErr: `expected a map, got 'bool'`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestLoadConfigEmpty(t *testing.T) {
	s := setupServer(t)

	err := s.loadConfig(Config{})
	assert.NilError(t, err)

	var providers, grants int64

	err = s.db.Model(&models.Provider{}).Count(&providers).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), providers) // internal infra provider only

	err = s.db.Model(&models.Grant{}).Count(&grants).Error
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
		"MissingProviderRequiredScopes": {
			Providers: []Provider{
				{
					Name:     "okta",
					URL:      "demo.okta.com",
					ClientID: "client-id",
					AuthURL:  "example.com",
					Scopes:   []string{"offline_access"},
				},
			},
		},
		"MissingGrantIdentity": {
			Grants: []Grant{
				{
					Role:     "admin",
					Resource: "test-cluster",
				},
			},
		},
	}

	for name, config := range cases {
		t.Run(name, func(t *testing.T) {
			s := setupServer(t)

			err := s.loadConfig(config)
			// TODO: add expectedErr for each case
			assert.ErrorContains(t, err, "") // could be any error
		})
	}
}

var cmpEncryptedAtRestEqual = cmp.Comparer(func(x, y models.EncryptedAtRest) bool {
	return string(x) == string(y)
})

func TestLoadConfigWithProviders(t *testing.T) {
	s := setupServer(t)

	config := Config{
		Providers: []Provider{
			{
				Name:         "okta",
				URL:          "demo.okta.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				AuthURL:      "dev.okta.com/oauth2/default/v1/token",
				Scopes:       []string{"openid", "email"},
			},
			{
				Name:         "azure",
				URL:          "demo.azure.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				Kind:         models.ProviderKindAzure.String(),
				AuthURL:      "demo.azure.com/oauth2/v2.0/authorize",
				Scopes:       []string{"openid", "email"},
			},
			{
				Name:             "google",
				URL:              "accounts.google.com",
				ClientID:         "client-id",
				ClientSecret:     "client-secret",
				Kind:             models.ProviderKindGoogle.String(),
				AuthURL:          "https://accounts.google.com/o/oauth2/v2/auth",
				Scopes:           []string{"openid", "email"},
				PrivateKey:       "-----BEGIN PRIVATE KEY-----\naaa=\n-----END PRIVATE KEY-----\n",
				ClientEmail:      "example@tenant.iam.gserviceaccount.com",
				DomainAdminEmail: "admin@example.com",
			},
		},
	}

	err := s.loadConfig(config)
	assert.NilError(t, err)

	var okta models.Provider
	err = s.db.Where("name = ?", "okta").First(&okta).Error
	assert.NilError(t, err)

	expected := models.Provider{
		Model:        okta.Model,     // not relevant
		CreatedBy:    okta.CreatedBy, // not relevant
		Name:         "okta",
		URL:          "demo.okta.com",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		Kind:         models.ProviderKindOIDC, // the kind gets the default value
		AuthURL:      "dev.okta.com/oauth2/default/v1/token",
		Scopes:       []string{"openid", "email"},
	}

	cmpProvider := cmp.Options{
		cmp.FilterPath(
			opt.PathField(models.Provider{}, "ClientSecret"),
			cmpEncryptedAtRestEqual),
		cmp.FilterPath(
			opt.PathField(models.Provider{}, "Scopes"),
			cmp.Comparer(slices.Equal)),
	}

	assert.DeepEqual(t, okta, expected, cmpProvider)

	var azure models.Provider
	err = s.db.Where("name = ?", "azure").First(&azure).Error
	assert.NilError(t, err)

	expected = models.Provider{
		Model:        azure.Model,     // not relevant
		CreatedBy:    azure.CreatedBy, // not relevant
		Name:         "azure",
		URL:          "demo.azure.com",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		Kind:         models.ProviderKindAzure, // when specified, the kind is set
		AuthURL:      "demo.azure.com/oauth2/v2.0/authorize",
		Scopes:       []string{"openid", "email"},
	}

	assert.DeepEqual(t, azure, expected, cmpProvider)

	var google models.Provider
	err = s.db.Where("name = ?", "google").First(&google).Error
	assert.NilError(t, err)

	expected = models.Provider{
		Model:            google.Model,     // not relevant
		CreatedBy:        google.CreatedBy, // not relevant
		Name:             "google",
		URL:              "accounts.google.com",
		ClientID:         "client-id",
		ClientSecret:     "client-secret",
		Kind:             models.ProviderKindGoogle,
		AuthURL:          "https://accounts.google.com/o/oauth2/v2/auth",
		Scopes:           []string{"openid", "email"},
		PrivateKey:       "-----BEGIN PRIVATE KEY-----\naaa=\n-----END PRIVATE KEY-----\n",
		ClientEmail:      "example@tenant.iam.gserviceaccount.com",
		DomainAdminEmail: "admin@example.com",
	}

	assert.DeepEqual(t, google, expected, cmpProvider)
}

func TestLoadConfigWithUsers(t *testing.T) {
	s := setupServer(t)

	config := Config{
		Users: []User{
			{
				Name: "bob",
			},
			{
				Name:     "alice",
				Password: "password",
			},
			{
				Name:      "sue",
				AccessKey: "aaaaaaaaaa.bbbbbbbbbbbbbbbbbbbbbbbb",
			},
			{
				Name:      "jim",
				Password:  "password",
				AccessKey: "bbbbbbbbbb.bbbbbbbbbbbbbbbbbbbbbbbb",
			},
		},
	}

	err := s.loadConfig(config)
	assert.NilError(t, err)

	user, _, _, err := getTestUserDetails(s.db, "bob")
	assert.NilError(t, err)
	assert.Equal(t, "bob", user.Name)

	user, creds, _, err := getTestUserDetails(s.db, "alice")
	assert.NilError(t, err)
	assert.Equal(t, "alice", user.Name)
	err = bcrypt.CompareHashAndPassword(creds.PasswordHash, []byte("password"))
	assert.NilError(t, err)

	user, _, key, err := getTestUserDetails(s.db, "sue")
	assert.NilError(t, err)
	assert.Equal(t, "sue", user.Name)
	assert.Equal(t, key.KeyID, "aaaaaaaaaa")
	chksm := sha256.Sum256([]byte("bbbbbbbbbbbbbbbbbbbbbbbb"))
	assert.Equal(t, bytes.Compare(key.SecretChecksum, chksm[:]), 0) // 0 means the byte slices are equal

	user, creds, key, err = getTestUserDetails(s.db, "jim")
	assert.NilError(t, err)
	assert.Equal(t, "jim", user.Name)
	err = bcrypt.CompareHashAndPassword(creds.PasswordHash, []byte("password"))
	assert.NilError(t, err)
	assert.Equal(t, key.KeyID, "bbbbbbbbbb")
	chksm = sha256.Sum256([]byte("bbbbbbbbbbbbbbbbbbbbbbbb"))
	assert.Equal(t, bytes.Compare(key.SecretChecksum, chksm[:]), 0) // 0 means the byte slices are equal
}

func TestLoadConfigWithUserGrants_OptionalRole(t *testing.T) {
	s := setupServer(t)

	config := Config{
		Grants: []Grant{
			{
				User:     "test@example.com",
				Resource: "test-cluster",
			},
		},
	}

	err := s.loadConfig(config)
	assert.NilError(t, err)

	var user models.Identity
	err = s.db.Where("name = ?", "test@example.com").First(&user).Error
	assert.NilError(t, err)

	var grant models.Grant
	err = s.db.Where("subject = ?", uid.NewIdentityPolymorphicID(user.ID)).First(&grant).Error
	assert.NilError(t, err)
	assert.Equal(t, "connect", grant.Privilege)
	assert.Equal(t, "test-cluster", grant.Resource)
}

func TestLoadConfigWithUserGrants(t *testing.T) {
	s := setupServer(t)

	config := Config{
		Grants: []Grant{
			{
				User:     "test@example.com",
				Role:     "admin",
				Resource: "test-cluster",
			},
		},
	}

	err := s.loadConfig(config)
	assert.NilError(t, err)

	var provider models.Provider
	err = s.db.Where("name = ?", models.InternalInfraProviderName).First(&provider).Error
	assert.NilError(t, err)

	var user models.Identity
	err = s.db.Where("name = ?", "test@example.com").First(&user).Error
	assert.NilError(t, err)

	var grant models.Grant
	err = s.db.Where("subject = ?", uid.NewIdentityPolymorphicID(user.ID)).First(&grant).Error
	assert.NilError(t, err)
	assert.Equal(t, "admin", grant.Privilege)
	assert.Equal(t, "test-cluster", grant.Resource)
}

func TestLoadConfigWithGroupGrants(t *testing.T) {
	s := setupServer(t)

	config := Config{
		Grants: []Grant{
			{
				Group:    "Everyone",
				Role:     "admin",
				Resource: "test-cluster",
			},
		},
	}

	err := s.loadConfig(config)
	assert.NilError(t, err)

	var group models.Group
	err = s.db.Where("name = ?", "Everyone").First(&group).Error
	assert.NilError(t, err)

	var grant models.Grant
	err = s.db.Where("subject = ?", uid.NewGroupPolymorphicID(group.ID)).First(&grant).Error
	assert.NilError(t, err)
	assert.Equal(t, "admin", grant.Privilege)
	assert.Equal(t, "test-cluster", grant.Resource)
}

func TestLoadConfigPruneConfig(t *testing.T) {
	s := setupServer(t)

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
				Resource: "test-cluster",
			},
			{
				Group:    "Everyone",
				Role:     "admin",
				Resource: "test-cluster",
			},
		},
	}

	err := s.loadConfig(config)
	assert.NilError(t, err)

	var providers, grants, groups, providerUsers int64

	err = s.db.Model(&models.Provider{}).Count(&providers).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(2), providers) // okta and infra providers

	err = s.db.Model(&models.Grant{}).Count(&grants).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(3), grants) // 2 from config, 1 internal connector

	identities, err := data.ListIdentities(s.db, &models.Pagination{})
	assert.NilError(t, err)
	assert.Equal(t, 2, len(identities))

	err = s.db.Model(&models.Group{}).Count(&groups).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), groups)

	err = s.db.Model(&models.ProviderUser{}).Count(&providerUsers).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(0), providerUsers)

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

	err = s.loadConfig(newConfig)
	assert.NilError(t, err)

	err = s.db.Model(&models.Provider{}).Count(&providers).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(2), providers) // infra and new okta

	err = s.db.Model(&models.Grant{}).Count(&grants).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), grants) // connector

	identities, err = data.ListIdentities(s.db, &models.Pagination{})
	assert.NilError(t, err)
	assert.Equal(t, 1, len(identities))

	err = s.db.Model(&models.Group{}).Count(&groups).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), groups)
}

func TestLoadConfigUpdate(t *testing.T) {
	s := setupServer(t)

	config := Config{
		Providers: []Provider{
			{
				Name:         "okta",
				URL:          "demo.okta.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				AuthURL:      "demo.okta.com/auth",
				Scopes:       []string{"openid", "email"},
			},
		},
		Users: []User{
			{
				Name: "r2d2",
			},
			{
				Name:      "c3po",
				AccessKey: "TllVlekkUz.NFnxSlaPQLosgkNsyzaMttfC",
			},
			{
				Name:     "sarah@email.com",
				Password: "supersecret",
			},
		},
		Grants: []Grant{
			{
				User:     "test@example.com",
				Role:     "admin",
				Resource: "test-cluster",
			},
			{
				Group:    "Everyone",
				Role:     "admin",
				Resource: "test-cluster",
			},
		},
	}

	err := s.loadConfig(config)
	assert.NilError(t, err)

	var providers, groups, credentials, accessKeys int64

	err = s.db.Model(&models.Provider{}).Count(&providers).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(2), providers) // infra and okta

	grants := make([]models.Grant, 0)
	err = s.db.Find(&grants).Error
	assert.NilError(t, err)
	assert.Assert(t, is.Len(grants, 3)) // 2 from config, 1 internal connector

	privileges := map[string]int{
		"admin":     0,
		"view":      0,
		"connector": 0,
	}

	for _, v := range grants {
		privileges[v.Privilege]++
	}

	assert.Equal(t, privileges["admin"], 2)
	assert.Equal(t, privileges["view"], 0)
	assert.Equal(t, privileges["connector"], 1)

	identities, err := data.ListIdentities(s.db, &models.Pagination{})
	assert.NilError(t, err)
	assert.Equal(t, 5, len(identities)) // john@example.com, sarah@example.com, test@example.com, connector, r2d2, c3po

	err = s.db.Model(&models.Group{}).Count(&groups).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), groups) // Everyone

	err = s.db.Model(&models.Credential{}).Count(&credentials).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), credentials) // sarah@example.com

	err = s.db.Model(&models.AccessKey{}).Count(&accessKeys).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), accessKeys) // c3po

	updatedConfig := Config{
		Providers: []Provider{
			{
				Name:         "atko",
				URL:          "demo.atko.com",
				ClientID:     "client-id-2",
				ClientSecret: "client-secret-2",
				AuthURL:      "demo.okta.com/v1/auth",
				Scopes:       []string{"openid", "email", "groups"},
			},
		},
		Grants: []Grant{
			{
				User:     "test@example.com",
				Role:     "view",
				Resource: "test-cluster",
			},
			{
				Group:    "Everyone",
				Role:     "view",
				Resource: "test-cluster",
			},
		},
	}

	err = s.loadConfig(updatedConfig)
	assert.NilError(t, err)

	err = s.db.Model(&models.Provider{}).Count(&providers).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(2), providers) // infra and atko

	var provider models.Provider
	err = s.db.Where("name = ?", "atko").First(&provider).Error
	assert.NilError(t, err)

	expected := models.Provider{
		Model:        provider.Model,     // not relevant
		CreatedBy:    provider.CreatedBy, // not relevant
		Name:         "atko",
		URL:          "demo.atko.com",
		ClientID:     "client-id-2",
		ClientSecret: "client-secret-2",
		Kind:         models.ProviderKindOIDC, // the kind gets the default value
		AuthURL:      "demo.okta.com/v1/auth",
		Scopes:       []string{"openid", "email", "groups"},
	}

	cmpProvider := cmp.Options{
		cmp.FilterPath(
			opt.PathField(models.Provider{}, "ClientSecret"),
			cmpEncryptedAtRestEqual),
		cmp.FilterPath(
			opt.PathField(models.Provider{}, "Scopes"),
			cmp.Comparer(slices.Equal)),
	}

	assert.DeepEqual(t, provider, expected, cmpProvider)

	grants = make([]models.Grant, 0)
	err = s.db.Find(&grants).Error
	assert.NilError(t, err)
	assert.Assert(t, is.Len(grants, 3))

	privileges = map[string]int{
		"admin":     0,
		"view":      0,
		"connector": 0,
	}

	for _, v := range grants {
		privileges[v.Privilege]++
	}

	assert.Equal(t, privileges["admin"], 0)
	assert.Equal(t, privileges["view"], 2)
	assert.Equal(t, privileges["connector"], 1)

	identities, err = data.ListIdentities(s.db, &models.Pagination{})
	assert.NilError(t, err)
	assert.Equal(t, 2, len(identities))

	var user models.Identity
	err = s.db.Where("name = ?", "test@example.com").First(&user).Error
	assert.NilError(t, err)

	err = s.db.Model(&models.Group{}).Count(&groups).Error
	assert.NilError(t, err)
	assert.Equal(t, int64(1), groups)

	var group models.Group
	err = s.db.Where("name = ?", "Everyone").First(&group).Error
	assert.NilError(t, err)
}

func TestLoadAccessKey(t *testing.T) {
	s := setupServer(t)

	// access key that we will attempt to assign to multiple users
	testAccessKey := "aaaaaaaaaa.bbbbbbbbbbbbbbbbbbbbbbbb"

	// create a user and assign them an access key
	bob := &models.Identity{Name: "bob"}
	err := data.CreateIdentity(s.db, bob)
	assert.NilError(t, err)

	err = s.loadAccessKey(s.db, bob, testAccessKey)
	assert.NilError(t, err)

	t.Run("access key can be reloaded for the same identity it was issued for", func(t *testing.T) {
		err = s.loadAccessKey(s.db, bob, testAccessKey)
		assert.NilError(t, err)
	})

	t.Run("duplicate access key ID is rejected", func(t *testing.T) {
		alice := &models.Identity{Name: "alice"}
		err = data.CreateIdentity(s.db, alice)
		assert.NilError(t, err)

		err = s.loadAccessKey(s.db, alice, testAccessKey)
		assert.Error(t, err, "access key assigned to \"alice\" is already assigned to another user, a user's access key must have a unique ID")
	})
}

// getTestUserDetails gets the attributes of a user created from a config file
func getTestUserDetails(db *gorm.DB, name string) (*models.Identity, *models.Credential, *models.AccessKey, error) {
	var user models.Identity
	var credential models.Credential
	var accessKey models.AccessKey

	err := db.Where("name = ?", name).First(&user).Error
	if err != nil {
		return nil, nil, nil, err
	}

	err = db.Where("identity_id = ?", user.ID).First(&credential).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil, err
	}

	err = db.Where("issued_for = ?", user.ID).First(&accessKey).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil, err
	}

	return &user, &credential, &accessKey, nil
}

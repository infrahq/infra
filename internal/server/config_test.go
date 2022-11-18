package server

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/assert/opt"
	"k8s.io/utils/strings/slices"

	"github.com/infrahq/infra/internal"
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

	providers, err := data.CountAllProviders(s.db)
	assert.NilError(t, err)
	assert.Equal(t, int64(1), providers) // internal infra provider only

	grants, err := data.CountAllGrants(s.db)
	assert.NilError(t, err)
	assert.Equal(t, int64(1), grants) // connector grant only
}

func TestLoadConfigInvalid(t *testing.T) {
	cases := map[string]Config{
		"MissingProviderName": {
			Providers: []Provider{
				{
					URL:          "example.com",
					ClientID:     "client-id",
					ClientSecret: "client-secret",
					AuthURL:      "example.com/auth",
					Scopes:       []string{"openid", "email"},
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
					URL:          "example.com",
					ClientSecret: "client-secret",
					AuthURL:      "example.com/auth",
					Scopes:       []string{"openid", "email"},
				},
			},
		},
		"MissingProviderClientSecret": {
			Providers: []Provider{
				{
					Name:     "okta",
					URL:      "example.com",
					ClientID: "client-id",
					AuthURL:  "example.com/auth",
					Scopes:   []string{"openid", "email"},
				},
			},
		},
		"MissingProviderRequiredScopes": {
			Providers: []Provider{
				{
					Name:     "okta",
					URL:      "example.com",
					ClientID: "client-id",
					AuthURL:  "example.com/auth",
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
		DefaultOrganizationDomain: "super.example.com",
		Providers: []Provider{
			{
				Name:         "okta",
				URL:          "example.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				AuthURL:      "example.com/oauth2/default/v1/token",
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

	tx := txnForTestCase(t, s.db, s.db.DefaultOrg.ID)

	defaultOrg := s.db.DefaultOrg
	updatedOrg, err := data.GetOrganization(tx, data.GetOrganizationOptions{ByID: defaultOrg.ID})
	assert.NilError(t, err)
	assert.Equal(t, updatedOrg.Domain, "super.example.com")

	okta, err := data.GetProvider(tx, data.GetProviderOptions{ByName: "okta"})
	assert.NilError(t, err)

	expected := &models.Provider{
		Model:              okta.Model,     // not relevant
		CreatedBy:          okta.CreatedBy, // not relevant
		Name:               "okta",
		URL:                "example.com",
		ClientID:           "client-id",
		ClientSecret:       "client-secret",
		Kind:               models.ProviderKindOIDC, // the kind gets the default value
		AuthURL:            "example.com/oauth2/default/v1/token",
		Scopes:             []string{"openid", "email"},
		OrganizationMember: models.OrganizationMember{OrganizationID: defaultOrg.ID},
	}

	cmpProvider := cmp.Options{
		cmp.FilterPath(
			opt.PathField(models.Provider{}, "ClientSecret"),
			cmpEncryptedAtRestEqual),
		cmp.FilterPath(
			opt.PathField(models.Provider{}, "Scopes"),
			cmp.Comparer(slices.Equal)),
		cmpopts.EquateEmpty(),
	}
	assert.DeepEqual(t, okta, expected, cmpProvider)

	azure, err := data.GetProvider(tx, data.GetProviderOptions{ByName: "azure"})
	assert.NilError(t, err)

	expected = &models.Provider{
		Model:              azure.Model,     // not relevant
		CreatedBy:          azure.CreatedBy, // not relevant
		Name:               "azure",
		URL:                "demo.azure.com",
		ClientID:           "client-id",
		ClientSecret:       "client-secret",
		Kind:               models.ProviderKindAzure, // when specified, the kind is set
		AuthURL:            "demo.azure.com/oauth2/v2.0/authorize",
		Scopes:             []string{"openid", "email"},
		OrganizationMember: models.OrganizationMember{OrganizationID: defaultOrg.ID},
	}
	assert.DeepEqual(t, azure, expected, cmpProvider)

	google, err := data.GetProvider(tx, data.GetProviderOptions{ByName: "google"})
	assert.NilError(t, err)

	expected = &models.Provider{
		Model:              google.Model,     // not relevant
		CreatedBy:          google.CreatedBy, // not relevant
		Name:               "google",
		URL:                "accounts.google.com",
		ClientID:           "client-id",
		ClientSecret:       "client-secret",
		Kind:               models.ProviderKindGoogle,
		AuthURL:            "https://accounts.google.com/o/oauth2/v2/auth",
		Scopes:             []string{"openid", "email"},
		PrivateKey:         "-----BEGIN PRIVATE KEY-----\naaa=\n-----END PRIVATE KEY-----\n",
		ClientEmail:        "example@tenant.iam.gserviceaccount.com",
		DomainAdminEmail:   "admin@example.com",
		OrganizationMember: models.OrganizationMember{OrganizationID: defaultOrg.ID},
	}
	assert.DeepEqual(t, google, expected, cmpProvider)
}

func TestLoadConfigWithUsers(t *testing.T) {
	s := setupServer(t)

	config := Config{
		Users: []User{
			{
				Name: "bob@example.com",
			},
			{
				Name:     "alice@example.com",
				Password: "password",
			},
			{
				Name:      "sue@example.com",
				AccessKey: "aaaaaaaaaa.bbbbbbbbbbbbbbbbbbbbbbbb",
			},
			{
				Name:      "jim@example.com",
				Password:  "password",
				AccessKey: "bbbbbbbbbb.bbbbbbbbbbbbbbbbbbbbbbbb",
			},
		},
	}

	err := s.loadConfig(config)
	assert.NilError(t, err)

	user, _, _ := getTestDefaultOrgUserDetails(t, s, "bob@example.com")
	assert.Equal(t, "bob@example.com", user.Name)

	user, creds, _ := getTestDefaultOrgUserDetails(t, s, "alice@example.com")
	assert.Equal(t, "alice@example.com", user.Name)
	err = bcrypt.CompareHashAndPassword(creds.PasswordHash, []byte("password"))
	assert.NilError(t, err)

	user, _, key := getTestDefaultOrgUserDetails(t, s, "sue@example.com")
	assert.Equal(t, "sue@example.com", user.Name)
	assert.Equal(t, key.KeyID, "aaaaaaaaaa")
	chksm := sha256.Sum256([]byte("bbbbbbbbbbbbbbbbbbbbbbbb"))
	assert.Equal(t, bytes.Compare(key.SecretChecksum, chksm[:]), 0) // 0 means the byte slices are equal

	user, creds, key = getTestDefaultOrgUserDetails(t, s, "jim@example.com")
	assert.Equal(t, "jim@example.com", user.Name)
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

	user, err := data.GetIdentity(s.db, data.GetIdentityOptions{ByName: "test@example.com"})
	assert.NilError(t, err)
	assert.Assert(t, user != nil)

	grant, err := data.GetGrant(s.db, data.GetGrantOptions{
		BySubject:   uid.NewIdentityPolymorphicID(user.ID),
		ByPrivilege: "connect",
		ByResource:  "test-cluster",
	})
	assert.NilError(t, err)
	assert.Assert(t, grant != nil)
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

	tx := txnForTestCase(t, s.db, s.db.DefaultOrg.ID)

	provider, err := data.GetProvider(tx, data.GetProviderOptions{ByName: models.InternalInfraProviderName})
	assert.NilError(t, err)
	assert.Assert(t, provider != nil)

	user, err := data.GetIdentity(tx, data.GetIdentityOptions{ByName: "test@example.com"})
	assert.NilError(t, err)
	assert.Assert(t, user != nil)

	grant, err := data.GetGrant(tx, data.GetGrantOptions{
		BySubject:   uid.NewIdentityPolymorphicID(user.ID),
		ByPrivilege: "admin",
		ByResource:  "test-cluster",
	})
	assert.NilError(t, err)
	assert.Assert(t, grant != nil)
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

	tx := txnForTestCase(t, s.db, s.db.DefaultOrg.ID)

	group, err := data.GetGroup(tx, data.GetGroupOptions{ByName: "Everyone"})
	assert.NilError(t, err)
	assert.Assert(t, group != nil)

	grant, err := data.GetGrant(tx, data.GetGrantOptions{
		BySubject:   uid.NewGroupPolymorphicID(group.ID),
		ByPrivilege: "admin",
		ByResource:  "test-cluster",
	})
	assert.NilError(t, err)
	assert.Assert(t, grant != nil)
}

func TestLoadConfigPruneConfig(t *testing.T) {
	s := setupServer(t)

	config := Config{
		Providers: []Provider{
			{
				Name:         "okta",
				URL:          "example.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				AuthURL:      "example.com/auth",
				Scopes:       []string{"openid", "email"},
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

	tx := txnForTestCase(t, s.db, s.db.DefaultOrg.ID)
	defaultOrg := s.db.DefaultOrg

	var providers, grants, identities, groups, providerUsers int64

	err = tx.QueryRow("SELECT COUNT(*) FROM providers WHERE organization_id = ?;", defaultOrg.ID).Scan(&providers)
	assert.NilError(t, err)
	assert.Equal(t, int64(2), providers) // okta and infra providers

	err = tx.QueryRow("SELECT COUNT(*) FROM grants WHERE organization_id = ?;", defaultOrg.ID).Scan(&grants)
	assert.NilError(t, err)
	assert.Equal(t, int64(3), grants) // 2 from config, 1 internal connector

	err = tx.QueryRow("SELECT COUNT(*) FROM identities WHERE organization_id = ?;", defaultOrg.ID).Scan(&identities)
	assert.NilError(t, err)
	assert.Equal(t, int64(2), identities)

	err = tx.QueryRow("SELECT COUNT(*) FROM groups WHERE organization_id = ?;", defaultOrg.ID).Scan(&groups)
	assert.NilError(t, err)
	assert.Equal(t, int64(1), groups)

	err = tx.QueryRow("SELECT COUNT(*) FROM provider_users").Scan(&providerUsers)
	assert.NilError(t, err)
	assert.Equal(t, int64(1), providerUsers)

	// previous config is cleared on new config application
	newConfig := Config{
		Providers: []Provider{
			{
				Name:         "okta",
				URL:          "new.example.com",
				ClientID:     "new-client-id",
				ClientSecret: "new-client-secret",
				AuthURL:      "new.example.com/auth",
				Scopes:       []string{"openid", "email"},
			},
		},
	}

	err = s.loadConfig(newConfig)
	assert.NilError(t, err)

	err = tx.QueryRow("SELECT COUNT(*) FROM providers WHERE organization_id = ? AND deleted_at IS null;", defaultOrg.ID).Scan(&providers)
	assert.NilError(t, err)
	assert.Equal(t, int64(2), providers) // infra and new okta

	err = tx.QueryRow("SELECT COUNT(*) FROM grants WHERE organization_id = ? AND deleted_at IS null;", defaultOrg.ID).Scan(&grants)
	assert.NilError(t, err)
	assert.Equal(t, int64(1), grants) // connector

	err = tx.QueryRow("SELECT COUNT(*) FROM identities WHERE organization_id = ? AND deleted_at IS null;", defaultOrg.ID).Scan(&identities)
	assert.NilError(t, err)
	assert.Equal(t, int64(1), identities)

	err = tx.QueryRow("SELECT COUNT(*) FROM groups WHERE organization_id = ? AND deleted_at IS null;", defaultOrg.ID).Scan(&groups)
	assert.NilError(t, err)
	assert.Equal(t, int64(1), groups)
}

func TestLoadConfigUpdate(t *testing.T) {
	s := setupServer(t)

	config := Config{
		Providers: []Provider{
			{
				Name:         "okta",
				URL:          "example.okta.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				AuthURL:      "example.com/auth",
				Scopes:       []string{"openid", "email"},
			},
		},
		Users: []User{
			{
				Name: "r2d2@example.com",
			},
			{
				Name:      "c3po@example.com",
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

	tx := txnForTestCase(t, s.db, s.db.DefaultOrg.ID)
	defaultOrg := s.db.DefaultOrg

	var providers, identities, groups, credentials, accessKeys int64

	err = tx.QueryRow("SELECT COUNT(*) FROM providers WHERE organization_id = ?;", defaultOrg.ID).Scan(&providers)
	assert.NilError(t, err)
	assert.Equal(t, int64(2), providers) // infra and okta

	grants, err := data.ListGrants(tx, data.ListGrantsOptions{})
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

	err = tx.QueryRow("SELECT COUNT(*) FROM identities WHERE organization_id = ? AND deleted_at IS null;", defaultOrg.ID).Scan(&identities)
	assert.NilError(t, err)
	assert.Equal(t, int64(5), identities)

	err = tx.QueryRow("SELECT COUNT(*) FROM groups WHERE organization_id = ?;", defaultOrg.ID).Scan(&groups)
	assert.NilError(t, err)
	assert.Equal(t, int64(1), groups) // Everyone

	err = tx.QueryRow("SELECT COUNT(*) FROM credentials WHERE organization_id = ?;", defaultOrg.ID).Scan(&credentials)
	assert.NilError(t, err)
	assert.Equal(t, int64(1), credentials) // sarah@example.com

	err = tx.QueryRow("SELECT COUNT(*) FROM access_keys WHERE organization_id = ?;", defaultOrg.ID).Scan(&accessKeys)
	assert.NilError(t, err)
	assert.Equal(t, int64(1), accessKeys) // c3po

	updatedConfig := Config{
		Providers: []Provider{
			{
				Name:         "atko",
				URL:          "new.example.com",
				ClientID:     "client-id-2",
				ClientSecret: "client-secret-2",
				AuthURL:      "new.example.com/v1/auth",
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

	providerCount, err := data.CountAllProviders(s.db)
	assert.NilError(t, err)
	assert.Equal(t, providerCount, int64(2)) // infra and atko

	provider, err := data.GetProvider(tx, data.GetProviderOptions{ByName: "atko"})
	assert.NilError(t, err)

	expected := &models.Provider{
		Model:              provider.Model,     // not relevant
		CreatedBy:          provider.CreatedBy, // not relevant
		Name:               "atko",
		URL:                "new.example.com",
		ClientID:           "client-id-2",
		ClientSecret:       "client-secret-2",
		Kind:               models.ProviderKindOIDC, // the kind gets the default value
		AuthURL:            "new.example.com/v1/auth",
		Scopes:             []string{"openid", "email", "groups"},
		OrganizationMember: models.OrganizationMember{OrganizationID: defaultOrg.ID},
	}

	cmpProvider := cmp.Options{
		cmp.FilterPath(
			opt.PathField(models.Provider{}, "ClientSecret"),
			cmpEncryptedAtRestEqual),
		cmp.FilterPath(
			opt.PathField(models.Provider{}, "Scopes"),
			cmp.Comparer(slices.Equal)),
		cmpopts.EquateEmpty(),
	}

	assert.DeepEqual(t, provider, expected, cmpProvider)

	grants, err = data.ListGrants(tx, data.ListGrantsOptions{})
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

	err = tx.QueryRow("SELECT COUNT(*) FROM identities WHERE organization_id = ? AND deleted_at IS null;", defaultOrg.ID).Scan(&identities)
	assert.NilError(t, err)
	assert.Equal(t, int64(2), identities)

	user, err := data.GetIdentity(s.db, data.GetIdentityOptions{ByName: "test@example.com"})
	assert.NilError(t, err)
	assert.Assert(t, user != nil)

	err = tx.QueryRow("SELECT COUNT(*) FROM groups WHERE organization_id = ? AND deleted_at IS null;", defaultOrg.ID).Scan(&groups)
	assert.NilError(t, err)
	assert.Equal(t, int64(1), groups)

	group, err := data.GetGroup(tx, data.GetGroupOptions{ByName: "Everyone"})
	assert.NilError(t, err)
	assert.Assert(t, group != nil)
}

func TestLoadAccessKey(t *testing.T) {
	s := setupServer(t)

	// access key that we will attempt to assign to multiple users
	testAccessKey := "aaaaaaaaaa.bbbbbbbbbbbbbbbbbbbbbbbb"

	// create a user and assign them an access key
	bob := &models.Identity{Name: "bob@example.com"}
	err := data.CreateIdentity(s.DB(), bob)
	assert.NilError(t, err)

	err = s.loadAccessKey(s.DB(), bob, testAccessKey)
	assert.NilError(t, err)

	t.Run("access key can be reloaded for the same identity it was issued for", func(t *testing.T) {
		err = s.loadAccessKey(s.DB(), bob, testAccessKey)
		assert.NilError(t, err)
	})

	t.Run("duplicate access key ID is rejected", func(t *testing.T) {
		alice := &models.Identity{Name: "alice@example.com"}
		err = data.CreateIdentity(s.DB(), alice)
		assert.NilError(t, err)

		err = s.loadAccessKey(s.DB(), alice, testAccessKey)
		assert.Error(t, err, "access key assigned to \"alice@example.com\" is already assigned to another user, a user's access key must have a unique ID")
	})
}

// getTestDefaultOrgUserDetails gets the attributes of a user created from a config file
func getTestDefaultOrgUserDetails(t *testing.T, server *Server, name string) (*models.Identity, *models.Credential, *models.AccessKey) {
	t.Helper()
	tx := txnForTestCase(t, server.db, server.db.DefaultOrg.ID)

	user, err := data.GetIdentity(tx, data.GetIdentityOptions{ByName: name})
	assert.NilError(t, err, "user")

	credential, err := data.GetCredentialByUserID(tx, user.ID)
	if !errors.Is(err, internal.ErrNotFound) {
		assert.NilError(t, err, "credentials")
	}

	keys, err := data.ListAccessKeys(tx, data.ListAccessKeyOptions{ByIssuedForID: user.ID})
	if !errors.Is(err, internal.ErrNotFound) {
		assert.NilError(t, err, "access_key")
	}

	// only return the first key
	var accessKey *models.AccessKey
	if len(keys) > 0 {
		accessKey = &keys[0]
	}

	return user, credential, accessKey
}

package server

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/cmd/cliopts"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
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

func TestLoadConfigUpdate(t *testing.T) {
	s := setupServer(t)

	config := Config{
		Users: []User{
			{
				Name: "r2d2@example.com",
				Role: "admin",
			},
			{
				Name:      "c3po@example.com",
				AccessKey: "TllVlekkUz.NFnxSlaPQLosgkNsyzaMttfC",
				Role:      "view",
			},
			{
				Name:     "sarah@email.com",
				Password: "supersecret",
			},
		},
	}

	err := s.loadConfig(config)
	assert.NilError(t, err)

	tx := txnForTestCase(t, s.db, s.db.DefaultOrg.ID)
	defaultOrg := s.db.DefaultOrg

	var identities, credentials, accessKeys int64

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

	assert.Equal(t, privileges["admin"], 1)
	assert.Equal(t, privileges["view"], 1)
	assert.Equal(t, privileges["connector"], 1)

	err = tx.QueryRow("SELECT COUNT(*) FROM identities WHERE organization_id = ? AND deleted_at IS null;", defaultOrg.ID).Scan(&identities)
	assert.NilError(t, err)
	assert.Equal(t, int64(4), identities)

	err = tx.QueryRow("SELECT COUNT(*) FROM credentials WHERE organization_id = ?;", defaultOrg.ID).Scan(&credentials)
	assert.NilError(t, err)
	assert.Equal(t, int64(1), credentials) // sarah@example.com

	err = tx.QueryRow("SELECT COUNT(*) FROM access_keys WHERE organization_id = ?;", defaultOrg.ID).Scan(&accessKeys)
	assert.NilError(t, err)
	assert.Equal(t, int64(1), accessKeys) // c3po

	updatedConfig := Config{
		Users: []User{
			{
				Name: "c3po@example.com",
				Role: "admin",
			},
		},
	}

	err = s.loadConfig(updatedConfig)
	assert.NilError(t, err)

	grants, err = data.ListGrants(tx, data.ListGrantsOptions{})
	assert.NilError(t, err)
	assert.Assert(t, is.Len(grants, 4))

	privileges = map[string]int{
		"admin":     0,
		"view":      0,
		"connector": 0,
	}

	for _, v := range grants {
		privileges[v.Privilege]++
	}

	assert.Equal(t, privileges["admin"], 2)
	assert.Equal(t, privileges["view"], 1)
	assert.Equal(t, privileges["connector"], 1)

	err = tx.QueryRow("SELECT COUNT(*) FROM identities WHERE organization_id = ? AND deleted_at IS null;", defaultOrg.ID).Scan(&identities)
	assert.NilError(t, err)
	assert.Equal(t, int64(4), identities)
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

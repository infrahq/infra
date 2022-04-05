package secrets

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/vault/api"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/infrahq/infra/testutil/docker"
)

func TestMain(m *testing.M) {
	defer func() {
		if r := recover(); r != nil {
			teardown()
			fmt.Println(r)
			os.Exit(1)
		}
	}()

	flag.Parse()
	setup()

	result := m.Run()

	teardown()
	// nolint
	os.Exit(result)
}

var (
	localstackCfg *aws.Config
	awskms        *kms.KMS
	containerIDs  []string
)

func setup() {
	if testing.Short() {
		return
	}

	var containerID string

	// setup localstack
	// eg docker run --rm -it -p 4566:4566 -p 4571:4571 localstack/localstack
	containerID = docker.LaunchContainer("localstack/localstack",
		[]docker.ExposedPort{
			{HostPort: 4566, ContainerPort: 4566},
		},
		nil, // cmd
		[]string{
			"SERVICES=secretsmanager,ssm,events",
		},
	)
	containerIDs = append(containerIDs, containerID)

	// setup kms
	containerID = docker.LaunchContainer("nsmithuk/local-kms", []docker.ExposedPort{
		{HostPort: 8380, ContainerPort: 8080},
	}, nil, nil)
	containerIDs = append(containerIDs, containerID)

	// setup vault
	containerID = docker.LaunchContainer("vault", []docker.ExposedPort{
		{HostPort: 8200, ContainerPort: 8200},
		{HostPort: 8201, ContainerPort: 8201},
	},
		nil,
		[]string{
			`VAULT_LOCAL_CONFIG={"disable_mlock":true}`,
			"SKIP_SETCAP=true",
			`VAULT_DEV_ROOT_TOKEN_ID=root`,
		},
	)
	containerIDs = append(containerIDs, containerID)

	// configure aws client
	sess := session.Must(session.NewSession())

	// for kms service
	cfg := aws.NewConfig().
		WithEndpoint("http://localhost:8380").
		WithCredentials(credentials.AnonymousCredentials).
		WithRegion("us-west-2")
	awskms = kms.New(sess, cfg)

	// for localstack (secrets manager, etc)
	localstackCfg = aws.NewConfig().
		WithCredentials(credentials.NewCredentials(&credentials.StaticProvider{
			Value: credentials.Value{
				AccessKeyID:     "test",
				SecretAccessKey: "test",
			},
		})).
		WithEndpoint("http://localhost:4566").
		WithRegion("us-east-1")
}

func teardown() {
	if testing.Short() {
		return
	}

	for _, containerID := range containerIDs {
		docker.KillContainer(containerID)
	}
}

func eachSecretStorageProvider(t *testing.T, eachFunc func(t *testing.T, p SecretStorage)) {
	eachProvider(t, func(t *testing.T, p interface{}) {
		if sp, ok := p.(SecretStorage); ok {
			eachFunc(t, sp)
		}
	})
}

func eachSymmetricKeyProvider(t *testing.T, eachFunc func(t *testing.T, p SymmetricKeyProvider)) {
	eachProvider(t, func(t *testing.T, p interface{}) {
		if sp, ok := p.(SymmetricKeyProvider); ok {
			eachFunc(t, sp)
		}
	})
}

func eachProvider(t *testing.T, eachFunc func(t *testing.T, p interface{})) {
	providers := map[string]interface{}{}

	// add AWS KMS
	k, err := NewAWSKMSSecretProvider(awskms)
	assert.NilError(t, err)

	providers["awskms"] = k

	// add AWS Secrets Manager
	sess := session.Must(session.NewSession())
	awssm := secretsmanager.New(sess, localstackCfg)
	sm := NewAWSSecretsManager(awssm)

	waitForLocalstackReady(t, awssm)

	providers["awssm"] = sm

	// add AWS SSM (Systems Manager Parameter Store)
	awsssm := ssm.New(sess, localstackCfg)
	ssm := NewAWSSSM(awsssm)

	providers["awsssm"] = ssm

	// add vault
	v, err := NewVaultSecretProvider("http://localhost:8200", "root", "")
	assert.NilError(t, err)

	waitForVaultReady(t, v)

	// make sure vault transit is configured
	_ = v.client.Sys().Mount("transit", &api.MountInput{Type: "transit"})

	providers["vault"] = v

	// add k8s
	clientset, err := kubernetes.NewForConfig(kubeConfig(t))
	assert.NilError(t, err)

	k8s := NewKubernetesSecretProvider(clientset, "default")

	providers["kubernetes"] = k8s

	// add native; depends on k8s secret storage
	n := NewNativeSecretProvider(k8s)

	providers["native"] = n

	// add "file"
	providers["file"] = &FileSecretProvider{
		FileConfig: FileConfig{
			GenericConfig: GenericConfig{
				Base64: true,
				// Base64Raw: true,
			},
		},
	}

	// add "env"
	providers["env"] = &EnvSecretProvider{
		GenericConfig: GenericConfig{Base64: true},
	}

	for name, provider := range providers {
		t.Run(name, func(t *testing.T) {
			eachFunc(t, provider)
		})
	}
}

func kubeConfig(t *testing.T) *rest.Config {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	assert.NilError(t, err)

	return restConfig
}

func TestSaveAndLoadSecret(t *testing.T) {
	if testing.Short() {
		t.Skip("test skipped in short mode")
		return
	}

	eachSecretStorageProvider(t, func(t *testing.T, storage SecretStorage) {
		t.Run("getting a secret that doesn't exist should error", func(t *testing.T) {
			_, err := storage.GetSecret("doesnt/exist")
			assert.ErrorIs(t, err, ErrNotFound)
		})

		t.Run("can set and get a secret", func(t *testing.T) {
			err := storage.SetSecret("foo/bar:secret", []byte("secret token"))
			assert.NilError(t, err)

			r, err := storage.GetSecret("foo/bar:secret")
			assert.NilError(t, err)

			assert.DeepEqual(t, []byte("secret token"), r)
		})

		t.Run("adding a new secret doesn't break past secret at same path", func(t *testing.T) {
			secret1 := []byte("secret token")
			secret2 := []byte("secret token 2")
			err := storage.SetSecret("foo2/bar", secret1)
			assert.NilError(t, err)

			err = storage.SetSecret("foo2/bar2", secret2)
			assert.NilError(t, err)

			r1, err := storage.GetSecret("foo2/bar")
			assert.NilError(t, err)

			r2, err := storage.GetSecret("foo2/bar2")
			assert.NilError(t, err)

			assert.DeepEqual(t, secret1, r1)
			assert.DeepEqual(t, secret2, r2)
		})

		t.Run("can set the same secret twice", func(t *testing.T) {
			err := storage.SetSecret("foo3/bar:secret", []byte("secret token"))
			assert.NilError(t, err)

			err = storage.SetSecret("foo3/bar:secret", []byte("new secret token"))
			assert.NilError(t, err)

			r, err := storage.GetSecret("foo3/bar:secret")
			assert.NilError(t, err)

			assert.DeepEqual(t, []byte("new secret token"), r)
		})
	})
}

func TestSealAndUnseal(t *testing.T) {
	if testing.Short() {
		t.Skip("test skipped in short mode")
		return
	}

	eachSymmetricKeyProvider(t, func(t *testing.T, p SymmetricKeyProvider) {
		noRootKeyYet := ""

		key, err := p.GenerateDataKey(noRootKeyYet)
		assert.NilError(t, err)
		assert.Assert(t, len(key.RootKeyID) != 0)

		assert.Assert(t, is.Len(key.unencrypted, 32)) // 256 bit keys should be used.

		key2, err := p.DecryptDataKey(key.RootKeyID, key.Encrypted)
		assert.NilError(t, err)

		assert.DeepEqual(t, key, key2)

		secretMessage := "Your scientists were so preoccupied with whether they could, they didn’t stop to think if they should"

		encrypted, err := Seal(key, []byte(secretMessage))
		assert.NilError(t, err)

		unsealed, err := Unseal(key, encrypted)
		assert.NilError(t, err)

		assert.DeepEqual(t, []byte(secretMessage), unsealed)
	})
}

func TestGeneratingASecondKeyFromARootKey(t *testing.T) {
	if testing.Short() {
		t.Skip("test skipped in short mode")
		return
	}

	eachSymmetricKeyProvider(t, func(t *testing.T, p SymmetricKeyProvider) {
		noRootKeyYet := ""

		key, err := p.GenerateDataKey(noRootKeyYet)
		assert.NilError(t, err)
		assert.Assert(t, len(key.RootKeyID) != 0)

		key2, err := p.GenerateDataKey(key.RootKeyID)
		assert.NilError(t, err)
		assert.Assert(t, len(key.RootKeyID) != 0)
		assert.Equal(t, key.RootKeyID, key2.RootKeyID)
		assert.Assert(t, key.unencrypted != key2.unencrypted)
	})
}

func TestSealSize(t *testing.T) {
	p := NewNativeSecretProvider(NewFileSecretProviderFromConfig(FileConfig{
		Path: os.TempDir(),
	}))

	key, err := p.GenerateDataKey("one")
	assert.NilError(t, err)
	assert.Assert(t, len(key.RootKeyID) != 0)

	secretMessage := "toast"

	encrypted, err := Seal(key, []byte(secretMessage))
	assert.NilError(t, err)

	assert.Assert(t, is.Len(encrypted, 72))

	orig, err := Unseal(key, encrypted)
	assert.NilError(t, err)
	assert.Equal(t, secretMessage, orig)

	encrypted, err = SealRaw(key, []byte(secretMessage))
	assert.NilError(t, err)

	assert.Assert(t, is.Len(encrypted, 54))

	orig, err = UnsealRaw(key, encrypted)
	assert.NilError(t, err)

	assert.Equal(t, secretMessage, orig)
}

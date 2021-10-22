package secrets

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/hashicorp/vault/api"
	"github.com/infrahq/infra/testutil/docker"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
	awskms       *kms.KMS
	awsssm       *secretsmanager.SecretsManager
	containerIDs []string
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
			"SERVICES=secretsmanager",
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
	cfg2 := aws.NewConfig().
		WithCredentials(credentials.NewCredentials(&credentials.StaticProvider{
			Value: credentials.Value{
				AccessKeyID:     "test",
				SecretAccessKey: "test",
			},
		})).
		WithEndpoint("http://localhost:4566").
		WithRegion("us-east-1")
	awsssm = secretsmanager.New(sess, cfg2)
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

func eachSecretSymmetricKeyProvider(t *testing.T, eachFunc func(t *testing.T, p SecretSymmetricKeyProvider)) {
	eachProvider(t, func(t *testing.T, p interface{}) {
		if sp, ok := p.(SecretSymmetricKeyProvider); ok {
			eachFunc(t, sp)
		}
	})
}

func eachProvider(t *testing.T, eachFunc func(t *testing.T, p interface{})) {
	providers := map[string]interface{}{}

	// add AWS KMS
	k, err := NewAWSKMSSecretProvider(awskms)
	require.NoError(t, err)

	providers["awskms"] = k

	// add AWS Secrets Manager
	ssm := NewAWSSecretsManager(awsssm)

	waitForSecretsManagerReady(t, awsssm)

	providers["awsssm"] = ssm

	// add vault
	v, err := NewVaultSecretProvider("http://localhost:8200", "root", "")
	require.NoError(t, err)

	waitForVaultReady(t, v)

	// make sure vault transit is configured
	_ = v.client.Sys().Mount("transit", &api.MountInput{Type: "transit"})

	providers["vault"] = v

	// add k8s
	clientset, err := kubernetes.NewForConfig(kubeConfig(t))
	require.NoError(t, err)

	k8s := NewKubernetesSecretProvider(clientset, "infrahq")

	providers["kubernetes"] = k8s

	for name, provider := range providers {
		t.Run(name, func(t *testing.T) {
			eachFunc(t, provider)
		})
	}
}

func kubeConfig(t *testing.T) *rest.Config {
	tlsClientConfig := rest.TLSClientConfig{}

	home := os.ExpandEnv("$HOME")
	f, err := os.Open(home + "/.kube/config")
	require.NoError(t, err)

	config := struct {
		Clusters []struct {
			Name    string `yaml:"name"`
			Cluster struct {
				Server                   string `yaml:"server"`
				CertificateAuthorityData string `yaml:"certificate-authority-data"`
			} `yaml:"cluster"`
		} `yaml:"clusters"`
		Contexts []struct {
			Context struct {
				Cluster string `yaml:"cluster"`
				User    string `yaml:"user"`
				Name    string `yaml:"name"`
			} `yaml:"context"`
		}
		CurrentContext string `yaml:"current-context"`
		Users          []struct {
			Name string `yaml:"name"`
			User struct {
				ClientCertificateData string `yaml:"client-certificate-data"`
				ClientKeyData         string `yaml:"client-key-data"`
			}
		}
	}{}

	b, err := ioutil.ReadAll(f)
	require.NoError(t, err)

	err = yaml.Unmarshal(b, &config)
	require.NoError(t, err)

	server := ""

	for _, cluster := range config.Clusters {
		if cluster.Name == config.CurrentContext {
			c := cluster.Cluster
			server = c.Server

			if len(c.CertificateAuthorityData) > 0 {
				ca, err := base64.StdEncoding.DecodeString(c.CertificateAuthorityData)
				require.NoError(t, err)
				certData, err := base64.StdEncoding.DecodeString(config.Users[0].User.ClientCertificateData)
				require.NoError(t, err)
				keyData, err := base64.StdEncoding.DecodeString(config.Users[0].User.ClientKeyData)
				require.NoError(t, err)

				tlsClientConfig.CAData = ca
				tlsClientConfig.CertData = certData
				tlsClientConfig.KeyData = keyData
			}
		}
	}

	return &rest.Config{
		Host:            server,
		TLSClientConfig: tlsClientConfig,
	}
}

func TestSaveAndLoadSecret(t *testing.T) {
	if testing.Short() {
		t.Skip("test skipped in short mode")
		return
	}

	eachSecretStorageProvider(t, func(t *testing.T, storage SecretStorage) {
		t.Run("getting a secret that doesn't exist probably shouldn't error", func(t *testing.T) {
			_, err := storage.GetSecret("doesnt/exist")
			require.NoError(t, err)
		})

		t.Run("can set and get a secret", func(t *testing.T) {
			err := storage.SetSecret("foo/bar:secret", []byte("secret token"))
			require.NoError(t, err)

			r, err := storage.GetSecret("foo/bar:secret")
			require.NoError(t, err)

			require.Equal(t, []byte("secret token"), r)
		})

		t.Run("adding a new secret doesn't break past secret at same path", func(t *testing.T) {
			secret1 := []byte("secret token")
			secret2 := []byte("secret token 2")
			err := storage.SetSecret("foo2/bar", secret1)
			require.NoError(t, err)

			err = storage.SetSecret("foo2/bar2", secret2)
			require.NoError(t, err)

			r1, err := storage.GetSecret("foo2/bar")
			require.NoError(t, err)

			r2, err := storage.GetSecret("foo2/bar2")
			require.NoError(t, err)

			require.Equal(t, secret1, r1)
			require.Equal(t, secret2, r2)
		})

		t.Run("can set the same secret twice", func(t *testing.T) {
			err := storage.SetSecret("foo3/bar:secret", []byte("secret token"))
			require.NoError(t, err)

			err = storage.SetSecret("foo3/bar:secret", []byte("new secret token"))
			require.NoError(t, err)

			r, err := storage.GetSecret("foo3/bar:secret")
			require.NoError(t, err)

			require.Equal(t, []byte("new secret token"), r)
		})
	})
}

func TestSealAndUnseal(t *testing.T) {
	if testing.Short() {
		t.Skip("test skipped in short mode")
		return
	}

	eachSecretSymmetricKeyProvider(t, func(t *testing.T, p SecretSymmetricKeyProvider) {
		noRootKeyYet := ""

		key, err := p.GenerateDataKey("random/name:foo", noRootKeyYet)
		require.NoError(t, err)
		require.NotEmpty(t, key.RootKeyID)

		key2, err := p.DecryptDataKey(key.RootKeyID, key.Encrypted)
		require.NoError(t, err)

		require.Equal(t, key, key2)

		secretMessage := "Your scientists were so preoccupied with whether they could, they didn’t stop to think if they should"

		encrypted, err := Seal(key, []byte(secretMessage))
		require.NoError(t, err)

		unsealed, err := Unseal(key, encrypted)
		require.NoError(t, err)

		require.Equal(t, []byte(secretMessage), unsealed)
	})
}

func TestGeneratingASecondKeyFromARootKey(t *testing.T) {
	if testing.Short() {
		t.Skip("test skipped in short mode")
		return
	}

	eachSecretSymmetricKeyProvider(t, func(t *testing.T, p SecretSymmetricKeyProvider) {
		noRootKeyYet := ""

		key, err := p.GenerateDataKey("key:test/foo", noRootKeyYet)
		require.NoError(t, err)
		require.NotEmpty(t, key.RootKeyID)

		key2, err := p.GenerateDataKey("key:test/foo", key.RootKeyID)
		require.NoError(t, err)
		require.NotEmpty(t, key.RootKeyID)
		require.Equal(t, key.RootKeyID, key2.RootKeyID)
		require.NotEqual(t, key.unencrypted, key2.unencrypted)
	})
}

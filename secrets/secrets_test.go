package secrets

import (
	"flag"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/hashicorp/vault/api"
	"github.com/infrahq/infra/testutil/docker"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	flag.Parse()
	setup()

	result := m.Run()

	teardown()
	os.Exit(result)
}

var (
	awskms       *kms.KMS
	containerIDs []string
)

func setup() {
	if testing.Short() {
		return
	}

	containerID := docker.LaunchContainer("nsmithuk/local-kms", []docker.ExposedPort{
		{HostPort: 8380, ContainerPort: 8080},
	}, nil, nil)
	containerIDs = append(containerIDs, containerID)

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

	cfg := aws.NewConfig()
	cfg.Endpoint = aws.String("http://localhost:8380")
	cfg.Credentials = credentials.AnonymousCredentials
	cfg.Region = aws.String("us-west-2")
	awskms = kms.New(session.Must(session.NewSession()), cfg)
}

func teardown() {
	if testing.Short() {
		return
	}

	for _, containerID := range containerIDs {
		docker.KillContainer(containerID)
	}
}

func eachProvider(t *testing.T, eachFunc func(p SecretProvider)) {
	providers := []SecretProvider{}

	// add aws
	k, err := NewAWSKMSSecretProvider(awskms)
	require.NoError(t, err)

	providers = append(providers, k)

	// add vault
	v, err := NewVaultSecretProvider("http://localhost:8200", "root", "")
	require.NoError(t, err)

	waitForVaultReady(t, v)

	// make sure vault transit is configured
	_ = v.client.Sys().Mount("transit", &api.MountInput{Type: "transit"})

	providers = append(providers, v)

	for _, provider := range providers {
		eachFunc(provider)
	}
}

func TestSaveAndLoadSecret(t *testing.T) {
	if testing.Short() {
		t.Skip("test skipped in short mode")
		return
	}

	eachProvider(t, func(p SecretProvider) {
		// if provider implements secret storage...
		if storage, ok := p.(SecretStorage); ok {
			err := storage.SetSecret("foo", []byte("secret token"))
			require.NoError(t, err)

			r, err := storage.GetSecret("foo")
			require.NoError(t, err)

			require.Equal(t, []byte("secret token"), r)
		}
	})
}

func TestSealAndUnseal(t *testing.T) {
	if testing.Short() {
		t.Skip("test skipped in short mode")
		return
	}

	eachProvider(t, func(p SecretProvider) {
		noRootKeyYet := ""

		key, err := p.GenerateDataKey("random", noRootKeyYet)
		require.NoError(t, err)
		require.NotEmpty(t, key.RootKeyID)

		key2, err := p.DecryptDataKey(key.RootKeyID, key.Encrypted)
		require.NoError(t, err)

		require.Equal(t, key, key2)

		encrypted, err := Seal(key, []byte("Your scientists were so preoccupied with whether they could, they didn’t stop to think if they should"))
		require.NoError(t, err)

		r, err := Unseal(key, encrypted)
		require.NoError(t, err)

		require.Equal(t, []byte("Your scientists were so preoccupied with whether they could, they didn’t stop to think if they should"), r)
	})
}

func TestGeneratingASecondKeyFromARootKey(t *testing.T) {
	if testing.Short() {
		t.Skip("test skipped in short mode")
		return
	}

	eachProvider(t, func(p SecretProvider) {
		noRootKeyYet := ""

		key, err := p.GenerateDataKey("keytest", noRootKeyYet)
		require.NoError(t, err)
		require.NotEmpty(t, key.RootKeyID)

		key2, err := p.GenerateDataKey("keytest", key.RootKeyID)
		require.NoError(t, err)
		require.NotEmpty(t, key.RootKeyID)
		require.Equal(t, key.RootKeyID, key2.RootKeyID)
		require.NotEqual(t, key.unencrypted, key2.unencrypted)
	})
}

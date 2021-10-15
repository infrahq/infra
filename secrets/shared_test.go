package secrets

import (
	"flag"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/infrahq/infra/testutil/docker"
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

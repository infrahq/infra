package kubernetes

import (
	"testing"

	"github.com/infrahq/infra/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
)

func TestInvalidSecretFormats(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	testConfig := &rest.Config{
		Host: "https://localhost",
	}
	clientset, err := kubernetes.NewForConfig(testConfig)
	require.NoError(t, err)
	testK8s := secrets.NewKubernetesSecretProvider(clientset, "infrahq")

	_, err = testK8s.GetSecret("invalid-secret-format")
	assert.NotNil(t, err)
	assert.Equal(t, "invalid Kubernetes secret path specified, expected exactly 2 parts but was 1", err.Error())

	_, err = testK8s.GetSecret("")
	assert.NotNil(t, err)
	assert.Equal(t, "invalid Kubernetes secret path specified, expected exactly 2 parts but was 1", err.Error())

	_, err = testK8s.GetSecret("/")
	assert.NotNil(t, err)
	assert.Equal(t, "resource name may not be empty", err.Error())

	_, err = testK8s.GetSecret("invalid/number/path")
	assert.NotNil(t, err)
	assert.Equal(t, "invalid Kubernetes secret path specified, expected exactly 2 parts but was 3", err.Error())
}

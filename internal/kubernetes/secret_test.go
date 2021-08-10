package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	rest "k8s.io/client-go/rest"
)

func TestInvalidSecretFormats(t *testing.T) {
	testConfig := &rest.Config{
		Host: "https://localhost",
	}
	testSecretReader := NewSecretReader("test-namespace")
	testK8s := &Kubernetes{Config: testConfig, SecretReader: testSecretReader}

	_, err := testK8s.GetSecret("invalid-secret-format")
	assert.NotNil(t, err)
	assert.Equal(t, "invalid Kubernetes secret path seperated, expected exactly 2 parts but was 1", err.Error())

	_, err = testK8s.GetSecret("")
	assert.NotNil(t, err)
	assert.Equal(t, "invalid Kubernetes secret path seperated, expected exactly 2 parts but was 1", err.Error())

	_, err = testK8s.GetSecret("/")
	assert.NotNil(t, err)
	assert.Equal(t, "resource name may not be empty", err.Error())

	_, err = testK8s.GetSecret("invalid/number/path")
	assert.NotNil(t, err)
	assert.Equal(t, "invalid Kubernetes secret path seperated, expected exactly 2 parts but was 3", err.Error())
}

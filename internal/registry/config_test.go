package registry

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"

	"github.com/infrahq/infra/secrets"
)

func setupRegistry(t *testing.T) *Registry {
	testdata, err := ioutil.ReadFile("_testdata/infra.yaml")
	require.NoError(t, err)

	return setupRegistryWithConfig(t, testdata)
}

func setupRegistryWithConfig(t *testing.T, config []byte) *Registry {
	return setupRegistryWithConfigAndDb(t, config, setupDB(t))
}

func setupRegistryWithConfigAndDb(t *testing.T, config []byte, db *gorm.DB) *Registry {
	var options Options
	err := yaml.Unmarshal(config, &options)
	require.NoError(t, err)

	r := &Registry{options: options, db: db}

	err = r.importSecrets()
	require.NoError(t, err)

	err = r.importConfig()
	require.NoError(t, err)

	return r
}

func TestImportKeyProvider(t *testing.T) {
	r := setupRegistry(t)

	sp, ok := r.keys["native"]
	require.True(t, ok)
	require.IsType(t, &secrets.NativeSecretProvider{}, sp)
}

package registry

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/secrets"
)

var (
	providerOkta *models.Provider

	userBond   *models.User
	userBourne *models.User

	groupEveryone  *models.Group
	groupEngineers *models.Group

	destinationAAA *models.Destination
	destinationBBB *models.Destination
	destinationCCC *models.Destination

	labelKubernetes = models.Label{Value: "kubernetes"}
	labelUSWest1    = models.Label{Value: "us-west-1"}
	labelUSEast1    = models.Label{Value: "us-east-1"}
)

func setupDB(t *testing.T) *gorm.DB {
	setupLogging(t)

	driver, err := data.NewSQLiteDriver("file::memory:")
	require.NoError(t, err)

	db, err := data.NewDB(driver)
	require.NoError(t, err)

	fp := secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{
		Path: os.TempDir(),
	})

	kp := secrets.NewNativeSecretProvider(fp)
	key, err := kp.GenerateDataKey("")
	require.NoError(t, err)

	models.SymmetricKey = key

	providerOkta, err = data.CreateProvider(db, &models.Provider{
		Kind:         models.ProviderKindOkta,
		Domain:       "test.okta.com",
		ClientSecret: "supersecret",
	})
	require.NoError(t, err)

	userBond = &models.User{Email: "jbond@infrahq.com"}
	err = data.CreateUser(db, userBond)
	require.NoError(t, err)

	userBourne = &models.User{Email: "jbourne@infrahq.com"}
	err = data.CreateUser(db, userBourne)
	require.NoError(t, err)

	groupEveryone = &models.Group{Name: "Everyone"}
	err = data.CreateGroup(db, groupEveryone)
	require.NoError(t, err)

	groupEngineers = &models.Group{Name: "Engineering"}
	err = data.CreateGroup(db, groupEngineers)
	require.NoError(t, err)

	err = data.BindUserGroups(db, userBourne, *groupEveryone)
	require.NoError(t, err)

	destinationAAA = &models.Destination{
		Kind:     models.DestinationKindKubernetes,
		Name:     "AAA",
		NodeID:   "AAA",
		Endpoint: "develop.infrahq.com",
		Labels: []models.Label{
			labelKubernetes,
		},
		Kubernetes: models.DestinationKubernetes{
			CA: "myca",
		},
	}
	err = data.CreateDestination(db, destinationAAA)
	require.NoError(t, err)

	destinationBBB = &models.Destination{
		Kind:     models.DestinationKindKubernetes,
		Name:     "BBB",
		NodeID:   "BBB",
		Endpoint: "stage.infrahq.com",
		Labels: []models.Label{
			labelKubernetes,
			labelUSWest1,
		},
		Kubernetes: models.DestinationKubernetes{
			CA: "myotherca",
		},
	}
	err = data.CreateDestination(db, destinationBBB)
	require.NoError(t, err)

	destinationCCC = &models.Destination{
		Kind:     models.DestinationKindKubernetes,
		Name:     "CCC",
		NodeID:   "CCC",
		Endpoint: "production.infrahq.com",
		Labels: []models.Label{
			labelKubernetes,
			labelUSEast1,
		},
		Kubernetes: models.DestinationKubernetes{
			CA: "myotherotherca",
		},
	}
	err = data.CreateDestination(db, destinationCCC)
	require.NoError(t, err)

	return db
}

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

	err = r.importSecretKeys()
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

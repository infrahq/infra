package registry

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
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

	userBond, err = data.CreateUser(db, &models.User{Email: "jbond@infrahq.com"})
	require.NoError(t, err)

	userBourne, err = data.CreateUser(db, &models.User{Email: "jbourne@infrahq.com"})
	require.NoError(t, err)

	groupEveryone, err = data.CreateGroup(db, &models.Group{Name: "Everyone"})
	require.NoError(t, err)

	groupEngineers, err = data.CreateGroup(db, &models.Group{Name: "Engineering"})
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

	err = r.importConfig()
	require.NoError(t, err)

	return r
}

func userGrants(t *testing.T, grants []models.Grant, email string) map[string][]string {
	destinations := make(map[string][]string)

	for _, grant := range grants {
		destinationName := grant.Destination.Name

		var key string

		switch grant.Kind {
		case models.GrantKindKubernetes:
			key = fmt.Sprintf("%s:%s:%s", grant.Kubernetes.Kind, grant.Kubernetes.Name, grant.Kubernetes.Namespace)
		case models.GrantKindInfra:
		default:
			require.Fail(t, "unknown grant kind")
		}

		for _, user := range grant.Users {
			if user.Email != email {
				continue
			}

			if _, ok := destinations[key]; !ok {
				destinations[key] = make([]string, 0)
			}

			destinations[key] = append(destinations[key], destinationName)
		}
	}

	return destinations
}

func TestImportUserGrants(t *testing.T) {
	r := setupRegistry(t)

	grants, err := data.ListGrants(r.db)
	require.NoError(t, err)

	bond := userGrants(t, grants, userBond.Email)
	require.ElementsMatch(t, []string{"AAA", "BBB", "CCC"}, bond["cluster-role:admin:"])
	require.ElementsMatch(t, []string{"CCC"}, bond["role:audit:infrahq"])
	require.ElementsMatch(t, []string{"CCC"}, bond["role:audit:development"])
	require.ElementsMatch(t, []string{"CCC"}, bond["role:pod-create:infrahq"])
	require.ElementsMatch(t, []string(nil), bond["role:view"])

	unknown := userGrants(t, grants, "unknown@infrahq.com")
	require.ElementsMatch(t, []string(nil), unknown["grant:writer"])
}

func groupGrants(t *testing.T, grants []models.Grant, name string) map[string][]string {
	destinations := make(map[string][]string)

	for _, grant := range grants {
		destinationName := grant.Destination.Name

		var key string

		switch grant.Kind {
		case models.GrantKindKubernetes:
			key = fmt.Sprintf("%s:%s:%s", grant.Kubernetes.Kind, grant.Kubernetes.Name, grant.Kubernetes.Namespace)
		case models.GrantKindInfra:
		default:
			require.Fail(t, "unknown grant kind")
		}

		for _, group := range grant.Groups {
			if group.Name != name {
				continue
			}

			if _, ok := destinations[key]; !ok {
				destinations[key] = make([]string, 0)
			}

			destinations[key] = append(destinations[key], destinationName)
		}
	}

	return destinations
}

func TestImportGroupGrants(t *testing.T) {
	r := setupRegistry(t)

	grants, err := data.ListGrants(r.db)
	require.NoError(t, err)

	everyone := groupGrants(t, grants, groupEveryone.Name)
	require.ElementsMatch(t, []string{"AAA"}, everyone["cluster-role:writer:"])
	require.ElementsMatch(t, []string{"CCC"}, everyone["role:audit:infrahq"])
	require.ElementsMatch(t, []string{"CCC"}, everyone["role:audit:development"])
	require.ElementsMatch(t, []string{"CCC"}, everyone["role:pod-create:infrahq"])

	engineering := groupGrants(t, grants, groupEngineers.Name)
	require.ElementsMatch(t, []string{"BBB"}, engineering["role:writer:"])
}

func TestImportGrantsUnknownDestinations(t *testing.T) {
	r := setupRegistry(t)

	grants, err := data.ListGrants(r.db)
	require.NoError(t, err)

	for _, g := range grants {
		_, err := data.GetDestination(r.db, r.db.Where("id = (?)", g.DestinationID))
		require.NoError(t, err)
	}
}

func TestImportGrantsNoMatchingLabels(t *testing.T) {
	r := setupRegistry(t)

	_, err := data.GetGrantByModel(r.db, &models.Grant{
		Kind:       models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{Name: "view"},
	})
	require.ErrorIs(t, err, internal.ErrNotFound)
}

func TestFirstNamespaceThenNoNamespace(t *testing.T) {
	db := setupDB(t)

	withNamespace := `
providers:
  - kind: okta
    domain: https://test.example.com
    clientID: plaintext:0oapn0qwiQPiMIyR35d6
    clientSecret: kubernetes:okta-secrets/clientSecret
groups:
  - name: Everyone
    provider: okta
    grants:
      - name: cluster-admin
        kind: cluster-role
        destinations:
          - name: AAA
            kind: kubernetes
            namespaces:
              - infrahq
`

	withoutNamespace := `
providers:
  - kind: okta
    domain: https://test.example.com
    clientID: plaintext:0oapn0qwiQPiMIyR35d6
    clientSecret: kubernetes:okta-secrets/clientSecret
groups:
  - name: Everyone
    provider: okta
    grants:
      - name: cluster-admin
        kind: cluster-role
        destinations:
          - name: AAA
            kind: kubernetes
`

	setupRegistryWithConfigAndDb(t, []byte(withNamespace), db)

	grants, err := data.ListGrants(db)
	require.NoError(t, err)
	require.Len(t, grants, 1)
	require.Equal(t, "infrahq", grants[0].Kubernetes.Namespace)

	setupRegistryWithConfigAndDb(t, []byte(withoutNamespace), db)

	grants, err = data.ListGrants(db)
	require.NoError(t, err)
	require.Len(t, grants, 1)
	require.Equal(t, "", grants[0].Kubernetes.Namespace)
}

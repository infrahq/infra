package registry

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/secrets"
)

func configure(t *testing.T, db *gorm.DB) (*Registry, *gorm.DB) {
	if db == nil {
		db = setupDB(t)
	}

	testdata, err := ioutil.ReadFile("_testdata/infra.yaml")
	require.NoError(t, err)

	r := &Registry{db: db}
	err = r.importSecretsConfig(testdata)
	require.NoError(t, err)

	err = r.importConfig(testdata)
	require.NoError(t, err)

	return r, db
}

func userGrants(t *testing.T, grants []models.Grant, email string) map[string][]string {
	destinations := make(map[string][]string)

	for _, grant := range grants {
		destinationName := grant.Destination.Name

		var key string

		switch grant.Kind {
		case models.GrantKindKubernetes:
			key = fmt.Sprintf("%s:%s:%s", grant.Kubernetes.Kind, grant.Kubernetes.Name, grant.Kubernetes.Namespace)
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
	_, db := configure(t, nil)

	grants, err := data.ListGrants(db)
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
	_, db := configure(t, nil)

	grants, err := data.ListGrants(db)
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
	_, db := configure(t, nil)

	grants, err := data.ListGrants(db)
	require.NoError(t, err)

	for _, r := range grants {
		_, err := data.GetDestination(db, db.Where("id = (?)", r.DestinationID))
		require.NoError(t, err)
	}
}

func TestImportGrantsNoMatchingLabels(t *testing.T) {
	_, db := configure(t, nil)

	_, err := data.GetGrantByModel(db, &models.Grant{
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
    apiToken: kubernetes:okta-secrets/apiToken
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
    apiToken: kubernetes:okta-secrets/apiToken
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

	r := Registry{db: db}

	err := r.importConfig([]byte(withNamespace))
	require.NoError(t, err)

	grants, err := data.ListGrants(db)
	require.NoError(t, err)
	require.Len(t, grants, 1)
	require.Equal(t, "infrahq", grants[0].Kubernetes.Namespace)

	err = r.importConfig([]byte(withoutNamespace))
	require.NoError(t, err)

	grants, err = data.ListGrants(db)
	require.NoError(t, err)
	require.Len(t, grants, 1)
	require.Equal(t, "", grants[0].Kubernetes.Namespace)
}

func TestImportKeyProvider(t *testing.T) {
	r, _ := configure(t, nil)

	sp, ok := r.keyProvider["native"]
	require.True(t, ok)
	require.IsType(t, &secrets.NativeSecretProvider{}, sp)
}

package registry

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
)

func configure(t *testing.T, db *gorm.DB) *gorm.DB {
	if db == nil {
		db = setupDB(t)
	}

	testdata, err := ioutil.ReadFile("_testdata/infra.yaml")
	require.NoError(t, err)

	r := Registry{db: db}
	err = r.importConfig(testdata)
	require.NoError(t, err)

	return db
}

func userGrants(t *testing.T, grants []models.Grant, email string) map[string][]string {
	grants := make(map[string][]string)

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

			if _, ok := grants[key]; !ok {
				grants[key] = make([]string, 0)
			}

			grants[key] = append(grants[key], destinationName)
		}
	}

	return grants
}

func TestImportUserGrants(t *testing.T) {
	db := configure(t, nil)

	grants, err := data.ListGrants(db, &models.Grant{})
	require.NoError(t, err)

	bond := userGrants(t, grants, userBond.Email)
	require.ElementsMatch(t, []string{"AAA", "BBB", "CCC"}, bond["cluster-grant:admin:"])
	require.ElementsMatch(t, []string{"CCC"}, bond["grant:audit:infrahq"])
	require.ElementsMatch(t, []string{"CCC"}, bond["grant:audit:development"])
	require.ElementsMatch(t, []string{"CCC"}, bond["grant:pod-create:infrahq"])
	require.ElementsMatch(t, []string(nil), bond["grant:view"])

	unknown := userGrants(t, grants, "unknown@infrahq.com")
	require.ElementsMatch(t, []string(nil), unknown["grant:writer"])
}

func groupGrants(t *testing.T, grants []models.Grant, name string) map[string][]string {
	grants := make(map[string][]string)

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

			if _, ok := grants[key]; !ok {
				grants[key] = make([]string, 0)
			}

			grants[key] = append(grants[key], destinationName)
		}
	}

	return grants
}

func TestImportGroupGrants(t *testing.T) {
	db := configure(t, nil)

	grants, err := data.ListGrants(db, &models.Grant{})
	require.NoError(t, err)

	everyone := groupGrants(t, grants, groupEveryone.Name)
	require.ElementsMatch(t, []string{"AAA"}, everyone["cluster-grant:writer:"])
	require.ElementsMatch(t, []string{"CCC"}, everyone["grant:audit:infrahq"])
	require.ElementsMatch(t, []string{"CCC"}, everyone["grant:audit:development"])
	require.ElementsMatch(t, []string{"CCC"}, everyone["grant:pod-create:infrahq"])

	engineering := groupGrants(t, grants, groupEngineers.Name)
	require.ElementsMatch(t, []string{"BBB"}, engineering["grant:writer:"])
}

func TestImportGrantsUnknownDestinations(t *testing.T) {
	db := configure(t, nil)

	grants, err := data.ListGrants(db, &models.Grant{})
	require.NoError(t, err)

	for _, r := range grants {
		_, err := data.GetDestination(db, db.Where("id = (?)", r.DestinationID))
		require.NoError(t, err)
	}
}

func TestImportGrantsNoMatchingLabels(t *testing.T) {
	db := configure(t, nil)

	grants, err := data.ListGrants(db, data.GrantSelector(db, &models.Grant{
		Kind: models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{
			Name: "view",
		},
	}))
	require.NoError(t, err)
	require.Len(t, grants, 0)
}

func TestImportGrantsRemovesUnusedGrants(t *testing.T) {
	db := setupDB(t)

	unused, err := data.CreateGrant(db, &models.Grant{})
	require.NoError(t, err)

	_ = configure(t, db)

	_, err = data.GetGrant(db, unused)
	require.EqualError(t, err, "record not found")
}

func TestImportProvidersOverrideDuplicate(t *testing.T) {
	db := configure(t, nil)

	providers, err := data.ListProviders(db, &models.Provider{})
	require.NoError(t, err)
	require.Len(t, providers, 1)
}

func TestCleanupDomain(t *testing.T) {
	p := ConfigProvider{Domain: "dev123123-admin.okta.com "}
	p.cleanupDomain()
	require.Equal(t, "dev123123.okta.com", p.Domain)

	p = ConfigProvider{Domain: "dev123123.okta.com "}
	p.cleanupDomain()
	require.Equal(t, "dev123123.okta.com", p.Domain)

	p = ConfigProvider{Domain: "https://dev123123.okta.com "}
	p.cleanupDomain()
	require.Equal(t, "dev123123.okta.com", p.Domain)

	p = ConfigProvider{Domain: "http://dev123123.okta.com "}
	p.cleanupDomain()
	require.Equal(t, "dev123123.okta.com", p.Domain)
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
        kind: cluster-grant
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
        kind: cluster-grant
        destinations:
          - name: AAA
            kind: kubernetes
`

	r := Registry{db: db}

	err := r.importConfig([]byte(withNamespace))
	require.NoError(t, err)

	grants, err := data.ListGrants(db, &models.Grant{})
	require.NoError(t, err)
	require.Len(t, grants, 1)
	require.Equal(t, "infrahq", grants[0].Kubernetes.Namespace)

	err = r.importConfig([]byte(withoutNamespace))
	require.NoError(t, err)

	grants, err = data.ListGrants(db, &models.Grant{})
	require.NoError(t, err)
	require.Len(t, grants, 1)
	require.Equal(t, "", grants[0].Kubernetes.Namespace)
}

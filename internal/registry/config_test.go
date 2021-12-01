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

func userGrants(t *testing.T, roles []models.Role, email string) map[string][]string {
	grants := make(map[string][]string)

	for _, role := range roles {
		destinationName := role.Destination.Name

		var key string

		switch role.Kind {
		case models.RoleKindKubernetes:
			key = fmt.Sprintf("%s:%s:%s", role.Kubernetes.Kind, role.Kubernetes.Name, role.Kubernetes.Namespace)
		default:
			require.Fail(t, "unknown role kind")
		}

		for _, user := range role.Users {
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

func TestImportUserRoles(t *testing.T) {
	db := configure(t, nil)

	roles, err := data.ListRoles(db, &models.Role{})
	require.NoError(t, err)

	bond := userGrants(t, roles, userBond.Email)
	require.ElementsMatch(t, []string{"AAA", "BBB", "CCC"}, bond["cluster-role:admin:"])
	require.ElementsMatch(t, []string{"CCC"}, bond["role:audit:infrahq"])
	require.ElementsMatch(t, []string{"CCC"}, bond["role:audit:development"])
	require.ElementsMatch(t, []string{"CCC"}, bond["role:pod-create:infrahq"])
	require.ElementsMatch(t, []string(nil), bond["role:view"])

	unknown := userGrants(t, roles, "unknown@infrahq.com")
	require.ElementsMatch(t, []string(nil), unknown["role:writer"])
}

func groupGrants(t *testing.T, roles []models.Role, name string) map[string][]string {
	grants := make(map[string][]string)

	for _, role := range roles {
		destinationName := role.Destination.Name

		var key string

		switch role.Kind {
		case models.RoleKindKubernetes:
			key = fmt.Sprintf("%s:%s:%s", role.Kubernetes.Kind, role.Kubernetes.Name, role.Kubernetes.Namespace)
		default:
			require.Fail(t, "unknown role kind")
		}

		for _, group := range role.Groups {
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

func TestImportGroupRoles(t *testing.T) {
	db := configure(t, nil)

	roles, err := data.ListRoles(db, &models.Role{})
	require.NoError(t, err)

	everyone := groupGrants(t, roles, groupEveryone.Name)
	require.ElementsMatch(t, []string{"AAA"}, everyone["cluster-role:writer:"])
	require.ElementsMatch(t, []string{"CCC"}, everyone["role:audit:infrahq"])
	require.ElementsMatch(t, []string{"CCC"}, everyone["role:audit:development"])
	require.ElementsMatch(t, []string{"CCC"}, everyone["role:pod-create:infrahq"])

	engineering := groupGrants(t, roles, groupEngineers.Name)
	require.ElementsMatch(t, []string{"BBB"}, engineering["role:writer:"])
}

func TestImportRolesUnknownDestinations(t *testing.T) {
	db := configure(t, nil)

	roles, err := data.ListRoles(db, &models.Role{})
	require.NoError(t, err)

	for _, r := range roles {
		_, err := data.GetDestination(db, db.Where("id = (?)", r.DestinationID))
		require.NoError(t, err)
	}
}

func TestImportRolesNoMatchingLabels(t *testing.T) {
	db := configure(t, nil)

	roles, err := data.ListRoles(db, data.RoleSelector(db, &models.Role{
		Kind: models.RoleKindKubernetes,
		Kubernetes: models.RoleKubernetes{
			Name: "view",
		},
	}))
	require.NoError(t, err)
	require.Len(t, roles, 0)
}

func TestImportRolesRemovesUnusedRoles(t *testing.T) {
	db := setupDB(t)

	unused, err := data.CreateRole(db, &models.Role{})
	require.NoError(t, err)

	_ = configure(t, db)

	_, err = data.GetRole(db, unused)
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
    roles:
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
    roles:
      - name: cluster-admin
        kind: cluster-role
        destinations:
          - name: AAA
            kind: kubernetes
`

	r := Registry{db: db}

	err := r.importConfig([]byte(withNamespace))
	require.NoError(t, err)

	roles, err := data.ListRoles(db, &models.Role{})
	require.NoError(t, err)
	require.Len(t, roles, 1)
	require.Equal(t, "infrahq", roles[0].Kubernetes.Namespace)

	err = r.importConfig([]byte(withoutNamespace))
	require.NoError(t, err)

	roles, err = data.ListRoles(db, &models.Role{})
	require.NoError(t, err)
	require.Len(t, roles, 1)
	require.Equal(t, "", roles[0].Kubernetes.Namespace)
}

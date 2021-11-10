package registry

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var db *gorm.DB

var (
	fakeOktaProvider = Provider{Id: "001", Kind: ProviderKindOkta, Domain: "test.example.com", ClientSecret: "kubernetes:okta-secrets/apiToken", APIToken: "kubernetes:okta-secrets/apiToken"}
	adminUser        = User{Id: "001", Email: "admin@example.com"}
	standardUser     = User{Id: "002", Email: "user@example.com"}
	iosDevUser       = User{Id: "003", Email: "woz@example.com"}
	iosDevGroup      = Group{Name: "ios-developers", ProviderId: fakeOktaProvider.Id}
	macAdminGroup    = Group{Name: "mac-admins", ProviderId: fakeOktaProvider.Id}
	notInConfigRole  = Role{Name: "does-not-exist"}
	clusterA         = Destination{NodeID: "cluster-AAA", Name: "cluster-AAA"}
	clusterB         = Destination{NodeID: "cluster-BBB", Name: "cluster-BBB"}
	clusterC         = Destination{NodeID: "cluster-CCC", Name: "cluster-CCC"}
	labelKubernetes  = Label{Value: "kubernetes"}
	labelUSWest1     = Label{Value: "us-west-1"}
	labelUSEast1     = Label{Value: "us-east-1"}

	registry *Registry
)

func setup() error {
	confFileData, err := ioutil.ReadFile("_testdata/infra.yaml")
	if err != nil {
		return err
	}

	db, err = NewDB("file::memory:")
	if err != nil {
		return err
	}

	err = db.Create(&fakeOktaProvider).Error
	if err != nil {
		return err
	}

	err = db.Create(&adminUser).Error
	if err != nil {
		return err
	}

	err = db.Model(&fakeOktaProvider).Association("Users").Append(&adminUser)
	if err != nil {
		return err
	}

	err = db.Create(&standardUser).Error
	if err != nil {
		return err
	}

	err = db.Model(&fakeOktaProvider).Association("Users").Append(&standardUser)
	if err != nil {
		return err
	}

	err = db.Create(&iosDevUser).Error
	if err != nil {
		return err
	}

	err = db.Model(&fakeOktaProvider).Association("Users").Append(&iosDevUser)
	if err != nil {
		return err
	}

	err = db.Create(&clusterA).Error
	if err != nil {
		return err
	}

	err = db.Create(&clusterB).Error
	if err != nil {
		return err
	}

	err = db.Create(&clusterC).Error
	if err != nil {
		return err
	}

	err = db.Create(&labelKubernetes).Error
	if err != nil {
		return err
	}

	err = db.Create(&labelUSWest1).Error
	if err != nil {
		return err
	}

	err = db.Model(&clusterA).Association("Labels").Replace([]Label{labelKubernetes})
	if err != nil {
		return err
	}

	err = db.Model(&clusterB).Association("Labels").Replace([]Label{labelKubernetes, labelUSWest1})
	if err != nil {
		return err
	}

	err = db.Model(&clusterC).Association("Labels").Replace([]Label{labelKubernetes, labelUSEast1})
	if err != nil {
		return err
	}

	err = db.Create(&iosDevGroup).Error
	if err != nil {
		return err
	}

	iosDevGroupUsers := []User{iosDevUser, standardUser}

	err = db.Model(&iosDevGroup).Association("Users").Replace(iosDevGroupUsers)
	if err != nil {
		return err
	}

	err = db.Create(&macAdminGroup).Error
	if err != nil {
		return err
	}

	registry = &Registry{
		db: db,
	}

	return registry.importConfig(confFileData)
}

func TestMain(m *testing.M) {
	err := setup()
	if err != nil {
		fmt.Println("Could not initialize test data: ", err)
		os.Exit(1)
	}

	code := m.Run()
	os.Exit(code)
}

func TestRolesForExistingUsersAndDestinationsAreCreated(t *testing.T) {
	isAdminAdminA, err := containsUserRoleForDestination(db, adminUser, clusterA.Id, "admin")
	if err != nil {
		t.Error(err)
	}

	assert.True(t, isAdminAdminA, "admin@example.com should have the admin role in cluster-AAA")

	isAdminAdminB, err := containsUserRoleForDestination(db, adminUser, clusterB.Id, "admin")
	if err != nil {
		t.Error(err)
	}

	assert.True(t, isAdminAdminB, "admin@example.com should have the admin role in cluster-BBB")

	isAdminAdminC, err := containsUserRoleForDestination(db, adminUser, clusterC.Id, "admin")
	if err != nil {
		t.Error(err)
	}

	assert.True(t, isAdminAdminC, "admin@example.com should have the admin role in cluster-CCC")

	isViewAdminB, err := containsUserRoleForDestination(db, adminUser, clusterB.Id, "view")
	if err != nil {
		t.Error(err)
	}

	assert.False(t, isViewAdminB, "admin@example.com should not have the view role in cluster-BBB")

	isViewAdminC, err := containsUserRoleForDestination(db, adminUser, clusterC.Id, "view")
	if err != nil {
		t.Error(err)
	}

	assert.False(t, isViewAdminC, "admin@example.com should not have the view role in cluster-CCC")

	unknownUser := User{Id: "0", Email: "unknown@example.com"}

	isUnknownUserGrantedRole, err := containsUserRoleForDestination(db, unknownUser, clusterA.Id, "writer")
	if err != nil {
		t.Error(err)
	}

	assert.False(t, isUnknownUserGrantedRole, "unknown user should not have roles assigned")
}

func TestImportRolesForUnknownDestinationsAreIgnored(t *testing.T) {
	var roles []Role
	if err := db.Find(&roles).Error; err != nil {
		t.Fatal(err)
	}

	for _, role := range roles {
		var dest Destination
		if err := db.Where(&Destination{Id: role.DestinationId}).First(&dest).Error; err != nil {
			t.Errorf("Created role for destination which does not exist: " + role.DestinationId)
		}
	}
}

func TestImportRolesNoMatchingLabels(t *testing.T) {
	var role Role
	if err := db.First(&role, &Role{Name: "view"}).Error; err == nil {
		t.Fatal("should not find role view")
	}
}

func TestImportRemovesUnusedRoles(t *testing.T) {
	var unused Role
	err := db.Where(&Role{Name: notInConfigRole.Name}).First(&unused).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestExistingProviderIsOverridden(t *testing.T) {
	// this provider comes second in the config so it will override the one before it
	var importedOkta Provider
	if err := db.Where(&Provider{Kind: ProviderKindOkta}).First(&importedOkta).Error; err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, fakeOktaProvider.Domain, importedOkta.Domain)
}

func TestClusterRolesAreAppliedToGroup(t *testing.T) {
	var group Group
	if err := db.Preload("Roles").Where(&Group{Name: "ios-developers"}).First(&group).Error; err != nil {
		t.Errorf("Could not find ios-developers group")
	}

	roles := make(map[string]bool)
	for _, role := range group.Roles {
		roles[role.Name] = true
	}

	assert.True(t, roles["writer"])
}

func TestRolesAreAppliedToGroup(t *testing.T) {
	var group Group
	if err := db.Preload("Roles").Where(&Group{Name: "ios-developers"}).First(&group).Error; err != nil {
		t.Errorf("Could not find ios-developers group")
	}

	roles := make(map[string]bool)
	for _, role := range group.Roles {
		roles[role.Name] = true
	}

	assert.True(t, roles["pod-create"])
}

func TestGroupClusterRolesAreAppliedWithNamespaces(t *testing.T) {
	var group Group
	if err := db.Preload("Roles").Where(&Group{Name: "ios-developers"}).First(&group).Error; err != nil {
		t.Errorf("Could not find ios-developers group")
	}

	foundAuditInfraHQ := false

	for _, role := range group.Roles {
		if role.Name == "audit" && role.Namespace == "infrahq" {
			foundAuditInfraHQ = true
		}
	}

	assert.True(t, foundAuditInfraHQ)

	foundAuditDevelopment := false

	for _, role := range group.Roles {
		if role.Name == "audit" && role.Namespace == "development" {
			foundAuditDevelopment = true
		}
	}

	assert.True(t, foundAuditDevelopment)
}

func TestClusterRolesAreAppliedToUser(t *testing.T) {
	var user User
	if err := db.Preload("Roles").Where(&User{Email: "admin@example.com"}).First(&user).Error; err != nil {
		t.Errorf("Could not find ios-developers group")
	}

	roles := make(map[string]bool)
	for _, role := range user.Roles {
		roles[role.Name] = true
	}

	assert.True(t, roles["admin"])
}

func TestRolesAreAppliedToUser(t *testing.T) {
	var user User
	if err := db.Preload("Roles").Where(&User{Email: "admin@example.com"}).First(&user).Error; err != nil {
		t.Errorf("Could not find ios-developers group")
	}

	roles := make(map[string]bool)
	for _, role := range user.Roles {
		roles[role.Name] = true
	}

	assert.True(t, roles["pod-create"])
}

func TestClusterRolesAreAppliedWithNamespacesToUsers(t *testing.T) {
	var user User
	if err := db.Preload("Roles").Where(&User{Email: "admin@example.com"}).First(&user).Error; err != nil {
		t.Errorf("Could not find ios-developers group")
	}

	foundAuditInfraHQ := false

	for _, role := range user.Roles {
		if role.Name == "audit" && role.Namespace == "infrahq" {
			foundAuditInfraHQ = true
		}
	}

	assert.True(t, foundAuditInfraHQ)

	foundAuditDevelopment := false

	for _, role := range user.Roles {
		if role.Name == "audit" && role.Namespace == "development" {
			foundAuditDevelopment = true
		}
	}

	assert.True(t, foundAuditDevelopment)
}

func TestCleanupDomain(t *testing.T) {
	p := ConfigProvider{Domain: "dev123123-admin.okta.com "}
	p.cleanupDomain()

	expected := "dev123123.okta.com"
	require.Equal(t, expected, p.Domain)
}

func containsUserRoleForDestination(db *gorm.DB, user User, destinationId string, roleName string) (bool, error) {
	var roles []Role

	err := db.Preload("Destination").Preload("Groups").Preload("Users").Find(&roles, &Role{Name: roleName, DestinationId: destinationId}).Error
	if err != nil {
		return false, err
	}

	// check direct role-user relations
	for _, role := range roles {
		for _, roleU := range role.Users {
			if roleU.Email == user.Email {
				return true, nil
			}
		}
	}

	// check user groups-roles
	var groups []Group
	if err := db.Model(&user).Association("Groups").Find(&groups); err != nil {
		return false, err
	}

	for i := range groups {
		var groupRoles []Role
		if err := db.Model(&groups[i]).Association("Roles").Find(&groupRoles, &Role{Name: roleName, DestinationId: destinationId}); err != nil {
			return false, err
		}

		if len(groupRoles) > 0 {
			return true, nil
		}
	}

	return false, nil
}

func TestSecretsLoadedOkay(t *testing.T) {
	foo, err := registry.secrets["plaintext"].GetSecret("foo")
	require.NoError(t, err)
	require.Equal(t, "foo", string(foo))

	var importedOkta Provider
	err = db.Where(&Provider{Kind: ProviderKindOkta}).First(&importedOkta).Error
	require.NoError(t, err)

	// simple manual secret reader
	parts := strings.Split(importedOkta.ClientID, ":")
	secretKind := parts[0]

	secretProvider, ok := registry.secrets[secretKind]
	require.True(t, ok)

	secret, err := secretProvider.GetSecret(parts[1])
	require.NoError(t, err)

	require.Equal(t, "0oapn0qwiQPiMIyR35d6", string(secret))
}

func TestFirstNamespaceThenNoNamespace(t *testing.T) {
	config1 := `
providers:
  - kind: okta
    domain: https://test.example.com
    clientID: plaintext:0oapn0qwiQPiMIyR35d6
    clientSecret: kubernetes:okta-secrets/clientSecret
    apiToken: kubernetes:okta-secrets/apiToken
groups:
  - name: ios-developers
    provider: okta
    roles:
      - name: cluster-admin
        kind: cluster-role
        destinations:
          - name: cluster-AAA
            namespaces:
              - infrahq
`
	config2 := `
providers:
  - kind: okta
    domain: https://test.example.com
    clientID: plaintext:0oapn0qwiQPiMIyR35d6
    clientSecret: kubernetes:okta-secrets/clientSecret
    apiToken: kubernetes:okta-secrets/apiToken
groups:
  - name: ios-developers
    provider: okta
    roles:
      - name: cluster-admin
        kind: cluster-role
        destinations:
          - name: cluster-AAA
`

	if err := registry.importConfig([]byte(config1)); err != nil {
		t.Fatal(err)
	}

	var role1 Role
	if err := db.First(&role1, &Role{Name: "cluster-admin", DestinationId: clusterA.Id}).Error; err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "infrahq", role1.Namespace)

	if err := registry.importConfig([]byte(config2)); err != nil {
		t.Fatal(err)
	}

	var role2 Role
	if err := db.First(&role2, &Role{Name: "cluster-admin", DestinationId: clusterA.Id}).Error; err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "", role2.Namespace)
}

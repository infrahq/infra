package registry

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var db *gorm.DB

var (
	fakeOktaSource = Source{Type: "okta", Domain: "test.example.com", ClientSecret: "okta-secrets/client-secret", ApiToken: "okta-secrets/api-token"}
	adminUser      = User{Email: "admin@example.com"}
	standardUser   = User{Email: "user@example.com"}
	iosDevUser     = User{Email: "woz@example.com"}
	clusterA       = Destination{Name: "cluster-AAA"}
	clusterB       = Destination{Name: "cluster-BBB"}
	clusterC       = Destination{Name: "cluster-CCC"}
)

func setup() error {
	confFile, err := ioutil.ReadFile("_testdata/infra.yaml")
	if err != nil {
		return err
	}
	db, err = NewDB("file::memory:")
	if err != nil {
		return err
	}

	err = db.Create(&fakeOktaSource).Error
	if err != nil {
		return err
	}

	err = db.Create(&adminUser).Error
	if err != nil {
		return err
	}

	err = db.Model(&fakeOktaSource).Association("Users").Append(&adminUser)
	if err != nil {
		return err
	}

	err = db.Create(&standardUser).Error
	if err != nil {
		return err
	}

	err = db.Model(&fakeOktaSource).Association("Users").Append(&standardUser)
	if err != nil {
		return err
	}

	err = db.Create(&iosDevUser).Error
	if err != nil {
		return err
	}

	err = db.Model(&fakeOktaSource).Association("Users").Append(&iosDevUser)
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

	return ImportConfig(db, confFile)
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

func TestImportCurrentValidConfig(t *testing.T) {
	confFile, err := ioutil.ReadFile("_testdata/infra.yaml")
	if err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, ImportConfig(db, confFile))
}

func TestGroupsForExistingSourcesAreCreated(t *testing.T) {
	var groups []Group
	db.Preload("Roles").Find(&groups)
	assert.Equal(t, 4, len(groups), "Exactly four groups should be created from the test config, the other group has an invalid source")

	groupNames := make(map[string]Group)
	for _, g := range groups {
		groupNames[g.Name] = g
	}
	assert.NotNil(t, groupNames["ios-developers"])
	assert.NotNil(t, groupNames["mac-admins"])
	assert.NotNil(t, groupNames["heroes"])
	assert.NotNil(t, groupNames["villains"])

	iosDevRoleDests := make(map[string]bool)
	for _, iosGroupRole := range groupNames["ios-developers"].Roles {
		iosDevRoleDests[iosGroupRole.DestinationId] = true
	}
	assert.True(t, iosDevRoleDests[clusterA.Id])

	assert.Len(t, groupNames["mac-admins"].Roles, 1)
	assert.Contains(t, groupNames["mac-admins"].Roles[0].DestinationId, clusterB.Id)
}

func TestGroupsForUnknownSourcesAreNotCreated(t *testing.T) {
	var groups []Group
	db.Find(&groups)
	assert.Equal(t, 4, len(groups), "Exactly four groups should be created from the test config, the other group has an invalid source")
	group1 := groups[0]
	group2 := groups[1]

	assert.NotEqual(t, "unknown", group1.Name, "A group was made for a source that does not exist")
	assert.NotEqual(t, "unknown", group2.Name, "A group was made for a source that does not exist")
}

func TestUsersForExistingUsersAndDestinationsAreCreated(t *testing.T) {
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

	isStandardWriterA, err := containsUserRoleForDestination(db, standardUser, clusterA.Id, "writer")
	if err != nil {
		t.Error(err)
	}
	assert.True(t, isStandardWriterA, "user@example.com should have the writer role in cluster-AAA")

	isStandardReaderA, err := containsUserRoleForDestination(db, standardUser, clusterA.Id, "reader")
	if err != nil {
		t.Error(err)
	}
	assert.True(t, isStandardReaderA, "user@example.com should have the reader role in cluster-AAA")

	unkownUser := User{Id: "0", Email: "unknown@example.com"}
	isUnknownUserGrantedRole, err := containsUserRoleForDestination(db, unkownUser, clusterA.Id, "writer")
	if err != nil {
		t.Error(err)
	}
	assert.False(t, isUnknownUserGrantedRole, "unknown user should not have roles assigned")
}

func TestImportRolesForUnknownDestinationsAreIgnored(t *testing.T) {
	var roles []Role
	err := db.Find(&roles).Error
	if err != nil {
		t.Fatal(err)
	}

	for _, role := range roles {
		var dest Destination
		err := db.Where(&Destination{Id: role.DestinationId}).First(&dest).Error
		if err != nil {
			t.Errorf("Created role for destination which does not exist: " + role.DestinationId)
		}
	}
}

func TestExistingSourceIsOverridden(t *testing.T) {
	// this source comes second in the config so it will override the one before it
	var importedOkta Source
	err := db.Where(&Source{Type: SOURCE_TYPE_OKTA}).First(&importedOkta).Error
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, fakeOktaSource.Domain, importedOkta.Domain)
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
	db.Model(&user).Association("Groups").Find(&groups)
	for _, g := range groups {
		var groupRoles []Role
		err := db.Model(&g).Association("Roles").Find(&groupRoles, &Role{Name: roleName, DestinationId: destinationId})
		if err != nil {
			return false, err
		}
		if len(groupRoles) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func TestClusterRolesAreAppliedToGroup(t *testing.T) {
	var group Group
	err := db.Preload("Roles").Where(&Group{Name: "ios-developers"}).First(&group).Error
	if err != nil {
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
	err := db.Preload("Roles").Where(&Group{Name: "ios-developers"}).First(&group).Error
	if err != nil {
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
	err := db.Preload("Roles").Where(&Group{Name: "ios-developers"}).First(&group).Error
	if err != nil {
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
	err := db.Preload("Roles").Where(&User{Email: "admin@example.com"}).First(&user).Error
	if err != nil {
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
	err := db.Preload("Roles").Where(&User{Email: "admin@example.com"}).First(&user).Error
	if err != nil {
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
	err := db.Preload("Roles").Where(&User{Email: "admin@example.com"}).First(&user).Error
	if err != nil {
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
	s := ConfigSource{Domain: "dev123123-admin.okta.com "}
	s.cleanupDomain()
	expected := "dev123123.okta.com"
	require.Equal(t, expected, s.Domain)
}

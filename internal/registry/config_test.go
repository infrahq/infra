package registry

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

var db *gorm.DB

var fakeOktaSource = Source{Type: "okta", Domain: "test.example.com", ClientSecret: "okta-secrets/client-secret", ApiToken: "okta-secrets/api-token"}
var adminUser = User{Email: "admin@example.com"}
var standardUser = User{Email: "user@example.com"}
var iosDevUser = User{Email: "woz@example.com"}
var clusterA = Destination{Name: "cluster-AAA"}
var clusterB = Destination{Name: "cluster-BBB"}

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
	err = db.Create(&standardUser).Error
	if err != nil {
		return err
	}
	err = db.Create(&iosDevUser).Error
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
	db.Find(&groups)
	assert.Equal(t, 2, len(groups), "Only two groups should be created from the test config, the other group has an invalid source")
	group1 := groups[0]
	group2 := groups[1]

	var sources []Source
	db.Model(&group1).Association("Sources").Find(&sources)
	assert.Equal(t, 1, len(sources), "Groups in the test config should only have the source \"okta\"")
	db.Model(&group2).Association("Sources").Find(&sources)
	assert.Equal(t, 1, len(sources), "Groups in the test config should only have the source \"okta\"")

	var roles1 []Role
	db.Model(&group1).Association("Roles").Find(&roles1)
	assert.Equal(t, 1, len(roles1), "The groups in the test config should have only one role")
	var roles2 []Role
	db.Model(&group2).Association("Roles").Find(&roles2)
	assert.Equal(t, 1, len(roles2), "The groups in the test config should have only one role")

	var destinationRoles = map[string]string{
		clusterA.Id: "writer",
		clusterB.Id: "writer",
	}
	var roles []Role
	roles = append(roles, roles1...)
	roles = append(roles, roles2...)
	// check all of our expected roles exist
	for _, role := range roles {
		if destinationRoles[role.DestinationId] == "" {
			t.Error("Unexpected role loaded from test config", role)
		}
		delete(destinationRoles, role.DestinationId)
	}
	assert.Empty(t, destinationRoles, "Not all roles expected to be loaded from test config were seen")
}

func TestGroupsForUnknownSourcesAreNotCreated(t *testing.T) {
	var groups []Group
	db.Find(&groups)
	assert.Equal(t, 2, len(groups), "Only two groups should be created from the test config, the other group has an invalid source")
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

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
var adminUser = User{Email: "admin@example.com"}
var standardUser = User{Email: "user@example.com"}
var clusterA = &Destination{Name: "cluster-AAA"}
var clusterB = &Destination{Name: "cluster-BBB"}

func setup() error {
	confFile, err := ioutil.ReadFile("_testdata/infra.yaml")
	if err != nil {
		return err
	}
	db, err = NewDB("file::memory:")
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
	err = db.Create(&clusterA).Error
	if err != nil {
		return err
	}
	err = db.Create(&clusterB).Error
	if err != nil {
		return err
	}

	ImportConfig(db, confFile)
	return nil
}

func TestMain(m *testing.M) {
	err := setup()
	if err != nil {
		fmt.Println("Could not initialize test data")
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

func TestRolesForExistingUsersAndDestinationsAreCreated(t *testing.T) {
	assert.True(t, containsUserRoleForDestination(db, adminUser, clusterA.Id, "admin"), "admin@example.com should have the admin role in cluster-AAA")
	assert.True(t, containsUserRoleForDestination(db, adminUser, clusterB.Id, "admin"), "admin@example.com should have the admin role in cluster-BBB")
	assert.True(t, containsUserRoleForDestination(db, standardUser, clusterA.Id, "writer"), "user@example.com should have the writer role in cluster-AAA")
	assert.True(t, containsUserRoleForDestination(db, standardUser, clusterB.Id, "reader"), "user@example.com should have the reader role in cluster-BBB")

	unkownUser := User{Id: "0", Email: "unknown@example.com"}
	assert.False(t, containsUserRoleForDestination(db, unkownUser, clusterA.Id, "writer"), "unknown user should not have roles assigned")
}

func TestUsersForExistingUsersAndDestinationsAreCreated(t *testing.T) {
	assert.True(t, containsUserRoleForDestination(db, adminUser, clusterA.Id, "admin"), "admin@example.com should have the admin role in cluster-AAA")
	assert.True(t, containsUserRoleForDestination(db, adminUser, clusterB.Id, "admin"), "admin@example.com should have the admin role in cluster-BBB")
	assert.True(t, containsUserRoleForDestination(db, standardUser, clusterA.Id, "writer"), "user@example.com should have the writer role in cluster-AAA")
	assert.True(t, containsUserRoleForDestination(db, standardUser, clusterB.Id, "reader"), "user@example.com should have the reader role in cluster-BBB")
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

func containsUserRoleForDestination(db *gorm.DB, user User, destinationId string, roleName string) bool {
	var roles []Role
	db.Model(&user).Association("Roles").Find(&roles)
	for _, role := range roles {
		if role.DestinationId == destinationId && role.Name == roleName {
			return true
		}
	}
	return false
}

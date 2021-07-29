package registry

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestImportCurrentValidConfig(t *testing.T) {
	conf, err := ioutil.ReadFile("_testdata/infra.yaml")
	if err != nil {
		t.Fatal(err)
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	assert.NoError(t, ImportConfig(db, conf))
}

func TestImportRolesForExistingUsersAndDestinations(t *testing.T) {
	confFile, err := ioutil.ReadFile("_testdata/infra.yaml")
	if err != nil {
		t.Fatal(err)
	}
	config := NewConfig()
	err = yaml.Unmarshal(confFile, &config)
	if err != nil {
		t.Fatal(err)
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	// Create the users and destinations that exist in the sample infra.yaml
	adminUser := User{Email: "admin@example.com"}
	err = db.Create(&adminUser).Error
	if err != nil {
		t.Fatal(err)
	}
	standardUser := User{Email: "user@example.com"}
	err = db.Create(&standardUser).Error
	if err != nil {
		t.Fatal(err)
	}
	clusterA := &Destination{Name: "cluster-AAA"}
	err = db.Create(&clusterA).Error
	if err != nil {
		t.Fatal(err)
	}
	clusterB := &Destination{Name: "cluster-BBB"}
	err = db.Create(&clusterB).Error
	if err != nil {
		t.Fatal(err)
	}

	ImportUserMappings(db, config.Users)

	var roles []Role
	err = db.Preload("User").Preload("Destination").Find(&roles).Error
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, containsUserRoleForDestination(roles, adminUser.Id, clusterA.Id, "admin"), "admin@example.com should have the admin role in cluster-AAA")
	assert.True(t, containsUserRoleForDestination(roles, adminUser.Id, clusterB.Id, "admin"), "admin@example.com should have the admin role in cluster-BBB")
	assert.True(t, containsUserRoleForDestination(roles, standardUser.Id, clusterA.Id, "writer"), "user@example.com should have the writer role in cluster-AAA")
	assert.True(t, containsUserRoleForDestination(roles, standardUser.Id, clusterB.Id, "reader"), "user@example.com should have the reader role in cluster-BBB")
}

func TestImportRolesForUnknownUsers(t *testing.T) {
	confFile, err := ioutil.ReadFile("_testdata/infra.yaml")
	if err != nil {
		t.Fatal(err)
	}
	config := NewConfig()
	err = yaml.Unmarshal(confFile, &config)
	if err != nil {
		t.Fatal(err)
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	adminUser := User{Email: "admin@example.com"}
	err = db.Create(&adminUser).Error
	if err != nil {
		t.Fatal(err)
	}
	standardUser := User{Email: "user@example.com"}
	err = db.Create(&standardUser).Error
	if err != nil {
		t.Fatal(err)
	}

	// users exist, but there are no destinations

	ImportUserMappings(db, config.Users)
	var roles []Role
	err = db.Preload("User").Preload("Destination").Find(&roles).Error
	if err != nil {
		t.Fatal(err)
	}
	assert.Empty(t, roles, "roles mappings were created when no destinations exist")
}

func TestImportRolesForUnknownDestinations(t *testing.T) {
	confFile, err := ioutil.ReadFile("_testdata/infra.yaml")
	if err != nil {
		t.Fatal(err)
	}
	config := NewConfig()
	err = yaml.Unmarshal(confFile, &config)
	if err != nil {
		t.Fatal(err)
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	// no users created in this database

	ImportUserMappings(db, config.Users)
	var roles []Role
	err = db.Preload("User").Preload("Destination").Find(&roles).Error
	if err != nil {
		t.Fatal(err)
	}
	assert.Empty(t, roles, "roles mappings were created when no users exist")
}

func containsUserRoleForDestination(roles []Role, userId string, destinationId string, roleName string) bool {
	for _, role := range roles {
		if role.UserId == userId && role.DestinationId == destinationId && role.Role == roleName {
			return true
		}
	}
	return false
}

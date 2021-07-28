package registry

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
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

// func TestImportUsersThatDoNotExist(t *testing.T) {
// 	confFile, err := ioutil.ReadFile("_testdata/infra.yaml")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	config := NewConfig()
// 	err = yaml.Unmarshal(confFile, &config)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	db, err := NewDB("file::memory:")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	ImportUserMappings(db, config.Users)
// 	for user, userMapping := range config.Users {
// 		for roleName, role := range userMapping.Roles {
// 			var permission Permission
// 			err = db.Where(&permission, &Permission{Role: roleName, Kind: role.Kind, UserId: user.Id, DestinationId: destination.Id, FromConfig: true}).Error
// 			if err != nil {
// 				return nil, err
// 			}
// 		}
// 	}
// }

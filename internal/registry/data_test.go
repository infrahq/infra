package registry

import (
	"testing"

	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/registry/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	rest "k8s.io/client-go/rest"
)

func TestSyncGroupsClearsOnlySource(t *testing.T) {
	// mocks no groups being present at the source
	mockGroups := make(map[string][]string)
	testOkta := new(mocks.Okta)
	testOkta.On("Groups", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockGroups, nil)

	testSecretReader := NewMockSecretReader()
	testConfig := &rest.Config{
		Host: "https://localhost",
	}
	testK8s := &kubernetes.Kubernetes{Config: testConfig, SecretReader: testSecretReader}

	fakeOktaSource.SyncGroups(db, testK8s, testOkta)
	// the total amount of groups for all sources should not change, just the users on the groups
	var groups []Group
	if err := db.Preload("Users").Find(&groups).Error; err != nil {
		t.Fatal(err)
	}
	assert.Len(t, groups, 4)
	for _, group := range groups {
		var src Source
		if err := db.Where(&Source{Id: group.SourceId}).First(&src).Error; err != nil {
			t.Fatal(err)
		}
		switch src.Type {
		case SOURCE_TYPE_INFRA:
			assert.Greater(t, len(group.Users), 0)
		case SOURCE_TYPE_OKTA:
			// these groups are part of the okta source and should be cleared
			assert.Len(t, group.Users, 0)
		}
	}
}

func TestSyncGroupsFromOktaIgnoresUnknownUsers(t *testing.T) {
	mockGroups := make(map[string][]string)
	mockGroups["heroes"] = []string{"goku@example.com", "woz@example.com"}
	testOkta := new(mocks.Okta)
	testOkta.On("Groups", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockGroups, nil)

	testSecretReader := NewMockSecretReader()
	testConfig := &rest.Config{
		Host: "https://localhost",
	}
	testK8s := &kubernetes.Kubernetes{Config: testConfig, SecretReader: testSecretReader}

	fakeOktaSource.SyncGroups(db, testK8s, testOkta)

	var heroGroup Group
	if err := db.Preload("Users").Where(&Group{Name: "heroes", SourceId: fakeOktaSource.Id}).First(&heroGroup).Error; err != nil {
		t.Fatal(err)
	}
	assert.Len(t, heroGroup.Users, 1)
	assert.Equal(t, heroGroup.Users[0].Email, "woz@example.com")
}

func TestSyncGroupsFromOktaRepopulatesEmptyGroups(t *testing.T) {
	mockGroups := make(map[string][]string)
	mockGroups["heroes"] = []string{"woz@example.com"}
	testOkta := new(mocks.Okta)
	testOkta.On("Groups", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockGroups, nil)

	testSecretReader := NewMockSecretReader()
	testConfig := &rest.Config{
		Host: "https://localhost",
	}
	testK8s := &kubernetes.Kubernetes{Config: testConfig, SecretReader: testSecretReader}

	fakeOktaSource.SyncGroups(db, testK8s, testOkta)

	var heroGroup Group
	if err := db.Preload("Users").Where(&Group{Name: "heroes", SourceId: fakeOktaSource.Id}).First(&heroGroup).Error; err != nil {
		t.Fatal(err)
	}
	assert.Len(t, heroGroup.Users, 1)
	assert.Equal(t, heroGroup.Users[0].Email, "woz@example.com")

	var villainGroup Group
	if err := db.Preload("Users").Where(&Group{Name: "villains", SourceId: fakeOktaSource.Id}).First(&villainGroup).Error; err != nil {
		t.Fatal(err)
	}
	assert.Len(t, villainGroup.Users, 0)

	mockGroups["villains"] = []string{"user@example.com"}
	fakeOktaSource.SyncGroups(db, testK8s, testOkta)

	if err := db.Preload("Users").Where(&Group{Name: "heroes", SourceId: fakeOktaSource.Id}).First(&heroGroup).Error; err != nil {
		t.Fatal(err)
	}
	assert.Len(t, heroGroup.Users, 1)
	assert.Equal(t, heroGroup.Users[0].Email, "woz@example.com")

	if err := db.Preload("Users").Where(&Group{Name: "villains", SourceId: fakeOktaSource.Id}).First(&villainGroup).Error; err != nil {
		t.Fatal(err)
	}
	assert.Len(t, villainGroup.Users, 1)
}

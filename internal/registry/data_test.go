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

	if err := fakeOktaSource.SyncGroups(db, testK8s, testOkta); err != nil {
		t.Fatal(err)
	}

	var groups []Group
	if err := db.Preload("Users").Find(&groups).Error; err != nil {
		t.Fatal(err)
	}

	assert.Len(t, groups, 0)
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

	if err := fakeOktaSource.SyncGroups(db, testK8s, testOkta); err != nil {
		t.Fatal(err)
	}

	var heroGroup Group
	if err := db.Preload("Users").Where(&Group{Name: "heroes", SourceId: fakeOktaSource.Id}).First(&heroGroup).Error; err != nil {
		t.Fatal(err)
	}

	assert.Len(t, heroGroup.Users, 1)
	assert.Equal(t, heroGroup.Users[0].Email, "woz@example.com")
}

func TestSyncGroupsFromOktaRecreatesGroups(t *testing.T) {
	mockGroups := make(map[string][]string)
	mockGroups["heroes"] = []string{"woz@example.com"}
	testOkta := new(mocks.Okta)
	testOkta.On("Groups", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockGroups, nil)

	testSecretReader := NewMockSecretReader()
	testConfig := &rest.Config{
		Host: "https://localhost",
	}
	testK8s := &kubernetes.Kubernetes{Config: testConfig, SecretReader: testSecretReader}

	if err := fakeOktaSource.SyncGroups(db, testK8s, testOkta); err != nil {
		t.Fatal(err)
	}

	var heroGroup Group
	if err := db.Preload("Users").Where(&Group{Name: "heroes", SourceId: fakeOktaSource.Id}).First(&heroGroup).Error; err != nil {
		t.Fatal(err)
	}

	assert.Len(t, heroGroup.Users, 1)
	assert.Equal(t, heroGroup.Users[0].Email, "woz@example.com")

	mockGroups["villains"] = []string{"user@example.com"}

	if err := fakeOktaSource.SyncGroups(db, testK8s, testOkta); err != nil {
		t.Fatal(err)
	}

	if err := db.Preload("Users").Where(&Group{Name: "heroes", SourceId: fakeOktaSource.Id}).First(&heroGroup).Error; err != nil {
		t.Fatal(err)
	}

	assert.Len(t, heroGroup.Users, 1)
	assert.Equal(t, heroGroup.Users[0].Email, "woz@example.com")

	var villainGroup Group
	if err := db.Preload("Users").Where(&Group{Name: "villains", SourceId: fakeOktaSource.Id}).First(&villainGroup).Error; err != nil {
		t.Fatal(err)
	}

	assert.Len(t, villainGroup.Users, 1)
}

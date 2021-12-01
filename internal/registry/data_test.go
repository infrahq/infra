package registry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/data"
	"github.com/infrahq/infra/internal/registry/mocks"
)

var (
	providerOkta *data.Provider

	userBond   *data.User
	userBourne *data.User

	groupEveryone  *data.Group
	groupEngineers *data.Group

	destinationAAA *data.Destination
	destinationBBB *data.Destination
	destinationCCC *data.Destination

	labelKubernetes = data.Label{Value: "kubernetes"}
	labelUSWest1    = data.Label{Value: "us-west-1"}
	labelUSEast1    = data.Label{Value: "us-east-1"}
)

func setupDB(t *testing.T) *gorm.DB {
	setupLogging(t)

	driver, err := data.NewSQLiteDriver("file::memory:")
	require.NoError(t, err)

	db, err := data.NewDB(driver)
	require.NoError(t, err)

	providerOkta, err = data.CreateProvider(db, &data.Provider{
		Kind:         data.ProviderKindOkta,
		Domain:       "test.okta.com",
		ClientSecret: "supersecret",
		Okta: data.ProviderOkta{
			APIToken: "verysupersecret",
		},
	})
	require.NoError(t, err)

	userBond, err = data.CreateUser(db, &data.User{Email: "jbond@infrahq.com"})
	require.NoError(t, err)

	userBourne, err = data.CreateUser(db, &data.User{Email: "jbourne@infrahq.com"})
	require.NoError(t, err)

	groupEveryone, err = data.CreateGroup(db, &data.Group{Name: "Everyone"})
	require.NoError(t, err)

	groupEngineers, err = data.CreateGroup(db, &data.Group{Name: "Engineering"})
	require.NoError(t, err)

	err = groupEveryone.BindUsers(db, *userBourne)
	require.NoError(t, err)

	destinationAAA, err = data.CreateDestination(db, &data.Destination{
		Kind:     data.DestinationKindKubernetes,
		Name:     "AAA",
		NodeID:   "AAA",
		Endpoint: "develop.infrahq.com",
		Labels: []data.Label{
			labelKubernetes,
		},
		Kubernetes: data.DestinationKubernetes{
			CA: "myca",
		},
	})
	require.NoError(t, err)

	destinationBBB, err = data.CreateDestination(db, &data.Destination{
		Kind:     data.DestinationKindKubernetes,
		Name:     "BBB",
		NodeID:   "BBB",
		Endpoint: "stage.infrahq.com",
		Labels: []data.Label{
			labelKubernetes,
			labelUSWest1,
		},
		Kubernetes: data.DestinationKubernetes{
			CA: "myotherca",
		},
	})
	require.NoError(t, err)

	destinationCCC, err = data.CreateDestination(db, &data.Destination{
		Kind:     data.DestinationKindKubernetes,
		Name:     "CCC",
		NodeID:   "CCC",
		Endpoint: "production.infrahq.com",
		Labels: []data.Label{
			labelKubernetes,
			labelUSEast1,
		},
		Kubernetes: data.DestinationKubernetes{
			CA: "myotherotherca",
		},
	})
	require.NoError(t, err)

	return db
}

func TestSyncUsers(t *testing.T) {
	db := setupDB(t)

	mockUsers := []string{"jbond@infrahq.com", "jbourne@infrahq.com"}

	testOkta := new(mocks.Okta)
	testOkta.On("Users", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockUsers, nil)

	err := syncUsers(db, mockUsers)
	require.NoError(t, err)

	users, err := data.ListUsers(db, &data.User{})
	require.NoError(t, err)
	require.Len(t, users, 2)
	require.Subset(t, []string{"jbond@infrahq.com", "jbourne@infrahq.com"}, []string{users[0].Email})
	require.Subset(t, []string{"jbond@infrahq.com", "jbourne@infrahq.com"}, []string{users[1].Email})
}

func TestSyncUsersDeleteNonExistentUsers(t *testing.T) {
	db := setupDB(t)

	// mock no users found in provider
	mockUsers := make([]string, 0)

	testOkta := new(mocks.Okta)
	testOkta.On("Users", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockUsers, nil)

	err := syncUsers(db, mockUsers)
	require.NoError(t, err)

	users, err := data.ListUsers(db, &data.User{})
	require.NoError(t, err)
	require.Len(t, users, 0)
}

func TestSyncGroups(t *testing.T) {
	db := setupDB(t)

	mockGroups := make(map[string][]string)
	mockGroups["heroes"] = []string{}

	testOkta := new(mocks.Okta)
	testOkta.On("Groups", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockGroups, nil)

	err := syncGroups(db, mockGroups)
	require.NoError(t, err)

	heroes, err := data.ListGroups(db, &data.Group{})
	require.NoError(t, err)
	require.Len(t, heroes, 1)
	require.Equal(t, "heroes", heroes[0].Name)
}

func TestSyncGroupsDeleteNonExistentGroups(t *testing.T) {
	db := setupDB(t)

	// mock no groups found in provider
	mockGroups := make(map[string][]string)

	testOkta := new(mocks.Okta)
	testOkta.On("Groups", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockGroups, nil)

	err := syncGroups(db, mockGroups)
	require.NoError(t, err)

	groups, err := data.ListGroups(db, &data.Group{})
	require.NoError(t, err)
	require.Len(t, groups, 0)
}

func TestSyncGroupsIgnoresUnknownUsers(t *testing.T) {
	db := setupDB(t)

	mockGroups := make(map[string][]string)
	mockGroups["heroes"] = []string{"jbourne@infrahq.com", "nonexistent@infrahq.com"}

	testOkta := new(mocks.Okta)
	testOkta.On("Groups", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockGroups, nil)

	err := syncGroups(db, mockGroups)
	require.NoError(t, err)

	heroes, err := data.GetGroup(db, &data.Group{Name: "heroes"})
	require.NoError(t, err)
	require.Len(t, heroes.Users, 1)
	require.Equal(t, heroes.Users[0].Email, "jbourne@infrahq.com")
}

func TestSyncGroupsRecreateGroups(t *testing.T) {
	db := setupDB(t)

	mockGroups := make(map[string][]string)
	mockGroups["heroes"] = []string{"jbourne@infrahq.com"}

	testOkta := new(mocks.Okta)
	testOkta.On("Groups", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockGroups, nil)

	err := syncGroups(db, mockGroups)
	require.NoError(t, err)

	heroes, err := data.GetGroup(db, &data.Group{Name: "heroes"})
	require.NoError(t, err)
	require.Len(t, heroes.Users, 1)
	require.Equal(t, heroes.Users[0].Email, "jbourne@infrahq.com")

	mockGroups["villains"] = []string{"jbond@infrahq.com"}

	err = syncGroups(db, mockGroups)
	require.NoError(t, err)

	heroes, err = data.GetGroup(db, &data.Group{Name: "heroes"})
	require.NoError(t, err)
	require.Len(t, heroes.Users, 1)
	require.Equal(t, heroes.Users[0].Email, "jbourne@infrahq.com")

	villains, err := data.GetGroup(db, &data.Group{Name: "villains"})
	require.NoError(t, err)
	require.Len(t, villains.Users, 1)
	require.Equal(t, villains.Users[0].Email, "jbond@infrahq.com")
}

func TestSyncProviders(t *testing.T) {
}

func TestSyncDestinations(t *testing.T) {
	db := setupDB(t)

	syncDestinations(db, time.Hour*1)

	destinations, err := data.ListDestinations(db, &data.Destination{})
	require.NoError(t, err)
	require.Len(t, destinations, 3)
}

func TestSyncDestinationsDeletePastTTL(t *testing.T) {
	db := setupDB(t)

	destinations, err := data.ListDestinations(db, &data.Destination{})
	require.NoError(t, err)
	require.Len(t, destinations, 3)

	// set TTL to zero so all destinations will expire
	syncDestinations(db, time.Second*0)

	destinations, err = data.ListDestinations(db, &data.Destination{})
	require.NoError(t, err)
	require.Len(t, destinations, 0)
}

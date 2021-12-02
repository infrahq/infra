package registry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/mocks"
	"github.com/infrahq/infra/internal/registry/models"
)

var (
	providerOkta *models.Provider

	userBond   *models.User
	userBourne *models.User

	groupEveryone  *models.Group
	groupEngineers *models.Group

	destinationAAA *models.Destination
	destinationBBB *models.Destination
	destinationCCC *models.Destination

	labelKubernetes = models.Label{Value: "kubernetes"}
	labelUSWest1    = models.Label{Value: "us-west-1"}
	labelUSEast1    = models.Label{Value: "us-east-1"}
)

func setupDB(t *testing.T) *gorm.DB {
	setupLogging(t)

	driver, err := data.NewSQLiteDriver("file::memory:")
	require.NoError(t, err)

	db, err := data.NewDB(driver)
	require.NoError(t, err)

	providerOkta, err = data.CreateProvider(db, &models.Provider{
		Kind:         models.ProviderKindOkta,
		Domain:       "test.okta.com",
		ClientSecret: "supersecret",
		Okta: models.ProviderOkta{
			APIToken: "verysupersecret",
		},
	})
	require.NoError(t, err)

	userBond, err = data.CreateUser(db, &models.User{Email: "jbond@infrahq.com"})
	require.NoError(t, err)

	userBourne, err = data.CreateUser(db, &models.User{Email: "jbourne@infrahq.com"})
	require.NoError(t, err)

	groupEveryone, err = data.CreateGroup(db, &models.Group{Name: "Everyone"})
	require.NoError(t, err)

	groupEngineers, err = data.CreateGroup(db, &models.Group{Name: "Engineering"})
	require.NoError(t, err)

	err = data.BindGroupUsers(db, groupEveryone, *userBourne)
	require.NoError(t, err)

	destinationAAA, err = data.CreateDestination(db, &models.Destination{
		Kind:     models.DestinationKindKubernetes,
		Name:     "AAA",
		NodeID:   "AAA",
		Endpoint: "develop.infrahq.com",
		Labels: []models.Label{
			labelKubernetes,
		},
		Kubernetes: models.DestinationKubernetes{
			CA: "myca",
		},
	})
	require.NoError(t, err)

	destinationBBB, err = data.CreateDestination(db, &models.Destination{
		Kind:     models.DestinationKindKubernetes,
		Name:     "BBB",
		NodeID:   "BBB",
		Endpoint: "stage.infrahq.com",
		Labels: []models.Label{
			labelKubernetes,
			labelUSWest1,
		},
		Kubernetes: models.DestinationKubernetes{
			CA: "myotherca",
		},
	})
	require.NoError(t, err)

	destinationCCC, err = data.CreateDestination(db, &models.Destination{
		Kind:     models.DestinationKindKubernetes,
		Name:     "CCC",
		NodeID:   "CCC",
		Endpoint: "production.infrahq.com",
		Labels: []models.Label{
			labelKubernetes,
			labelUSEast1,
		},
		Kubernetes: models.DestinationKubernetes{
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

	users, err := data.ListUsers(db, &models.User{})
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

	users, err := data.ListUsers(db, &models.User{})
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

	heroes, err := data.ListGroups(db, &models.Group{})
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

	groups, err := data.ListGroups(db, &models.Group{})
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

	heroes, err := data.GetGroup(db, &models.Group{Name: "heroes"})
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

	heroes, err := data.GetGroup(db, &models.Group{Name: "heroes"})
	require.NoError(t, err)
	require.Len(t, heroes.Users, 1)
	require.Equal(t, heroes.Users[0].Email, "jbourne@infrahq.com")

	mockGroups["villains"] = []string{"jbond@infrahq.com"}

	err = syncGroups(db, mockGroups)
	require.NoError(t, err)

	heroes, err = data.GetGroup(db, &models.Group{Name: "heroes"})
	require.NoError(t, err)
	require.Len(t, heroes.Users, 1)
	require.Equal(t, heroes.Users[0].Email, "jbourne@infrahq.com")

	villains, err := data.GetGroup(db, &models.Group{Name: "villains"})
	require.NoError(t, err)
	require.Len(t, villains.Users, 1)
	require.Equal(t, villains.Users[0].Email, "jbond@infrahq.com")
}

func TestSyncProviders(t *testing.T) {
}

func TestSyncDestinations(t *testing.T) {
	db := setupDB(t)

	syncDestinations(db, time.Hour*1)

	destinations, err := data.ListDestinations(db, &models.Destination{})
	require.NoError(t, err)
	require.Len(t, destinations, 3)
}

func TestSyncDestinationsDeletePastTTL(t *testing.T) {
	db := setupDB(t)

	destinations, err := data.ListDestinations(db, &models.Destination{})
	require.NoError(t, err)
	require.Len(t, destinations, 3)

	// set TTL to zero so all destinations will expire
	syncDestinations(db, time.Second*0)

	destinations, err = data.ListDestinations(db, &models.Destination{})
	require.NoError(t, err)
	require.Len(t, destinations, 0)
}

package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var (
	destinationDevelop    = Destination{Name: "develop", Kind: "kubernetes", Endpoint: "dev.kubernetes.com", Kubernetes: DestinationKubernetes{CA: "notsosecret"}}
	destinationProduction = Destination{Name: "production", Kind: "kubernetes", Endpoint: "prod.kubernetes.com", Kubernetes: DestinationKubernetes{CA: "supersecret"}}

	labelUSWest1 = Label{Value: "us-west-1"}
	labelUSEast1 = Label{Value: "us-east-1"}
)

func TestDestination(t *testing.T) {
	db := setup(t)

	err := db.Create(&destinationDevelop).Error
	require.NoError(t, err)

	var destination Destination
	err = db.Preload("Kubernetes").First(&destination, &Destination{Kind: "kubernetes"}).Error
	require.NoError(t, err)
	require.Equal(t, DestinationKindKubernetes, destination.Kind)
	require.Equal(t, "notsosecret", destination.Kubernetes.CA)
}

func TestCreateDestinationKubernetes(t *testing.T) {
	db := setup(t)

	destination, err := CreateDestination(db, &destinationDevelop)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, destination.ID)
	require.Equal(t, destinationDevelop.Kind, destination.Kind)
	require.Equal(t, destinationDevelop.Kubernetes.CA, destination.Kubernetes.CA)
}

func createDestinations(t *testing.T, db *gorm.DB, destinations ...Destination) {
	for i := range destinations {
		_, err := CreateDestination(db, &destinations[i])
		require.NoError(t, err)
	}
}

func TestCreateDuplicateDestination(t *testing.T) {
	db := setup(t)
	createDestinations(t, db, destinationDevelop, destinationProduction)

	_, err := CreateDestination(db, &destinationDevelop)
	require.EqualError(t, err, "UNIQUE constraint failed: destinations.id")
}

func TestCreateOrUpdateDestinationCreate(t *testing.T) {
	db := setup(t)

	destination, err := CreateOrUpdateDestination(db, &destinationDevelop, &destinationDevelop)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, destination.ID)
	require.Equal(t, "develop", destination.Name)
	require.Equal(t, "notsosecret", destination.Kubernetes.CA)
}

func TestCreateOrUpdateDestinationUpdate(t *testing.T) {
	db := setup(t)
	createDestinations(t, db, destinationDevelop, destinationProduction)

	destination, err := CreateOrUpdateDestination(db, &Destination{Name: "testing"}, &destinationDevelop)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, destination.ID)
	require.Equal(t, "testing", destination.Name)
	require.Equal(t, "notsosecret", destination.Kubernetes.CA)
}

func TestCreateOrUpdateDestinationUpdateCA(t *testing.T) {
	db := setup(t)
	createDestinations(t, db, destinationDevelop, destinationProduction)

	updateCA := Destination{
		Kind: DestinationKindKubernetes,
		Kubernetes: DestinationKubernetes{
			CA: "updated-ca",
		},
	}

	destination, err := CreateOrUpdateDestination(db, &updateCA, &destinationDevelop)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, destination.ID)

	fromDB, err := GetDestination(db, &Destination{Name: destination.Name})
	require.NoError(t, err)
	require.Equal(t, "develop", fromDB.Name)
	require.Equal(t, "updated-ca", fromDB.Kubernetes.CA)
}

func TestCreateDestinationLabels(t *testing.T) {
	db := setup(t)

	labels := Destination{
		Name: "labeled",
		Kind: DestinationKindKubernetes,
		Labels: []Label{
			labelUSWest1,
		},
	}

	destination, err := CreateDestination(db, &labels)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, destination.ID)
	require.Len(t, destination.Labels, 1)

	fromDB, err := GetDestination(db, &Destination{Name: labels.Name})
	require.NoError(t, err)
	require.Len(t, fromDB.Labels, 1)
	require.Equal(t, "us-west-1", fromDB.Labels[0].Value)
}

func TestCreateDestinationMoreLabels(t *testing.T) {
	db := setup(t)

	labels := Destination{
		Name: "labeled",
		Kind: DestinationKindKubernetes,
		Labels: []Label{
			labelUSWest1,
		},
	}

	destination, err := CreateOrUpdateDestination(db, &labels, &labels)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, destination.ID)
	require.Len(t, destination.Labels, 1)

	labels.Labels = append(labels.Labels, labelUSEast1)

	destination, err = CreateOrUpdateDestination(db, &labels, &labels)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, destination.ID)
	require.Len(t, destination.Labels, 2)

	fromDB, err := GetDestination(db, &Destination{Name: labels.Name})
	require.NoError(t, err)
	require.Len(t, fromDB.Labels, 2)
	require.ElementsMatch(t, []string{"us-west-1", "us-east-1"}, []string{
		fromDB.Labels[0].Value,
		fromDB.Labels[1].Value,
	})
}

func TestCreateDestinationLessLabels(t *testing.T) {
	db := setup(t)

	labels := Destination{
		Name: "labeled",
		Kind: DestinationKindKubernetes,
		Labels: []Label{
			labelUSWest1,
			labelUSEast1,
		},
	}

	destination, err := CreateOrUpdateDestination(db, &labels, &labels)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, destination.ID)
	require.Len(t, destination.Labels, 2)

	labels.Labels = []Label{labelUSWest1}

	destination, err = CreateOrUpdateDestination(db, &labels, &labels)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, destination.ID)
	require.Len(t, destination.Labels, 1)

	fromDB, err := GetDestination(db, &Destination{Name: labels.Name})
	require.NoError(t, err)
	require.Len(t, fromDB.Labels, 1)
	require.Equal(t, "us-west-1", fromDB.Labels[0].Value)
}

func TestGetDestination(t *testing.T) {
	db := setup(t)
	createDestinations(t, db, destinationDevelop, destinationProduction)

	destination, err := GetDestination(db, &Destination{Kind: "kubernetes"})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, destination.ID)
	require.Equal(t, destinationDevelop.Kubernetes.CA, destination.Kubernetes.CA)
}

func TestGetDestinationLabelSelector(t *testing.T) {
	db := setup(t)

	destinationLabels := destinationDevelop
	destinationLabels.Labels = []Label{
		{Value: "us-west-1"},
		{Value: "aws"},
	}

	createDestinations(t, db, destinationLabels)

	destination, err := GetDestination(db, LabelSelector(db, "destination_id", "us-west-1"))
	require.NoError(t, err)
	require.Equal(t, 2, len(destination.Labels))

	_, err = GetDestination(db, LabelSelector(db, "destination_id", "eu-central-1"))
	require.EqualError(t, err, "record not found")
}

func TestGetDestinationLabelSelectorCombo(t *testing.T) {
	db := setup(t)

	destinationLabels := destinationDevelop
	destinationLabels.Labels = []Label{
		{Value: "us-west-1"},
		{Value: "aws"},
	}

	createDestinations(t, db, destinationLabels, destinationProduction)

	destination, err := GetDestination(db, db.Where(
		LabelSelector(db, "destination_id", "us-west-1"),
		&Destination{Name: "develop"},
	))
	require.NoError(t, err)
	require.Equal(t, 2, len(destination.Labels))

	_, err = GetDestination(db, db.Where(
		LabelSelector(db, "destination_id", "us-west-1"),
		&Destination{Name: "production"},
	))
	require.EqualError(t, err, "record not found")
}

func TestListDestinations(t *testing.T) {
	db := setup(t)
	createDestinations(t, db, destinationDevelop, destinationProduction)

	destinations, err := ListDestinations(db, &Destination{})
	require.NoError(t, err)
	require.Equal(t, 2, len(destinations))

	destinations, err = ListDestinations(db, &Destination{Name: "production"})
	require.NoError(t, err)
	require.Equal(t, 1, len(destinations))
}

func TestListDestinationsLabelSelector(t *testing.T) {
	db := setup(t)

	destinationLabels := destinationDevelop
	destinationLabels.Labels = []Label{
		{Value: "us-west-1"},
		{Value: "aws"},
	}

	createDestinations(t, db, destinationLabels)

	destinations, err := ListDestinations(db, LabelSelector(db, "destination_id", "us-west-1"))
	require.NoError(t, err)
	require.Equal(t, 1, len(destinations))
	require.Equal(t, 2, len(destinations[0].Labels))

	destinations, err = ListDestinations(db, LabelSelector(db, "destination_id", "eu-central-1"))
	require.NoError(t, err)
	require.Equal(t, 0, len(destinations))
}

func TestDeleteDestinations(t *testing.T) {
	db := setup(t)
	createDestinations(t, db, destinationDevelop, destinationProduction)

	destination, err := GetDestination(db, &Destination{Name: "develop"})
	require.NoError(t, err)

	err = DeleteDestinations(db, &Destination{Name: "develop"})
	require.NoError(t, err)

	// deleting a destination should remove its associated labels
	labels := make([]Label, 0)
	err = db.Where("destination_id IN (?)", destination.Labels).Find(&labels).Error
	require.NoError(t, err)
	require.Equal(t, 0, len(labels))

	_, err = GetDestination(db, &Destination{Name: "develop"})
	require.EqualError(t, err, "record not found")

	// deleting a nonexistent destination should not fail
	err = DeleteDestinations(db, &Destination{Name: "develop"})
	require.NoError(t, err)

	// deleting a destination should not delete unrelated destinations
	_, err = GetDestination(db, &Destination{Name: "production"})
	require.NoError(t, err)
}

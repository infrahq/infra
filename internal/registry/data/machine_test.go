package data

import (
	"testing"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/stretchr/testify/require"
)

func TestCreateMachineExistingName(t *testing.T) {
	db := setup(t)
	err := CreateMachine(db, &models.Machine{Name: "conflict"})
	require.NoError(t, err)

	err = CreateMachine(db, &models.Machine{Name: "conflict"})
	require.Error(t, err, internal.ErrDuplicate)
}

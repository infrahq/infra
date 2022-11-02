package data

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestDeleteExpiredDeviceFlowAuthRequests(t *testing.T) {
	tx := setupDB(t)
	dfar := &models.DeviceFlowAuthRequest{
		UserCode:   "BCDFGHJK",
		DeviceCode: "abcdefghijklmnopqrstuvwxyz123456789000",
		ExpiresAt:  time.Now().Add(-1),
	}
	err := CreateDeviceFlowAuthRequest(tx, dfar)
	assert.NilError(t, err)

	err = DeleteExpiredDeviceFlowAuthRequests(tx)
	assert.NilError(t, err)

	_, err = GetDeviceFlowAuthRequest(tx, ByID(dfar.ID))
	assert.ErrorContains(t, err, "not found")
}

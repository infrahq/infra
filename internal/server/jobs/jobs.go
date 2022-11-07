package jobs

import (
	"context"

	"github.com/infrahq/infra/internal/server/data"
)

func RemoveOldDeviceFlowRequests(ctx context.Context, tx *data.DB) error {
	return data.DeleteExpiredDeviceFlowAuthRequests(tx)
}

func RemoveExpiredAccessKeys(ctx context.Context, tx *data.DB) error {
	return data.RemoveExpiredAccessKeys(tx)
}

func RemoveExpiredPasswordResetTokens(ctx context.Context, tx *data.DB) error {
	return data.RemoveExpiredPasswordResetTokens(tx)
}

package jobs

import (
	"context"
	"time"

	"github.com/infrahq/infra/internal/server/data"
)

func RemoveOldDeviceFlowRequests(ctx context.Context, tx data.WriteTxn, lastRunAt, currentTime time.Time) error {
	return data.DeleteExpiredDeviceFlowAuthRequests(tx)
}

func RemoveExpiredAccessKeys(ctx context.Context, tx data.WriteTxn, lastRunAt, currentTime time.Time) error {
	return data.RemoveExpiredAccessKeys(tx)
}

func RemoveExpiredPasswordResetTokens(ctx context.Context, tx data.WriteTxn, lastRunAt, currentTime time.Time) error {
	return data.RemoveExpiredPasswordResetTokens(tx)
}

package access

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
)

// Query is used by RunBlockingRequest to perform some operation, generally
// querying the database.
//
// Implementations of Query must follow these rules for correct behaviour:
//
//  1. The query performed by Do must have a consistent view of the database,
//     which generally means using a transaction with isolation level
//     sql.LevelRepeatableRead.
//  2. IsDone must only return true if the first call to Do returns a result
//     that is new to the client.
//  3. Any change to the result of the query must result in a notification
//     to at least one of the channels passed to RunBlockingQuery.
//
// For an optimal query:
//
//  1. IsDone should always return true when there is a new result (returning
//     false in this case is safe, but will prevent the client from seeing an
//     update until there is a notification).
//  2. No notification should be sent to any of the channels passed to
//     RunBlockingQuery when there is no change to the result of the query.
//     Extra notifications are safe, but result in additional API requests.
type Query interface {
	// Do performs the query. It will be called by RunBlocking request at least
	// once.
	Do() error
	// IsDone is called by RunBlockingRequest after the first call to Do. It
	// should return true if the query returns new results. If it returns false
	// RunBlockingRequest will block until it receives a notification.
	IsDone() bool
}

// RunBlockingRequest handles a long-polling blocking request. See Query for more
// details.
func RunBlockingRequest(
	rCtx RequestContext,
	query Query,
	channels ...data.ListenChannelDescriptor,
) error {
	listener, err := data.ListenForNotify(rCtx.Request.Context(), rCtx.DataDB, channels...)
	if err != nil {
		return fmt.Errorf("listen for notify: %w", err)
	}
	defer func() {
		// use a context with a separate deadline so that we still release
		// when the request timeout is reached
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := listener.Release(ctx); err != nil {
			logging.L.Error().Err(err).Msg("failed to release listener conn")
		}
	}()

	if err = query.Do(); err != nil {
		return err
	}

	// The query returned results that are new to the client
	if query.IsDone() {
		return nil
	}

	err = listener.WaitForNotification(rCtx.Request.Context())
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return internal.ErrNotModified
	case err != nil:
		return fmt.Errorf("waiting for notify: %w", err)
	}

	return query.Do()
}

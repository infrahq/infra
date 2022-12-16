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

type updateIndexable interface {
	UpdateIndex() int64
	ItemCount() int
}

func blockingRequest[Result updateIndexable](rCtx RequestContext, listenOpts data.ListenForNotifyOptions, query func() (Result, error), lastUpdateIndex int64) (Result, error) {

	listener, err := data.ListenForNotify(rCtx.Request.Context(), rCtx.DataDB, listenOpts)
	if err != nil {
		// nolint: gocritic
		return *new(Result), fmt.Errorf("listen for notify: %w", err)
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

	result, err := query()
	if err != nil {
		return result, err
	}

	// The query returned results that are new to the client
	if result.ItemCount() > 0 && result.UpdateIndex() > lastUpdateIndex {
		return result, nil
	}

	err = listener.WaitForNotification(rCtx.Request.Context())
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		logging.Debugf("listener wait deadline exceeded")
		return result, internal.ErrNotModified
	case err != nil:
		logging.Debugf("error waiting for notify: %s", err)
		return result, fmt.Errorf("waiting for notify: %w", err)
	}

	logging.Debugf("listener woke")

	result, err = query()
	if err != nil {
		return result, err
	}

	// TODO: check if the maxIndex > lastUpdateIndex, and start waiting for
	// notification again when it's false. When we include group membership
	// changes in the query this will be an optimization.
	return result, nil
}

package repeat

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// Waiter tracks the number of failures and waits for a duration of time based
// on the number of failures.
type Waiter struct {
	backOff backoff.BackOff
}

func NewWaiter(backOff backoff.BackOff) *Waiter {
	// Initialize the backoff here, so we don't have to remember to do it in
	// other places.
	if e, ok := backOff.(*backoff.ExponentialBackOff); ok {
		e.Clock = backoff.SystemClock
		e.Stop = backoff.Stop
		if e.MaxInterval == 0 {
			panic("exponential backoff requires a maximum interval")
		}
		e.Reset()
	}
	return &Waiter{backOff: backOff}
}

// Reset sets the number of failures to 0, so that the next call to Wait sleeps
// for the minimum duration.
func (b Waiter) Reset() {
	b.backOff.Reset()
}

// Wait blocks for the duration of delay calculated by BackOff, or until the
// context is done.
// Returns an error when the context is done, or the retry limit is reached.
// Otherwise, returns nil when the timer waited the full duration.
func (b Waiter) Wait(ctx context.Context) error {
	delay := b.backOff.NextBackOff()
	if delay == backoff.Stop {
		return fmt.Errorf("retry limit exceeded")
	}
	timer := time.NewTimer(delay)
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		timer.Stop()
		return ctx.Err()
	}
}

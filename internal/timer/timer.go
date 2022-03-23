package timer

import (
	"context"
	"time"
)

// Start calls run and then sleeps for interval. Start runs until the context is cancelled.
func Start(ctx context.Context, interval time.Duration, run func(context.Context)) {
	go func() {
		run(ctx)

		for {
			timer := time.NewTimer(interval)
			select {
			case <-timer.C:
				run(ctx)
			case <-ctx.Done():
				timer.Stop()
				return
			}
		}
	}()
}

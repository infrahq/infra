package repeat

import (
	"context"
	"time"
)

// Start a goroutine which repeatedly calls run and then sleep for interval between each
// call. The goroutine runs until the context is cancelled.
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

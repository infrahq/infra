package timer

import (
	"time"

	"github.com/infrahq/infra/internal/logging"
)

type Timer struct {
	stop chan bool
}

func NewTimer() *Timer {
	return &Timer{
		stop: make(chan bool),
	}
}

func (t *Timer) Start(interval time.Duration, sync func()) {
	ticker := time.NewTicker(interval)

	go sync()

	go func() {
		for {
			select {
			case <-ticker.C:
				sync()
			case <-t.stop:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop should be called only once. It waits for the sync function to exit before returning
func (t *Timer) Stop() {
	t.stop <- true
}

// LogTimeElapsed logs the amount of time since this function was defered at the debug level
func LogTimeElapsed(start time.Time, task string) {
	elapsed := time.Since(start)
	logging.L.Sugar().Debugf("%s in %s", task, elapsed)
}

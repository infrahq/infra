package timer

import (
	"time"
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

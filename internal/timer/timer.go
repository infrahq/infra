package timer

import (
	"time"
)

type Timer struct {
	stop chan bool
}

func (t *Timer) Start(interval int, sync func()) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)

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

func (t *Timer) Stop() {
	t.stop <- true
}

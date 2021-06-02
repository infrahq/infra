package server

import "time"

type Sync struct {
	stop chan bool
}

const SYNC_INTERVAL_SECONDS = 10

func (s *Sync) Start(sync func()) {
	ticker := time.NewTicker(SYNC_INTERVAL_SECONDS * time.Second)
	sync()

	go func() {
		for {
			select {
			case <-ticker.C:
				sync()
			case <-s.stop:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *Sync) Stop() {
	s.stop <- true
}

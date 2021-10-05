package timer

import (
	"sync"
	"testing"
	"time"
)

func TestStop(t *testing.T) {
	tm := NewTimer()
	wg := sync.WaitGroup{}
	wg.Add(1)
	tm.Start(5*time.Second, func() {
		wg.Done()
	})
	tm.Stop()
	wg.Wait()
}

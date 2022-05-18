package repeat

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestStart_StopsWithContextCancelled(t *testing.T) {
	done := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	Start(ctx, 5*time.Second, func(ctx2 context.Context) {
		close(done)
	})
	cancel()
	<-done

	assert.Assert(t, time.Since(start) < time.Second)
}

func TestStart_CallsToRunNeverOverlap(t *testing.T) {
	done := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var count, overlap int32
	Start(ctx, time.Millisecond, func(ctx2 context.Context) {
		value := atomic.AddInt32(&overlap, 1)
		// value should only be 1 if the calls never overlap
		assert.Check(t, is.Equal(int32(1), value))

		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&overlap, -1)

		if atomic.AddInt32(&count, 1) == 2 {
			close(done)
		}
	})

	<-done
}

func TestStart_SkipsRunWhenPreviousRunsLongerThanInterval(t *testing.T) {
	done := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var count int32
	start := time.Now()
	Start(ctx, 10*time.Millisecond, func(ctx2 context.Context) {
		time.Sleep(30 * time.Millisecond)

		if atomic.AddInt32(&count, 1) == 5 {
			close(done)
		}
	})

	<-done
	// 30 * 5 + 10 = 160 Milliseconds
	assert.Assert(t, time.Since(start) > 160*time.Millisecond)
}

func TestInGroup_StopsWithContextCancelled(t *testing.T) {
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	InGroup(&wg, ctx, cancel, 2*time.Second, func(ctx2 context.Context, cancel2 context.CancelFunc) {
		cancel()
	})

	wg.Wait()

	assert.Assert(t, time.Since(start) < time.Second)
}

func TestInGroup_StopsAllWithContextCancelled(t *testing.T) {
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	InGroup(&wg, ctx, cancel, 2*time.Second, func(ctx2 context.Context, cancel2 context.CancelFunc) {
		cancel()
	})
	InGroup(&wg, ctx, cancel, 2*time.Second, func(ctx2 context.Context, cancel2 context.CancelFunc) {
		// intentionally blank, does not cancel
	})

	wg.Wait()

	assert.Assert(t, time.Since(start) < time.Second)
}

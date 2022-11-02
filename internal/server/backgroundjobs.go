package server

import (
	"context"
	"time"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/jobs"
)

// BackgroundJobFunc is the interface for implementing a new background job.
//
// currentTime is the time the job was invoked at, and should be used for segmenting records into processable chunks.
// Jobs may want to ignore records past the currentTime, as those will be handled in a future job run.
//
// lastRunAt is the time that the background job was last invoked. This value will only be updated if the function returns a non-error
//
// errors will be logged but will not cause the app to crash.
//
// panics will be caught and logged
//
// jobs should gracefully exit if their context quits, eg ctx.Done() or ctx.Err()
type BackgroundJobFunc func(ctx context.Context, tx data.WriteTxn, lastRunAt, currentTime time.Time) error

func (s *Server) SetupBackgroundJobs() {
	s.registerJob(jobs.RemoveOldDeviceFlowRequests, 10*time.Minute)
	s.registerJob(jobs.RemoveExpiredAccessKeys, 1*time.Minute)
	s.registerJob(jobs.RemoveExpiredPasswordResetTokens, 15*time.Minute)
}

func (s *Server) registerJob(job BackgroundJobFunc, every time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())
	s.routines = append(s.routines, routine{
		run:  jobWrapper(ctx, s.db, job, every),
		stop: cancel,
	})
}

func jobWrapper(ctx context.Context, tx data.WriteTxn, job BackgroundJobFunc, every time.Duration) func() error {
	return func() error { // jobs shouldn't really return errors, we just do this to be compatible with the "routine" struct.
		t := time.NewTicker(every)
		lastRunAt := time.Time{}

		jobWithRescue := func() {
			if ctx.Err() != nil {
				return
			}
			defer func() {
				if err := recover(); err != nil {
					logging.Errorf("background job %s panic: %s", getFuncName(job), err)
				}
			}()

			startAt := time.Now().UTC()
			logging.Debugf("background job %s starting", getFuncName(job))

			// TODO: consider attaching the context to the tx so long-running executions don't hold up app shutdown
			err := job(ctx, tx, lastRunAt, startAt)
			if err != nil {
				logging.Errorf("background job %s error: %s", getFuncName(job), err.Error())
			} else {
				logging.Debugf("background job %s successful, elapsed: %s", getFuncName(job), time.Since(startAt))
				lastRunAt = startAt
			}
		}

		for {
			select {
			case <-t.C:
				jobWithRescue()
			case <-ctx.Done():
				return nil // time to quit.
			}
		}
	}
}

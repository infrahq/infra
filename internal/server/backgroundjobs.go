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
// errors will be logged but will not cause the app to crash.
//
// panics will be caught and logged
//
// jobs should gracefully exit if their context quits, eg ctx.Done() or ctx.Err()
type BackgroundJobFunc func(ctx context.Context, tx *data.DB) error

func (s *Server) SetupBackgroundJobs(ctx context.Context) {
	s.registerJob(ctx, jobs.RemoveOldDeviceFlowRequests, 10*time.Minute)
	s.registerJob(ctx, jobs.RemoveExpiredAccessKeys, 12*time.Hour)
	s.registerJob(ctx, jobs.RemoveExpiredPasswordResetTokens, 15*time.Minute)
	s.registerJob(ctx, jobs.RemoveExpiredCredentialRequests, 14*time.Minute)
}

func (s *Server) registerJob(ctx context.Context, job BackgroundJobFunc, every time.Duration) {
	s.routines = append(s.routines, routine{
		run:  jobWrapper(ctx, s.db, job, every),
		stop: func() {}, // uses the context to stop
	})
}

func jobWrapper(ctx context.Context, tx *data.DB, job BackgroundJobFunc, every time.Duration) func() error {
	tx = &data.DB{DB: tx.WithContext(ctx), DefaultOrgSettings: tx.DefaultOrgSettings, DefaultOrg: tx.DefaultOrg}

	return func() error { // jobs shouldn't return errors, we just do this to be compatible with the "routine" struct.
		t := time.NewTicker(every)
		funcName := getFuncName(job)

		jobWithRescue := func() {
			if ctx.Err() != nil {
				return
			}
			defer func() {
				if err := recover(); err != nil {
					logging.Errorf("background job %s panic: %s", funcName, err)
				}
			}()

			startAt := time.Now().UTC()
			logging.Debugf("background job %s starting", funcName)

			err := job(ctx, tx)
			if err != nil {
				logging.Errorf("background job %s error: %s", funcName, err.Error())
			} else {
				logging.Infof("background job %s successful, elapsed: %s", funcName, time.Since(startAt))
			}
		}

		for {
			select {
			case <-t.C:
				jobWithRescue()
			case <-ctx.Done():
				t.Stop()
				return nil // time to quit.
			}
		}
	}
}

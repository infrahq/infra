package server

import (
	"context"
	"fmt"
	"time"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/jobs"
)

// BackgroundJobFunc is the interface for running periodic background jobs, like
// garbage collection. Errors and panics from the job will be logged by
// jobWrapper, and will not stop the goroutine running the job.
// The job should exit when ctx is cancelled. Unlike most transactions, the
// transaction passed to this job will not have an OrganizationID.
type BackgroundJobFunc func(ctx context.Context, tx *data.Transaction) error

func (s *Server) SetupBackgroundJobs(ctx context.Context) {
	s.registerJob(ctx, jobs.RemoveOldDeviceFlowRequests, 10*time.Minute)
	s.registerJob(ctx, jobs.RemoveExpiredAccessKeys, 12*time.Hour)
	s.registerJob(ctx, jobs.RemoveExpiredPasswordResetTokens, 15*time.Minute)
}

func (s *Server) registerJob(ctx context.Context, job BackgroundJobFunc, every time.Duration) {
	s.routines = append(s.routines, routine{
		run:  jobWrapper(ctx, s.db, job, every),
		stop: func() {}, // uses the context to stop
	})
}

func jobWrapper(ctx context.Context, db *data.DB, job BackgroundJobFunc, every time.Duration) func() error {
	return func() error {
		t := time.NewTicker(every)
		funcName := getFuncName(job)

		jobWithRescue := func() error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			defer func() {
				if err := recover(); err != nil {
					logging.Errorf("background job %s panic: %s", funcName, err)
				}
			}()

			tx, err := db.Begin(ctx, nil)
			if err != nil {
				return fmt.Errorf("failed to start transaction :%w", err)
			}
			if err := job(ctx, tx); err != nil {
				_ = tx.Rollback()
				return err
			}
			return tx.Commit()
		}

		for {
			select {
			case <-t.C:
				startAt := time.Now().UTC()
				logging.Debugf("background job %s starting", funcName)
				if err := jobWithRescue(); err != nil {
					logging.Errorf("background job %s error: %s", funcName, err.Error())
				} else {
					logging.Infof("background job %s successful, elapsed: %s", funcName, time.Since(startAt))
				}
			case <-ctx.Done():
				t.Stop()
				return nil // time to quit.
			}
		}
	}
}

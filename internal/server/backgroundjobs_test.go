package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestJobWrapper(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	db := setupDB(t)

	var signal struct{} // used to signal goroutines to continue
	chReady := make(chan struct{})
	chRun := make(chan struct{})

	runOnce := func() {
		chRun <- signal
		<-chReady
	}

	var runID int
	job := func(tx data.WriteTxn) error {
		chReady <- signal
		<-chRun

		defer func() {
			runID++
		}()

		var err error
		switch runID {
		case 0: // no error commits transaction
			err = data.CreateIdentity(tx, &models.Identity{
				Name:               "user0@example.com",
				OrganizationMember: models.OrganizationMember{OrganizationID: db.DefaultOrg.ID},
			})
		case 1: // error does rollback
			err = data.CreateIdentity(tx, &models.Identity{
				Name:               "user1@example.com",
				OrganizationMember: models.OrganizationMember{OrganizationID: db.DefaultOrg.ID},
			})
			if err == nil {
				err = fmt.Errorf("cause an error")
			}
		case 2: // panic is recovered
			panic("something went wrong")
		case 3: // cancel shutdown
			close(chReady)
			_, err = tx.Exec("SELECT pg_sleep(10)")
		}

		return err
	}

	g := errgroup.Group{}
	fn := backgroundJob(ctx, db, job, time.Millisecond)
	g.Go(fn)
	<-chReady

	runStep(t, "no error commits transaction", func(t *testing.T) {
		runOnce()
		_, err := data.GetIdentity(db, data.GetIdentityOptions{ByName: "user0@example.com"})
		assert.NilError(t, err)
	})
	runStep(t, "an error rolls back the transaction", func(t *testing.T) {
		runOnce()
		_, err := data.GetIdentity(db, data.GetIdentityOptions{ByName: "user1@example.com"})
		assert.ErrorIs(t, err, internal.ErrNotFound)
	})
	runStep(t, "panic is recovered", func(t *testing.T) {
		runOnce()
	})
	runStep(t, "cancel shutdowns the job", func(t *testing.T) {
		runOnce()
		start := time.Now()
		cancel()
		assert.NilError(t, g.Wait())
		assert.Assert(t, time.Since(start) < 3*time.Second)
	})
}

func runStep(t *testing.T, name string, fn func(t *testing.T)) {
	if !t.Run(name, fn) {
		t.FailNow()
	}
}

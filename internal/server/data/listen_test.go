package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestListenForNotify(t *testing.T) {
	type operation struct {
		name         string
		run          func(t *testing.T, tx WriteTxn)
		expectUpdate bool
	}
	type testCase struct {
		name string
		opts ListenChannelGrantsByDestination
		ops  []operation
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	runDBTests(t, func(t *testing.T, db *DB) {
		mainOrg := &models.Organization{Name: "Main", Domain: "main.example.org"}
		assert.NilError(t, CreateOrganization(db, mainOrg))

		otherOrg := &models.Organization{Name: "Other", Domain: "other.example.org"}
		assert.NilError(t, CreateOrganization(db, otherOrg))

		dest := &models.Destination{
			Name:               "mydest",
			Kind:               "ssh",
			OrganizationMember: models.OrganizationMember{OrganizationID: mainOrg.ID},
		}
		createDestinations(t, db, dest)

		run := func(t *testing.T, tc testCase) {
			listener, err := ListenForNotify(ctx, db, tc.opts)
			assert.NilError(t, err)

			chResult := make(chan struct{})
			g, ctx := errgroup.WithContext(ctx)

			g.Go(func() error {
				for {
					err := listener.WaitForNotification(ctx)
					switch {
					case errors.Is(err, context.Canceled):
						return nil
					case err != nil:
						return err
					}
					select {
					case chResult <- struct{}{}:
					case <-ctx.Done():
						return nil
					}
				}
			})

			for _, op := range tc.ops {
				t.Run(op.name, func(t *testing.T) {
					tx, err := db.Begin(ctx, nil)
					assert.NilError(t, err)
					tx = tx.WithOrgID(mainOrg.ID)
					op.run(t, tx)
					assert.NilError(t, tx.Commit())

					if op.expectUpdate {
						isNotBlocked(t, chResult)
						return
					}
					isBlocked(t, chResult)
				})
			}

			cancel()
			assert.NilError(t, g.Wait())
		}

		testcases := []testCase{
			{
				name: "grants by destination",
				opts: ListenChannelGrantsByDestination{
					DestinationID: dest.ID,
					OrgID:         mainOrg.ID,
				},
				ops: []operation{
					{
						name: "grant resource matches exactly",
						run: func(t *testing.T, tx WriteTxn) {
							err := CreateGrant(tx, &models.Grant{
								Subject:   models.NewSubjectForUser(1999),
								Resource:  "mydest",
								Privilege: "view",
							})
							assert.NilError(t, err)
						},
						expectUpdate: true,
					},
					{
						name: "grant resource does not match",
						run: func(t *testing.T, tx WriteTxn) {
							err := CreateGrant(tx, &models.Grant{
								Subject:   models.NewSubjectForUser(1999),
								Resource:  "otherdest",
								Privilege: "mydest",
							})
							assert.NilError(t, err)
						},
					},
					{
						name: "grant resource prefix match",
						run: func(t *testing.T, tx WriteTxn) {
							err := CreateGrant(tx, &models.Grant{
								Subject:   models.NewSubjectForUser(1999),
								Resource:  "mydest.also.ns1",
								Privilege: "admin",
							})
							assert.NilError(t, err)
						},
						expectUpdate: true,
					},
					{
						name: "different org",
						run: func(t *testing.T, tx WriteTxn) {
							tx = tx.(*Transaction).WithOrgID(otherOrg.ID) // nolint
							err := CreateGrant(tx, &models.Grant{
								Subject:   models.NewSubjectForUser(1999),
								Resource:  "mydest",
								Privilege: "admin",
							})
							assert.NilError(t, err)
						},
					},
				},
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				run(t, tc)
			})
		}
	})
}

func isBlocked[T any](t *testing.T, ch chan T) {
	t.Helper()
	select {
	case item := <-ch:
		t.Fatalf("expected operation to be blocked, but it returned: %v", item)
	case <-time.After(200 * time.Millisecond):
	}
}

func isNotBlocked[T any](t *testing.T, ch chan T) (result T) {
	t.Helper()
	timeout := 100 * time.Millisecond
	select {
	case item := <-ch:
		return item
	case <-time.After(timeout):
		t.Fatalf("expected operation to not block, timeout after: %v", timeout)
		return result
	}
}

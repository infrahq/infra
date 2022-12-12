package data

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestPgNotifyWithTriggers(t *testing.T) {
	db := setupDB(t)

	orgID := uid.New()
	destID := uid.New()
	userID := uid.New()
	dcID := uid.New()

	m := sync.Mutex{}
	m.Lock()
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		listener, err := ListenForNotify(context.Background(), db, ListenForNotifyOptions{
			OrgID:                                 orgID,
			DestinationCredentialsByDestinationID: destID,
		})
		assert.NilError(t, err)

		m.Unlock()
		err = listener.WaitForNotification(context.Background())
		assert.NilError(t, err)
		wg.Done()
	}()

	go func() {
		m.Lock() // don't create your new transaction until the previous goroutine is ready

		tx1 := txnForTestCase(t, db, orgID)

		err := CreateDestinationCredential(tx1, &models.DestinationCredential{
			ID:                 dcID,
			OrganizationMember: models.OrganizationMember{OrganizationID: orgID},
			ExpiresAt:          time.Now().Add(10 * time.Minute),
			DestinationID:      destID,
			UserID:             userID,
		})
		assert.NilError(t, err)

		err = tx1.Commit()
		assert.NilError(t, err)
		wg.Done()
	}()

	completed := make(chan bool)

	go func() {
		wg.Wait()
		completed <- true
	}()

	select {
	case <-time.NewTimer(5 * time.Second).C:
		t.Error("test timed out waiting for pg_notify")
	case <-completed:
	}

	t.Run("can update credential with token", func(t *testing.T) {
		dc, err := GetDestinationCredential(db, dcID, orgID)
		assert.NilError(t, err)

		expiry := time.Now().Add(5 * time.Second)
		dc.BearerToken = sql.NullString{Valid: true, String: "foo.bar"}
		dc.ExpiresAt = expiry
		dc.Answered = true

		err = AnswerDestinationCredential(db, dc)
		assert.NilError(t, err)

		dc, err = GetDestinationCredential(db, dcID, orgID)
		assert.NilError(t, err)

		assert.Equal(t, dc.Answered, true)
		assert.Equal(t, dc.ExpiresAt.UnixNano(), expiry.UnixNano())
		assert.Equal(t, dc.BearerToken.String, "foo.bar")

	})

	t.Run("can list credentials", func(t *testing.T) {
		destID := uid.New()
		orgID := uid.New()
		dc := &models.DestinationCredential{
			ID:                 uid.New(),
			OrganizationMember: models.OrganizationMember{OrganizationID: orgID},
			ExpiresAt:          time.Now().Add(5 * time.Second),
			DestinationID:      destID,
			UserID:             uid.New(),
		}
		err := CreateDestinationCredential(db, dc)
		assert.NilError(t, err)

		tx, err := db.Begin(context.Background(), nil)
		assert.NilError(t, err)
		tx = tx.WithOrgID(orgID)
		defer func() { _ = tx.Rollback() }()

		creds, err := ListDestinationCredentials(tx, destID)
		assert.NilError(t, err)

		// inject used updateIndex because we don't want to compare it
		dc.UpdateIndex = creds[0].UpdateIndex

		assert.DeepEqual(t, creds, []models.DestinationCredential{*dc})
	})

	t.Run("can remove expired credentials", func(t *testing.T) {
		dc := &models.DestinationCredential{
			ID:                 uid.New(),
			OrganizationMember: models.OrganizationMember{OrganizationID: orgID},
			ExpiresAt:          time.Now(),
			DestinationID:      uid.New(),
			UserID:             uid.New(),
		}
		err := CreateDestinationCredential(db, dc)
		assert.NilError(t, err)

		err = RemoveExpiredDestinationCredentials(db)
		assert.NilError(t, err)

		_, err = GetDestinationCredential(db, dc.ID, dc.OrganizationID)
		assert.ErrorContains(t, err, "not found")
	})
}

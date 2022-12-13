package data

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestPgNotifyRequestWithTriggers(t *testing.T) {
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
			RequestExpiresAt:   time.Now().Add(10 * time.Minute),
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
}

func TestPgNotifyResponseWithTriggers(t *testing.T) {
	db := setupDB(t)

	orgID := uid.New()
	destID := uid.New()
	userID := uid.New()
	dcID := uid.New()

	dc := &models.DestinationCredential{
		ID:                 dcID,
		OrganizationMember: models.OrganizationMember{OrganizationID: orgID},
		RequestExpiresAt:   time.Now().Add(10 * time.Minute),
		DestinationID:      destID,
		UserID:             userID,
	}
	err := CreateDestinationCredential(db, dc)
	assert.NilError(t, err)

	m := sync.Mutex{}
	m.Lock()
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		listener, err := ListenForNotify(context.Background(), db, ListenForNotifyOptions{
			OrgID:                      orgID,
			DestinationCredentialsByID: dcID,
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

		expiry := time.Now().Add(10 * time.Minute)
		dc.BearerToken = "robin.sparkles"
		dc.CredentialExpiresAt = &expiry

		err = AnswerDestinationCredential(tx1, dc)
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
}

func TestDestinationCredentials(t *testing.T) {
	db := setupDB(t)
	orgID := uid.New()
	destID := uid.New()
	userID := uid.New()
	dcID := uid.New()
	requestExpiry := time.Now().Add(10 * time.Minute)

	t.Run("can create credential", func(t *testing.T) {
		err := CreateDestinationCredential(db, &models.DestinationCredential{
			ID:                 dcID,
			OrganizationMember: models.OrganizationMember{OrganizationID: orgID},
			RequestExpiresAt:   requestExpiry,
			DestinationID:      destID,
			UserID:             userID,
		})
		assert.NilError(t, err)

	})

	t.Run("can update credential with token", func(t *testing.T) {
		dc, err := GetDestinationCredential(db, dcID, orgID)
		assert.NilError(t, err)

		credentialExpiry := time.Now().Add(5 * time.Second)
		dc.BearerToken = "foo.bar"
		dc.CredentialExpiresAt = &credentialExpiry
		dc.Answered = true

		err = AnswerDestinationCredential(db, dc)
		assert.NilError(t, err)

		dc, err = GetDestinationCredential(db, dcID, orgID)
		assert.NilError(t, err)

		expected := &models.DestinationCredential{
			ID:                 dcID,
			OrganizationMember: models.OrganizationMember{OrganizationID: orgID},
			RequestExpiresAt:   requestExpiry,
			UserID:             userID,
			DestinationID:      destID,

			Answered:            true,
			CredentialExpiresAt: &credentialExpiry,
			BearerToken:         "foo.bar",
		}

		assert.DeepEqual(t, dc, expected, destCredCompareOpts)
	})

	t.Run("can list credentials", func(t *testing.T) {
		destID := uid.New()
		orgID := uid.New()
		dc := &models.DestinationCredential{
			ID:                 uid.New(),
			OrganizationMember: models.OrganizationMember{OrganizationID: orgID},
			RequestExpiresAt:   time.Now().Add(5 * time.Second).Truncate(time.Millisecond),
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

		assert.DeepEqual(t, creds, []models.DestinationCredential{*dc}, destCredCompareOpts)
	})

	t.Run("can remove expired credentials", func(t *testing.T) {
		dc := &models.DestinationCredential{
			ID:                 uid.New(),
			OrganizationMember: models.OrganizationMember{OrganizationID: orgID},
			RequestExpiresAt:   time.Now(),
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

var destCredCompareOpts = cmp.Options{
	cmp.FilterPath(opt.PathField(models.DestinationCredential{}, "UpdateIndex"), notZeroInt64),
	cmp.FilterPath(opt.PathField(models.DestinationCredential{}, "RequestExpiresAt"), opt.TimeWithThreshold(1*time.Second)),
	cmp.FilterPath(opt.PathField(models.DestinationCredential{}, "CredentialExpiresAt"), TimePtrWithThreshold(1*time.Second)),
}

var notZeroInt64 = cmp.Comparer(func(x, y interface{}) bool {
	xi, _ := x.(int64)
	yi, _ := y.(int64)

	return xi != 0 || yi != 0
})

func TimePtrWithThreshold(threshold time.Duration) cmp.Option {
	return cmp.Comparer(cmpTimePtr(threshold))
}

func cmpTimePtr(threshold time.Duration) func(x, y *time.Time) bool {
	return func(x, y *time.Time) bool {
		if x == nil && y == nil {
			return true
		}
		if x == nil || y == nil {
			return false
		}
		if x.IsZero() || y.IsZero() {
			return false
		}
		delta := x.Sub(*y)
		return delta <= threshold && delta >= -threshold
	}
}

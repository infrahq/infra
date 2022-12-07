package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/uid"
	"github.com/jackc/pgx/v4"
	pgxstdlib "github.com/jackc/pgx/v4/stdlib"
)

type Listener struct {
	sqlDB   *sql.DB
	pgxConn *pgx.Conn

	isMatchingNotify func(payload string) error
}

var errNotificationNoMatch = fmt.Errorf("notification did not match")

// WaitForNotification blocks until the listener receivers a notification on
// one of the channels, or until the context is cancelled.
// Returns the notification on success, or an error on failure or timeout.
func (l *Listener) WaitForNotification(ctx context.Context) error {
	for {
		notficaition, err := l.pgxConn.WaitForNotification(ctx)
		if err != nil {
			return err
		}

		if l.isMatchingNotify != nil {
			err = l.isMatchingNotify(notficaition.Payload)
			switch {
			case errors.Is(err, errNotificationNoMatch):
				continue
			case err != nil:
				return err
			default:
				return nil
			}
		}
		return nil
	}
}

func (l *Listener) Release(ctx context.Context) error {
	var errs []error
	logging.Debugf("unlisten *")
	if _, err := l.pgxConn.Exec(ctx, `UNLISTEN *`); err != nil {
		errs = append(errs, err)
	}
	if err := pgxstdlib.ReleaseConn(l.sqlDB, l.pgxConn); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to unlisten to postgres channels: %v", errs)
	}
	return nil
}

type ListenForNotifyOptions struct {
	OrgID                             uid.ID
	GrantsByDestination               string
	CredentialRequestsByDestinationID uid.ID
	CredentialRequestsByID            uid.ID
}

// ListenForNotify starts listening for notification on one or more
// postgres channels for notifications that a grant has changed. The channels to
// listen on are determined by opts. Use Listener.WaitForNotification to block
// and receive notifications.
//
// If error is nil the caller must call Listener.Release to return the database
// connection to the pool.
func ListenForNotify(ctx context.Context, db *DB, opts ListenForNotifyOptions) (*Listener, error) {
	if opts.OrgID == 0 {
		return nil, fmt.Errorf("OrgID is required")
	}

	sqlDB := db.SQLdb()
	pgxConn, err := pgxstdlib.AcquireConn(sqlDB)
	if err != nil {
		return nil, err
	}

	listener := &Listener{sqlDB: sqlDB, pgxConn: pgxConn}

	var channel string
	switch {
	case opts.GrantsByDestination != "":
		channel = fmt.Sprintf("grants_%d", opts.OrgID)
	case opts.CredentialRequestsByDestinationID != 0:
		channel = fmt.Sprintf("credreq_%d_%d", opts.OrgID, opts.CredentialRequestsByDestinationID)
	case opts.CredentialRequestsByID != 0:
		channel = fmt.Sprintf("credreq_%d_%d", opts.OrgID, opts.CredentialRequestsByID)
	}

	logging.Debugf("listing for notify on %s", channel)
	_, err = pgxConn.Exec(ctx, "SELECT listen_on_chan($1)", channel)
	if err != nil {
		if err := pgxstdlib.ReleaseConn(sqlDB, pgxConn); err != nil {
			logging.L.Warn().Err(err).Msgf("release pgx conn")
		}
		return nil, err
	}

	switch {
	case opts.GrantsByDestination != "":
		listener.isMatchingNotify = func(payload string) error {
			var grant grantJSON
			err := json.Unmarshal([]byte(payload), &grant)
			if err != nil {
				return err
			}
			destination, _, _ := strings.Cut(grant.Resource, ".")
			if destination != opts.GrantsByDestination {
				return errNotificationNoMatch
			}
			return nil
		}
	}
	return listener, nil
}

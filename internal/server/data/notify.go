package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4"
	pgxstdlib "github.com/jackc/pgx/v4/stdlib"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/uid"
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

		if l.isMatchingNotify == nil {
			return nil
		}

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
	OrgID                                 uid.ID
	GrantsByDestinationName               string
	DestinationCredentialsByDestinationID uid.ID
	DestinationCredentialsByID            uid.ID
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
	case opts.GrantsByDestinationName != "":
		channel = fmt.Sprintf("grants_%d", opts.OrgID)
	case opts.DestinationCredentialsByDestinationID != 0:
		channel = fmt.Sprintf("credreq_%s_%s", opts.OrgID.String(), opts.DestinationCredentialsByDestinationID.String())
	case opts.DestinationCredentialsByID != 0:
		channel = fmt.Sprintf("credans_%s_%s", opts.OrgID.String(), opts.DestinationCredentialsByID.String())
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
	case opts.GrantsByDestinationName != "":
		listener.isMatchingNotify = func(payload string) error {
			var grant grantJSON
			err := json.Unmarshal([]byte(payload), &grant)
			if err != nil {
				return err
			}
			if grant.DestinationName != opts.GrantsByDestinationName {
				return errNotificationNoMatch
			}
			return nil
		}
	}
	return listener, nil
}

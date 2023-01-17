package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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

// ListenChannelDescriptor provides a channel name that ListenForNotify uses
// to listen for notifications.
type ListenChannelDescriptor interface {
	// Channel returns the name of the channel to listen on. The channel
	// name is limited by the size of a postgres identifier. It should
	// be no more than 40 characters, because 23 characters are reserved
	// for the schema name that we need to prepend.
	//
	// The channel name should use camel case, and follow the convention of
	//
	//   <channel type>.<encoded orgID>.<encoded entity ID>
	//
	// The encoded ids use internal/uid base58 encoding. Encoded IDs are up to
	// 11 characters long, so the channel type should be no more than 16
	// characters.
	Channel() string
}

// ListenForNotify starts listening for notification on one or more
// postgres channels for notifications that a grant has changed. The channels to
// listen on are determined by opts. Use Listener.WaitForNotification to block
// and receive notifications.
//
// If error is nil the caller must call Listener.Release to return the database
// connection to the pool.
func ListenForNotify(ctx context.Context, db *DB, descriptors ...ListenChannelDescriptor) (*Listener, error) {
	sqlDB := db.SQLdb()
	pgxConn, err := pgxstdlib.AcquireConn(sqlDB)
	if err != nil {
		return nil, err
	}

	listener := &Listener{sqlDB: sqlDB, pgxConn: pgxConn}

	for _, descriptor := range descriptors {
		_, err = pgxConn.Exec(ctx, "SELECT listen_on_chan($1)", descriptor.Channel())
		if err != nil {
			if err := pgxstdlib.ReleaseConn(sqlDB, pgxConn); err != nil {
				logging.L.Warn().Err(err).Msgf("release pgx conn")
			}
			return nil, err
		}

		// FIXME: this won't work now that we support multiple channels
		if matcher, ok := descriptor.(interface {
			Match(payload string) error
		}); ok {
			listener.isMatchingNotify = matcher.Match
		}
	}
	return listener, nil
}

type ListenChannelGrantsByDestination struct {
	OrgID       uid.ID
	Destination string
}

func (l ListenChannelGrantsByDestination) Channel() string {
	return fmt.Sprintf("grants_%d", l.OrgID)
}

func (l ListenChannelGrantsByDestination) Match(payload string) error {
	var grant grantJSON
	err := json.Unmarshal([]byte(payload), &grant)
	if err != nil {
		return err
	}
	destination, _, _ := strings.Cut(grant.Resource, ".")
	if destination != l.Destination {
		return errNotificationNoMatch
	}
	return nil
}

type ListenChannelDestinationCredentialsByDestinationID struct {
	OrgID         uid.ID
	DestinationID uid.ID
}

func (l ListenChannelDestinationCredentialsByDestinationID) Channel() string {
	return fmt.Sprintf("credreq_%v_%v", l.OrgID, l.DestinationID)
}

type ListenChannelDestinationCredentialsByID struct {
	OrgID                    uid.ID
	DestinationCredentialsID uid.ID
}

func (l ListenChannelDestinationCredentialsByID) Channel() string {
	return fmt.Sprintf("credans_%v_%v", l.OrgID, l.DestinationCredentialsID)
}

type ListenChannelGroupMembership struct {
	OrgID   uid.ID
	GroupID uid.ID
}

func (l ListenChannelGroupMembership) Channel() string {
	return fmt.Sprintf("groupMembers.%v.%v", l.OrgID, l.GroupID)
}

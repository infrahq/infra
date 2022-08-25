package server

import (
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type Properties = analytics.Properties

type Telemetry struct {
	client  analytics.Client
	db      data.GormTxn
	infraID uid.ID
}

func NewTelemetry(db data.GormTxn, infraID uid.ID) *Telemetry {
	return &Telemetry{
		client:  analytics.New(internal.TelemetryWriteKey),
		db:      db,
		infraID: infraID,
	}
}

func (t *Telemetry) Enqueue(track analytics.Message) error {
	if internal.TelemetryWriteKey == "" {
		return nil
	}

	switch track := track.(type) {
	case analytics.Track:
		if track.Properties == nil {
			track.Properties = analytics.Properties{}
		}

		track.Properties.Set("infraId", t.infraID)
		track.Properties.Set("version", internal.Version)
	case analytics.Page:
		if track.Properties == nil {
			track.Properties = analytics.Properties{}
		}

		track.Properties.Set("infraId", t.infraID)
		track.Properties.Set("version", internal.Version)
	}

	return t.client.Enqueue(track)
}

func (t *Telemetry) Close() {
	if t == nil {
		return
	}
	// the only error here is "already closed"
	_ = t.client.Close()
}

func (t *Telemetry) EnqueueHeartbeat() {
	users, err := data.GlobalCount[models.Identity](t.db)
	if err != nil {
		logging.Debugf("%s", err.Error())
	}

	groups, err := data.GlobalCount[models.Group](t.db)
	if err != nil {
		logging.Debugf("%s", err.Error())
	}

	grants, err := data.GlobalCount[models.Grant](t.db)
	if err != nil {
		logging.Debugf("%s", err.Error())
	}

	providers, err := data.GlobalCount[models.Provider](t.db)
	if err != nil {
		logging.Debugf("%s", err.Error())
	}

	destinations, err := data.GlobalCount[models.Destination](t.db)
	if err != nil {
		logging.Debugf("%s", err.Error())
	}

	t.Event("heartbeat", "", map[string]interface{}{
		"users":        users,
		"groups":       groups,
		"providers":    providers,
		"destinations": destinations,
		"grants":       grants,
	})
}

func (t *Telemetry) RouteEvent(c *gin.Context, event string, properties ...map[string]interface{}) {
	var uid string
	if c != nil {
		if u := access.AuthenticatedIdentity(c); u != nil {
			uid = u.ID.String()
		}
	}

	t.Event(event, uid, properties...)
}

func (t *Telemetry) Event(event string, userId string, properties ...map[string]interface{}) {
	if t == nil {
		return
	}
	track := analytics.Track{
		AnonymousId: t.infraID.String(),
		UserId:      userId,
		Timestamp:   time.Now().UTC(),
		Event:       "server:" + event,
		Properties:  analytics.Properties{},
	}

	if len(properties) > 0 {
		for k, v := range properties[0] {
			track.Properties.Set(k, v)
		}
	}

	if err := t.Enqueue(track); err != nil {
		logging.Debugf("%s", err.Error())
	}
}

func (t *Telemetry) Alias(id string) {
	if t == nil {
		return
	}
	err := t.Enqueue(analytics.Alias{
		PreviousId: t.infraID.String(),
		UserId:     id,
		Timestamp:  time.Now().UTC(),
	})
	if err != nil {
		logging.Debugf("%s", err.Error())
	}
}

func (t *Telemetry) User(id string, name string) {
	if t == nil {
		return
	}
	err := t.Enqueue(analytics.Identify{
		UserId:    id,
		Traits:    analytics.NewTraits().SetEmail(name),
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		logging.Debugf("%s", err.Error())
	}
}

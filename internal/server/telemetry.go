package server

import (
	"errors"
	"time"

	"gopkg.in/segmentio/analytics-go.v3"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type Properties = analytics.Properties

type Telemetry struct {
	client analytics.Client
	db     *gorm.DB
}

func NewTelemetry(db *gorm.DB) (*Telemetry, error) {
	if db == nil {
		return nil, errors.New("db cannot be nil")
	}

	return &Telemetry{
		client: analytics.New(internal.TelemetryWriteKey),
		db:     db,
	}, nil
}

var settings *models.Settings

func (t *Telemetry) Enqueue(track analytics.Message) error {
	if internal.TelemetryWriteKey == "" {
		return nil
	}

	if settings == nil {
		var err error
		settings, err = data.GetSettings(t.db)
		if err != nil {
			return err
		}
	}

	switch track := track.(type) {
	case analytics.Track:
		if track.Properties == nil {
			track.Properties = analytics.Properties{}
		}

		track.Properties.Set("infraId", settings.ID)
		track.Properties.Set("version", internal.Version)
	case analytics.Page:
		if track.Properties == nil {
			track.Properties = analytics.Properties{}
		}

		track.Properties.Set("infraId", settings.ID)
		track.Properties.Set("version", internal.Version)
	}

	return t.client.Enqueue(track)
}

func (t *Telemetry) Close() {
	if t.client != nil {
		t.client.Close()
	}
}

func (t *Telemetry) EnqueueHeartbeat() {
	users, err := data.Count[models.Identity](t.db)
	if err != nil {
		logging.S.Debug(err)
	}

	groups, err := data.Count[models.Group](t.db)
	if err != nil {
		logging.S.Debug(err)
	}

	grants, err := data.Count[models.Grant](t.db)
	if err != nil {
		logging.S.Debug(err)
	}

	providers, err := data.Count[models.Provider](t.db)
	if err != nil {
		logging.S.Debug(err)
	}

	destinations, err := data.Count[models.Destination](t.db)
	if err != nil {
		logging.S.Debug(err)
	}

	t.Event(nil, "infra.heartbeat", map[string]interface{}{
		"users":        users,
		"groups":       groups,
		"providers":    providers,
		"destinations": destinations,
		"grants":       grants,
	})
}

func (t *Telemetry) Event(c *gin.Context, event string, properties ...map[string]interface{}) {
	if t == nil {
		return
	}
	var userIDStr string
	if c != nil {
		if user, ok := c.Get("user"); ok {
			if u, ok := user.(*models.User); ok {
				userIDStr = u.ID.String()
			}
		}
	}
	track := analytics.Track{
		AnonymousId: "system",
		Timestamp:   time.Now().UTC(),
		Event:       "server." + event,
		UserId:      userIDStr,
		Properties:  analytics.Properties{},
	}

	if len(properties) > 0 {
		for k, v := range properties[0] {
			track.Properties.Set(k, v)
		}
	}

	if err := t.Enqueue(track); err != nil {
		logging.S.Debug(err)
	}
}

func (t *Telemetry) User(id uid.ID) {
	if t == nil {
		return
	}
	err := t.Enqueue(analytics.Identify{
		UserId:    id.String(),
		Timestamp: time.Now().UTC(),
		Traits: analytics.Traits{
			"userType": "user", // or machine when bruce's changes are in
		},
	})
	if err != nil {
		logging.S.Debug(err)
	}
}

func (t *Telemetry) Group(identityID, groupID uid.ID, traits map[string]interface{}) {
	if t == nil {
		return
	}
	err := t.Enqueue(analytics.Group{
		UserId:    identityID.String(),
		GroupId:   groupID.String(),
		Timestamp: time.Now().UTC(),
		Traits:    traits,
	})
	if err != nil {
		logging.S.Debug(err)
	}
}

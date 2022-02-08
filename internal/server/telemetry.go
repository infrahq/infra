package server

import (
	"errors"

	"gopkg.in/segmentio/analytics-go.v3"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

type Telemetry struct {
	client  analytics.Client
	db      *gorm.DB
}

func NewTelemetry(db *gorm.DB) (*Telemetry, error) {
	if db == nil {
		return nil, errors.New("db cannot be nil")
	}

	return &Telemetry{
		client:  analytics.New(internal.TelemetryWriteKey),
		db:      db,
	}, nil
}

func (t *Telemetry) Enqueue(track analytics.Track) error {
	if internal.TelemetryWriteKey == "" {
		return nil
	}

	if track.Properties == nil {
		track.Properties = analytics.NewProperties()
	}

	settings, err := data.GetSettings(t.db)
	if err != nil {
		return err
	}

	track.Properties.Set("infraId", settings.ID)
	track.Properties.Set("version", internal.Version)

	return t.client.Enqueue(track)
}

func (t *Telemetry) Close() {
	if t.client != nil {
		t.client.Close()
	}
}

func (t *Telemetry) EnqueueHeartbeat() error {
	users, err := data.Count[models.User](t.db)
	if err != nil {
		return err
	}

	groups, err := data.Count[models.User](t.db)
	if err != nil {
		return err
	}

	grants, err := data.Count[models.Grant](t.db)
	if err != nil {
		return err
	}

	providers, err := data.Count[models.Provider](t.db)
	if err != nil {
		return err
	}

	destinations, err := data.Count[models.Destination](t.db)
	if err != nil {
		return err
	}

	return t.Enqueue(analytics.Track{
		AnonymousId: "system",
		Event:       "infra.heartbeat",
		Properties: analytics.NewProperties().
			Set("users", users).
			Set("groups", groups).
			Set("providers", providers).
			Set("destinations", destinations).
			Set("grants", grants),
	})
}

package registry

import (
	"errors"

	"gopkg.in/segmentio/analytics-go.v3"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
)

type Telemetry struct {
	enabled bool
	client  analytics.Client
	db      *gorm.DB
}

func NewTelemetry(db *gorm.DB) (*Telemetry, error) {
	if db == nil {
		return nil, errors.New("db cannot be nil")
	}

	enabled := internal.TelemetryWriteKey != ""

	return &Telemetry{
		enabled: enabled,
		client:  analytics.New(internal.TelemetryWriteKey),
		db:      db,
	}, nil
}

func (t *Telemetry) SetEnabled(enabled bool) {
	t.enabled = enabled
}

func (t *Telemetry) Enqueue(track analytics.Track) error {
	if !t.enabled {
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
	users, err := data.Count(t.db, &models.User{}, &models.User{})
	if err != nil {
		return err
	}

	groups, err := data.Count(t.db, &models.Group{}, &models.Group{})
	if err != nil {
		return err
	}

	roles, err := data.Count(t.db, &models.Role{}, &models.Role{})
	if err != nil {
		return err
	}

	providers, err := data.Count(t.db, &models.Provider{}, &models.Provider{})
	if err != nil {
		return err
	}

	destinations, err := data.Count(t.db, &models.Destination{}, &models.Destination{})
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
			Set("roles", roles),
	})
}

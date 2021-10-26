package registry

import (
	"errors"

	"github.com/infrahq/infra/internal"
	"gopkg.in/segmentio/analytics-go.v3"
	"gorm.io/gorm"
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

	var settings Settings
	if err := t.db.First(&settings).Error; err != nil {
		return err
	}

	track.Properties.Set("infraId", settings.Id)
	track.Properties.Set("version", internal.Version)

	return t.client.Enqueue(track)
}

func (t *Telemetry) Close() {
	if t.client != nil {
		t.client.Close()
	}
}

func (t *Telemetry) EnqueueHeartbeat() error {
	var users, groups, sources, destinations, roles int64
	if err := t.db.Model(&User{}).Count(&users).Error; err != nil {
		return err
	}

	if err := t.db.Model(&Group{}).Count(&groups).Error; err != nil {
		return err
	}

	if err := t.db.Model(&Source{}).Count(&sources).Error; err != nil {
		return err
	}

	if err := t.db.Model(&Destination{}).Count(&destinations).Error; err != nil {
		return err
	}

	if err := t.db.Model(&Role{}).Count(&roles).Error; err != nil {
		return err
	}

	return t.Enqueue(analytics.Track{
		AnonymousId: "system",
		Event:       "infra.heartbeat",
		Properties: analytics.NewProperties().
			Set("users", users).
			Set("groups", groups).
			Set("sources", sources).
			Set("destinations", destinations).
			Set("roles", roles),
	})
}

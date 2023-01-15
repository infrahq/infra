package server

import (
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/uid"
)

type Properties = analytics.Properties

type Telemetry struct {
	client  analytics.Client
	db      *data.DB
	infraID uid.ID
}

// todo: store global settings like email/signup configured
func NewTelemetry(db *data.DB, infraID uid.ID) *Telemetry {
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
	users, err := data.CountAllIdentities(t.db)
	if err != nil {
		logging.Debugf("%s", err.Error())
	}

	groups, err := data.CountAllGroups(t.db)
	if err != nil {
		logging.Debugf("%s", err.Error())
	}

	grants, err := data.CountAllGrants(t.db)
	if err != nil {
		logging.Debugf("%s", err.Error())
	}

	providers, err := data.CountAllProviders(t.db)
	if err != nil {
		logging.Debugf("%s", err.Error())
	}

	destinations, err := data.CountAllDestinations(t.db)
	if err != nil {
		logging.Debugf("%s", err.Error())
	}

	t.Event("heartbeat", "", "", map[string]interface{}{
		"users":        users,
		"groups":       groups,
		"providers":    providers,
		"destinations": destinations,
		"grants":       grants,
	})
}

func (t *Telemetry) RouteEvent(c *gin.Context, event string, properties ...map[string]interface{}) {
	var uid, oid string
	if c != nil {
		a := getRequestContext(c).Authenticated
		if user := a.User; user != nil {
			uid = user.ID.String()
		}

		if org := a.Organization; org != nil {
			oid = org.ID.String()
		}
	}

	t.Event(event, uid, oid, properties...)
}

func (t *Telemetry) Event(event string, userId string, orgId string, properties ...map[string]interface{}) {
	if t == nil {
		return
	}
	track := analytics.Track{
		AnonymousId: t.infraID.String(),
		UserId:      userId,
		Timestamp:   time.Now().UTC(),
		Event:       "server:" + event,
		Properties: map[string]interface{}{
			"orgId": orgId,
		},
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

func (t *Telemetry) Org(id, userID, name, domain string) {
	if t == nil {
		return
	}
	err := t.Enqueue(analytics.Group{
		GroupId: id,
		UserId:  userID,
		Traits: map[string]interface{}{
			"name":   name,
			"$name":  name,
			"domain": domain,
			"orgId":  id,
		},
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		logging.Debugf("%s", err.Error())
	}
}

func (t *Telemetry) OrgMembership(orgID, userID string) {
	if t == nil {
		return
	}
	err := t.Enqueue(analytics.Group{
		GroupId:   orgID,
		UserId:    userID,
		Timestamp: time.Now().UTC(),
		Traits: map[string]interface{}{
			"orgId": orgID,
		},
	})
	if err != nil {
		logging.Debugf("%s", err.Error())
	}
}

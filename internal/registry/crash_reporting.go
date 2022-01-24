package registry

import (
	"time"

	"github.com/getsentry/sentry-go"
)

func recoverWithSentryHub(hub *sentry.Hub) {
	err := recover()
	if err != nil {
		hub.Recover(err)
		sentry.Flush(time.Second * 5)
	}
}

package infra

import (
	"time"

	"github.com/getsentry/sentry-go"
)

func newSentryHub(name string) *sentry.Hub {
	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("goroutine", name)
	})

	return hub
}

func recoverWithSentryHub(hub *sentry.Hub) {
	err := recover()
	if err != nil {
		hub.Recover(err)
		sentry.Flush(time.Second * 5)
	}
}

package audit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/apis/audit"
	"k8s.io/apiserver/pkg/audit/policy"
	genericapifilters "k8s.io/apiserver/pkg/endpoints/filters"
	"k8s.io/apiserver/pkg/endpoints/request"
	genericfilters "k8s.io/apiserver/pkg/server/filters"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
)

type AuditEvent struct {
	Level       string  `json:"level"`
	User        string  `json:"user"`
	Destination string  `json:"destination"`
	Timestamp   float64 `json:"ts"`

	Action      string `json:"action"`
	Kind        string `json:"kind"`
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Resource    string `json:"resource"`
	Subresource string `json:"subresource"`

	Status int `json:"status"`
}

type EmptyAuditSink struct{}

func (k *EmptyAuditSink) ProcessEvents(events ...*audit.Event) bool {
	return true
}

func AuditPrintMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, ok := r.Context().Value(internal.HttpContextKeyEmail{}).(string)
		if !ok {
			logging.L.Warn("audit middleware: unable to retrieve email from context")
		}

		destination, ok := r.Context().Value(internal.HttpContextKeyDestinationName{}).(string)
		if !ok {
			logging.L.Warn("audit middleware: unable to retrieve destination from context")
		}

		next.ServeHTTP(w, r)

		e := request.AuditEventFrom(r.Context())

		event := AuditEvent{}
		event.Level = "audit"
		event.Action = e.Verb
		event.Kind = e.Kind

		nanos := time.Now().UnixNano()
		event.Timestamp = float64(nanos) / float64(time.Second)

		if e.ObjectRef != nil {
			event.Namespace = e.ObjectRef.Namespace
			event.Name = e.ObjectRef.Name
			event.Resource = e.ObjectRef.Resource
			event.Subresource = e.ObjectRef.Subresource
		}

		if e.ResponseStatus != nil {
			event.Status = int(e.ResponseStatus.Code)
		}

		event.User = email
		event.Destination = destination

		bts, err := json.Marshal(&event)
		if err != nil {
			logging.S.Errorf("audit print event marshal: %w", err)
			return
		}

		fmt.Println(string(bts))
	})
}

func AuditMiddleware(next http.Handler) http.Handler {
	p := &audit.Policy{
		Rules: []audit.PolicyRule{
			{
				Level:      audit.LevelRequestResponse,
				OmitStages: []audit.Stage{audit.StageRequestReceived},
			},
		},
	}

	withAudit := genericapifilters.WithAudit(AuditPrintMiddleware(next), &EmptyAuditSink{}, policy.NewChecker(p), genericfilters.BasicLongRunningRequestCheck(sets.NewString("watch"), sets.NewString()))

	return genericapifilters.WithRequestInfo(withAudit, &request.RequestInfoFactory{
		APIPrefixes:          sets.NewString("api", "apis"),
		GrouplessAPIPrefixes: sets.NewString("api"),
	})
}

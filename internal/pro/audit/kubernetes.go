package audit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/apis/audit"
	"k8s.io/apiserver/pkg/audit/policy"
	"k8s.io/apiserver/pkg/authentication/user"
	genericapifilters "k8s.io/apiserver/pkg/endpoints/filters"
	"k8s.io/apiserver/pkg/endpoints/request"
	genericfilters "k8s.io/apiserver/pkg/server/filters"
)

type KubernetesAuditEvent struct {
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

type KubernetesAuditSink struct{}

func (k *KubernetesAuditSink) ProcessEvents(events ...*audit.Event) bool {
	for _, e := range events {
		event := KubernetesAuditEvent{}
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

		event.User = e.User.Username

		bts, err := json.Marshal(&event)
		if err != nil {
			logging.S.Errorf("audit process event marshal: %w", err)
			return false
		}

		fmt.Println(string(bts))
	}

	return true
}

func KubernetesUserFromInfraUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, ok := r.Context().Value(internal.HttpContextKeyEmail{}).(string)
		if !ok {
			logging.L.Debug("Audit middleware unable to retrieve email from context")
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		next.ServeHTTP(w, r.WithContext(request.WithUser(r.Context(), &user.DefaultInfo{Name: email})))
	})
}

func KubernetesAuditMiddleware(next http.Handler, destination string) http.Handler {
	p := &audit.Policy{
		Rules: []audit.PolicyRule{
			{
				Level:      audit.LevelRequestResponse,
				OmitStages: []audit.Stage{audit.StageRequestReceived},
			},
		},
	}

	withAudit := genericapifilters.WithAudit(next, &KubernetesAuditSink{}, policy.NewChecker(p), genericfilters.BasicLongRunningRequestCheck(sets.NewString("watch"), sets.NewString()))

	return genericapifilters.WithRequestInfo(KubernetesUserFromInfraUserMiddleware(withAudit), &request.RequestInfoFactory{
		APIPrefixes:          sets.NewString("api", "apis"),
		GrouplessAPIPrefixes: sets.NewString("api"),
	})
}

package audit

import (
	"encoding/json"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/endpoints/request"
)

type Event struct {
	Level       string `json:"level"`
	User        string `json:"user"`
	Destination string `json:"destination"`

	Action      string `json:"action"`
	Kind        string `json:"kind"`
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Resource    string `json:"resource"`
	Subresource string `json:"subresource"`
}

func Log(req *http.Request, user string, destination string) error {
	rif := &request.RequestInfoFactory{
		APIPrefixes:          sets.NewString("api", "apis"),
		GrouplessAPIPrefixes: sets.NewString("api"),
	}

	ri, err := rif.NewRequestInfo(req)
	if err != nil {
		return err
	}

	event := &Event{
		Level:       "audit",
		User:        user,
		Destination: destination,
		Action:      ri.Verb,
		Kind:        ri.Resource,
		Namespace:   ri.Namespace,
		Name:        ri.Name,
		Resource:    ri.Resource,
		Subresource: ri.Subresource,
	}

	bts, err := json.Marshal(event)
	if err != nil {
		return err
	}

	fmt.Println(string(bts))

	return nil
}

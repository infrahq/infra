package pro

import (
	"encoding/json"
	"fmt"
)

type Event struct {
	User        string `json:"user"`
	Destination string `json:"destination"`
	Action      string `json:"action"`

	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`

	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
}

func Log(e Event) error {
	s, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("unmarshalling: %w", err)
	}

	fmt.Println(string(s))

	return nil
}

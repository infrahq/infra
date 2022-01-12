package api

import (
	"encoding/json"
	"fmt"
)

// ProviderKind the model 'ProviderKind'
type ProviderKind string

// List of ProviderKind
const (
	ProviderKindOkta ProviderKind = "okta"
)

var ValidProviderKinds = []ProviderKind{
	ProviderKindOkta,
}

func (v *ProviderKind) UnmarshalJSON(src []byte) error {
	var value string

	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}

	enumTypeValue := ProviderKind(value)

	for _, existing := range ValidProviderKinds {
		if existing == enumTypeValue {
			*v = enumTypeValue
			return nil
		}
	}

	return fmt.Errorf("%+v is not a valid ProviderKind", value)
}

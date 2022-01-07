package api

import (
	"encoding/json"
	"fmt"
)

// DestinationKind the model 'DestinationKind'
type DestinationKind string

// List of DestinationKind
const (
	DESTINATIONKIND_KUBERNETES DestinationKind = "kubernetes"
)

var allowedDestinationKindEnumValues = []DestinationKind{
	"kubernetes",
}

func (v *DestinationKind) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	enumTypeValue := DestinationKind(value)
	for _, existing := range allowedDestinationKindEnumValues {
		if existing == enumTypeValue {
			*v = enumTypeValue
			return nil
		}
	}

	return fmt.Errorf("%+v is not a valid DestinationKind", value)
}

// IsValid return true if the value is valid for the enum, false otherwise
func (v DestinationKind) IsValid() bool {
	for _, existing := range allowedDestinationKindEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

package api

import (
	"encoding/json"
	"fmt"
)

// GrantKind the model 'GrantKind'
type GrantKind string

// List of GrantKind
const (
	GRANTKIND_KUBERNETES GrantKind = "kubernetes"
)

var allowedGrantKindEnumValues = []GrantKind{
	"kubernetes",
}

func (v *GrantKind) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	enumTypeValue := GrantKind(value)
	for _, existing := range allowedGrantKindEnumValues {
		if existing == enumTypeValue {
			*v = enumTypeValue
			return nil
		}
	}

	return fmt.Errorf("%+v is not a valid GrantKind", value)
}

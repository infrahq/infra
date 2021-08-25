/*
 * Infra API
 *
 * Infra REST API
 *
 * API version: 0.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package api

type Cred struct {
	Token string `json:"token"`

	Expires int64 `json:"expires"`
}

// AssertCredRequired checks if the required fields are not zero-ed
func AssertCredRequired(obj Cred) error {
	elements := map[string]interface{}{
		"token":   obj.Token,
		"expires": obj.Expires,
	}
	for name, el := range elements {
		if isZero := IsZeroValue(el); isZero {
			return &RequiredError{Field: name}
		}
	}

	return nil
}

// AssertRecurseCredRequired recursively checks if required fields are not zero-ed in a nested slice.
// Accepts only nested slice of Cred (e.g. [][]Cred), otherwise ErrTypeAssertionError is thrown.
func AssertRecurseCredRequired(objSlice interface{}) error {
	return AssertRecurseInterfaceRequired(objSlice, func(obj interface{}) error {
		aCred, ok := obj.(Cred)
		if !ok {
			return ErrTypeAssertionError
		}
		return AssertCredRequired(aCred)
	})
}

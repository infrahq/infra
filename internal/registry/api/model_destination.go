/*
 * Infra API
 *
 * Infra REST API
 *
 * API version: 0.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package api

type Destination struct {
	Id string `json:"id"`

	Name string `json:"name"`

	Created int64 `json:"created"`

	Updated int64 `json:"updated"`

	Kubernetes DestinationKubernetes `json:"kubernetes,omitempty"`
}

// AssertDestinationRequired checks if the required fields are not zero-ed
func AssertDestinationRequired(obj Destination) error {
	elements := map[string]interface{}{
		"id":      obj.Id,
		"name":    obj.Name,
		"created": obj.Created,
		"updated": obj.Updated,
	}
	for name, el := range elements {
		if isZero := IsZeroValue(el); isZero {
			return &RequiredError{Field: name}
		}
	}

	if err := AssertDestinationKubernetesRequired(obj.Kubernetes); err != nil {
		return err
	}
	return nil
}

// AssertRecurseDestinationRequired recursively checks if required fields are not zero-ed in a nested slice.
// Accepts only nested slice of Destination (e.g. [][]Destination), otherwise ErrTypeAssertionError is thrown.
func AssertRecurseDestinationRequired(objSlice interface{}) error {
	return AssertRecurseInterfaceRequired(objSlice, func(obj interface{}) error {
		aDestination, ok := obj.(Destination)
		if !ok {
			return ErrTypeAssertionError
		}
		return AssertDestinationRequired(aDestination)
	})
}

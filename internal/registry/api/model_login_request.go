/*
 * Infra API
 *
 * Infra REST API
 *
 * API version: 0.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package api

type LoginRequest struct {
	Infra LoginRequestInfra `json:"infra,omitempty"`

	Okta LoginRequestOkta `json:"okta,omitempty"`
}

// AssertLoginRequestRequired checks if the required fields are not zero-ed
func AssertLoginRequestRequired(obj LoginRequest) error {
	if err := AssertLoginRequestInfraRequired(obj.Infra); err != nil {
		return err
	}
	if err := AssertLoginRequestOktaRequired(obj.Okta); err != nil {
		return err
	}
	return nil
}

// AssertRecurseLoginRequestRequired recursively checks if required fields are not zero-ed in a nested slice.
// Accepts only nested slice of LoginRequest (e.g. [][]LoginRequest), otherwise ErrTypeAssertionError is thrown.
func AssertRecurseLoginRequestRequired(objSlice interface{}) error {
	return AssertRecurseInterfaceRequired(objSlice, func(obj interface{}) error {
		aLoginRequest, ok := obj.(LoginRequest)
		if !ok {
			return ErrTypeAssertionError
		}
		return AssertLoginRequestRequired(aLoginRequest)
	})
}

// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// Okta is an autogenerated mock type for the Okta type
type Okta struct {
	mock.Mock
}

// EmailFromCode provides a mock function with given fields: code, domain, clientID, clientSecret
func (_m *Okta) EmailFromCode(code string, domain string, clientID string, clientSecret string) (string, error) {
	ret := _m.Called(code, domain, clientID, clientSecret)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string, string, string) string); ok {
		r0 = rf(code, domain, clientID, clientSecret)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, string, string) error); ok {
		r1 = rf(code, domain, clientID, clientSecret)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Emails provides a mock function with given fields: domain, clientID, apiToken
func (_m *Okta) Emails(domain string, clientID string, apiToken string) ([]string, error) {
	ret := _m.Called(domain, clientID, apiToken)

	var r0 []string
	if rf, ok := ret.Get(0).(func(string, string, string) []string); ok {
		r0 = rf(domain, clientID, apiToken)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, string) error); ok {
		r1 = rf(domain, clientID, apiToken)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ValidateOktaConnection provides a mock function with given fields: domain, clientID, apiToken
func (_m *Okta) ValidateOktaConnection(domain string, clientID string, apiToken string) error {
	ret := _m.Called(domain, clientID, apiToken)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(domain, clientID, apiToken)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

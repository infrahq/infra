/*
Infra API

Infra REST API

API version: 0.1.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package client

import (
	"encoding/json"
)

// LoginRequest struct for LoginRequest
type LoginRequest struct {
	Infra *LoginRequestInfra `json:"infra,omitempty"`
	Okta  *LoginRequestOkta  `json:"okta,omitempty"`
}

// NewLoginRequest instantiates a new LoginRequest object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewLoginRequest() *LoginRequest {
	this := LoginRequest{}
	return &this
}

// NewLoginRequestWithDefaults instantiates a new LoginRequest object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewLoginRequestWithDefaults() *LoginRequest {
	this := LoginRequest{}
	return &this
}

// GetInfra returns the Infra field value if set, zero value otherwise.
func (o *LoginRequest) GetInfra() LoginRequestInfra {
	if o == nil || o.Infra == nil {
		var ret LoginRequestInfra
		return ret
	}
	return *o.Infra
}

// GetInfraOk returns a tuple with the Infra field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LoginRequest) GetInfraOk() (*LoginRequestInfra, bool) {
	if o == nil || o.Infra == nil {
		return nil, false
	}
	return o.Infra, true
}

// HasInfra returns a boolean if a field has been set.
func (o *LoginRequest) HasInfra() bool {
	if o != nil && o.Infra != nil {
		return true
	}

	return false
}

// SetInfra gets a reference to the given LoginRequestInfra and assigns it to the Infra field.
func (o *LoginRequest) SetInfra(v LoginRequestInfra) {
	o.Infra = &v
}

// GetOkta returns the Okta field value if set, zero value otherwise.
func (o *LoginRequest) GetOkta() LoginRequestOkta {
	if o == nil || o.Okta == nil {
		var ret LoginRequestOkta
		return ret
	}
	return *o.Okta
}

// GetOktaOk returns a tuple with the Okta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LoginRequest) GetOktaOk() (*LoginRequestOkta, bool) {
	if o == nil || o.Okta == nil {
		return nil, false
	}
	return o.Okta, true
}

// HasOkta returns a boolean if a field has been set.
func (o *LoginRequest) HasOkta() bool {
	if o != nil && o.Okta != nil {
		return true
	}

	return false
}

// SetOkta gets a reference to the given LoginRequestOkta and assigns it to the Okta field.
func (o *LoginRequest) SetOkta(v LoginRequestOkta) {
	o.Okta = &v
}

func (o LoginRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.Infra != nil {
		toSerialize["infra"] = o.Infra
	}
	if o.Okta != nil {
		toSerialize["okta"] = o.Okta
	}
	return json.Marshal(toSerialize)
}

type NullableLoginRequest struct {
	value *LoginRequest
	isSet bool
}

func (v NullableLoginRequest) Get() *LoginRequest {
	return v.value
}

func (v *NullableLoginRequest) Set(val *LoginRequest) {
	v.value = val
	v.isSet = true
}

func (v NullableLoginRequest) IsSet() bool {
	return v.isSet
}

func (v *NullableLoginRequest) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableLoginRequest(val *LoginRequest) *NullableLoginRequest {
	return &NullableLoginRequest{value: val, isSet: true}
}

func (v NullableLoginRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableLoginRequest) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

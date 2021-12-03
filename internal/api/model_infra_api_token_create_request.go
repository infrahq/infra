/*
Infra API

Infra REST API

API version: 0.1.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package api

import (
	"encoding/json"
)

// InfraAPITokenCreateRequest struct for InfraAPITokenCreateRequest
type InfraAPITokenCreateRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	// token time to live before expirry in the form XhYmZs, for example 1h30m
	Ttl *string `json:"ttl,omitempty"`
}

// NewInfraAPITokenCreateRequest instantiates a new InfraAPITokenCreateRequest object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewInfraAPITokenCreateRequest(name string, permissions []string) *InfraAPITokenCreateRequest {
	this := InfraAPITokenCreateRequest{}
	this.Name = name
	this.Permissions = permissions
	return &this
}

// NewInfraAPITokenCreateRequestWithDefaults instantiates a new InfraAPITokenCreateRequest object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewInfraAPITokenCreateRequestWithDefaults() *InfraAPITokenCreateRequest {
	this := InfraAPITokenCreateRequest{}
	return &this
}

// GetName returns the Name field value
func (o *InfraAPITokenCreateRequest) GetName() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Name
}

// GetNameOK returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *InfraAPITokenCreateRequest) GetNameOK() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value
func (o *InfraAPITokenCreateRequest) SetName(v string) {
	o.Name = v
}

// GetPermissions returns the Permissions field value
func (o *InfraAPITokenCreateRequest) GetPermissions() []string {
	if o == nil {
		var ret []string
		return ret
	}

	return o.Permissions
}

// GetPermissionsOK returns a tuple with the Permissions field value
// and a boolean to check if the value has been set.
func (o *InfraAPITokenCreateRequest) GetPermissionsOK() (*[]string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Permissions, true
}

// SetPermissions sets field value
func (o *InfraAPITokenCreateRequest) SetPermissions(v []string) {
	o.Permissions = v
}

// GetTtl returns the Ttl field value if set, zero value otherwise.
func (o *InfraAPITokenCreateRequest) GetTtl() string {
	if o == nil || o.Ttl == nil {
		var ret string
		return ret
	}
	return *o.Ttl
}

// GetTtlOK returns a tuple with the Ttl field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *InfraAPITokenCreateRequest) GetTtlOK() (*string, bool) {
	if o == nil || o.Ttl == nil {
		return nil, false
	}
	return o.Ttl, true
}

// HasTtl returns a boolean if a field has been set.
func (o *InfraAPITokenCreateRequest) HasTtl() bool {
	if o != nil && o.Ttl != nil {
		return true
	}

	return false
}

// SetTtl gets a reference to the given string and assigns it to the Ttl field.
func (o *InfraAPITokenCreateRequest) SetTtl(v string) {
	o.Ttl = &v
}

func (o InfraAPITokenCreateRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["name"] = o.Name
	}
	if true {
		toSerialize["permissions"] = o.Permissions
	}
	if o.Ttl != nil {
		toSerialize["ttl"] = o.Ttl
	}
	return json.Marshal(toSerialize)
}

type NullableInfraAPITokenCreateRequest struct {
	value *InfraAPITokenCreateRequest
	isSet bool
}

func (v NullableInfraAPITokenCreateRequest) Get() *InfraAPITokenCreateRequest {
	return v.value
}

func (v *NullableInfraAPITokenCreateRequest) Set(val *InfraAPITokenCreateRequest) {
	v.value = val
	v.isSet = true
}

func (v NullableInfraAPITokenCreateRequest) IsSet() bool {
	return v.isSet
}

func (v *NullableInfraAPITokenCreateRequest) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableInfraAPITokenCreateRequest(val *InfraAPITokenCreateRequest) *NullableInfraAPITokenCreateRequest {
	return &NullableInfraAPITokenCreateRequest{value: val, isSet: true}
}

func (v NullableInfraAPITokenCreateRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableInfraAPITokenCreateRequest) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

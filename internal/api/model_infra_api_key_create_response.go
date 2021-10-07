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

// InfraAPIKeyCreateResponse struct for InfraAPIKeyCreateResponse
type InfraAPIKeyCreateResponse struct {
	Key         string               `json:"key"`
	Id          string               `json:"id"`
	Created     int64                `json:"created"`
	Name        string               `json:"name"`
	Permissions []InfraAPIPermission `json:"permissions"`
}

// NewInfraAPIKeyCreateResponse instantiates a new InfraAPIKeyCreateResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewInfraAPIKeyCreateResponse(key string, id string, created int64, name string, permissions []InfraAPIPermission) *InfraAPIKeyCreateResponse {
	this := InfraAPIKeyCreateResponse{}
	this.Id = id
	this.Created = created
	this.Name = name
	this.Permissions = permissions
	return &this
}

// NewInfraAPIKeyCreateResponseWithDefaults instantiates a new InfraAPIKeyCreateResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewInfraAPIKeyCreateResponseWithDefaults() *InfraAPIKeyCreateResponse {
	this := InfraAPIKeyCreateResponse{}
	return &this
}

// GetKey returns the Key field value
func (o *InfraAPIKeyCreateResponse) GetKey() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Key
}

// GetKeyOk returns a tuple with the Key field value
// and a boolean to check if the value has been set.
func (o *InfraAPIKeyCreateResponse) GetKeyOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Key, true
}

// SetKey sets field value
func (o *InfraAPIKeyCreateResponse) SetKey(v string) {
	o.Key = v
}

// GetId returns the Id field value
func (o *InfraAPIKeyCreateResponse) GetId() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *InfraAPIKeyCreateResponse) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value
func (o *InfraAPIKeyCreateResponse) SetId(v string) {
	o.Id = v
}

// GetCreated returns the Created field value
func (o *InfraAPIKeyCreateResponse) GetCreated() int64 {
	if o == nil {
		var ret int64
		return ret
	}

	return o.Created
}

// GetCreatedOk returns a tuple with the Created field value
// and a boolean to check if the value has been set.
func (o *InfraAPIKeyCreateResponse) GetCreatedOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Created, true
}

// SetCreated sets field value
func (o *InfraAPIKeyCreateResponse) SetCreated(v int64) {
	o.Created = v
}

// GetName returns the Name field value
func (o *InfraAPIKeyCreateResponse) GetName() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *InfraAPIKeyCreateResponse) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value
func (o *InfraAPIKeyCreateResponse) SetName(v string) {
	o.Name = v
}

// GetPermissions returns the Permissions field value
func (o *InfraAPIKeyCreateResponse) GetPermissions() []InfraAPIPermission {
	if o == nil {
		var ret []InfraAPIPermission
		return ret
	}

	return o.Permissions
}

// GetPermissionsOk returns a tuple with the Permissions field value
// and a boolean to check if the value has been set.
func (o *InfraAPIKeyCreateResponse) GetPermissionsOk() (*[]InfraAPIPermission, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Permissions, true
}

// SetPermissions sets field value
func (o *InfraAPIKeyCreateResponse) SetPermissions(v []InfraAPIPermission) {
	o.Permissions = v
}

func (o InfraAPIKeyCreateResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["key"] = o.Key
	}
	if true {
		toSerialize["id"] = o.Id
	}
	if true {
		toSerialize["created"] = o.Created
	}
	if true {
		toSerialize["name"] = o.Name
	}
	if true {
		toSerialize["permissions"] = o.Permissions
	}
	return json.Marshal(toSerialize)
}

type NullableInfraAPIKeyCreateResponse struct {
	value *InfraAPIKeyCreateResponse
	isSet bool
}

func (v NullableInfraAPIKeyCreateResponse) Get() *InfraAPIKeyCreateResponse {
	return v.value
}

func (v *NullableInfraAPIKeyCreateResponse) Set(val *InfraAPIKeyCreateResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableInfraAPIKeyCreateResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableInfraAPIKeyCreateResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableInfraAPIKeyCreateResponse(val *InfraAPIKeyCreateResponse) *NullableInfraAPIKeyCreateResponse {
	return &NullableInfraAPIKeyCreateResponse{value: val, isSet: true}
}

func (v NullableInfraAPIKeyCreateResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableInfraAPIKeyCreateResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

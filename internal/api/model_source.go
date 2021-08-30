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

// Source struct for Source
type Source struct {
	Id      string      `json:"id"`
	Created int64       `json:"created"`
	Updated int64       `json:"updated"`
	Okta    *SourceOkta `json:"okta,omitempty"`
}

// NewSource instantiates a new Source object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewSource(id string, created int64, updated int64) *Source {
	this := Source{}
	this.Id = id
	this.Created = created
	this.Updated = updated
	return &this
}

// NewSourceWithDefaults instantiates a new Source object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewSourceWithDefaults() *Source {
	this := Source{}
	return &this
}

// GetId returns the Id field value
func (o *Source) GetId() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *Source) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value
func (o *Source) SetId(v string) {
	o.Id = v
}

// GetCreated returns the Created field value
func (o *Source) GetCreated() int64 {
	if o == nil {
		var ret int64
		return ret
	}

	return o.Created
}

// GetCreatedOk returns a tuple with the Created field value
// and a boolean to check if the value has been set.
func (o *Source) GetCreatedOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Created, true
}

// SetCreated sets field value
func (o *Source) SetCreated(v int64) {
	o.Created = v
}

// GetUpdated returns the Updated field value
func (o *Source) GetUpdated() int64 {
	if o == nil {
		var ret int64
		return ret
	}

	return o.Updated
}

// GetUpdatedOk returns a tuple with the Updated field value
// and a boolean to check if the value has been set.
func (o *Source) GetUpdatedOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Updated, true
}

// SetUpdated sets field value
func (o *Source) SetUpdated(v int64) {
	o.Updated = v
}

// GetOkta returns the Okta field value if set, zero value otherwise.
func (o *Source) GetOkta() SourceOkta {
	if o == nil || o.Okta == nil {
		var ret SourceOkta
		return ret
	}
	return *o.Okta
}

// GetOktaOk returns a tuple with the Okta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Source) GetOktaOk() (*SourceOkta, bool) {
	if o == nil || o.Okta == nil {
		return nil, false
	}
	return o.Okta, true
}

// HasOkta returns a boolean if a field has been set.
func (o *Source) HasOkta() bool {
	if o != nil && o.Okta != nil {
		return true
	}

	return false
}

// SetOkta gets a reference to the given SourceOkta and assigns it to the Okta field.
func (o *Source) SetOkta(v SourceOkta) {
	o.Okta = &v
}

func (o Source) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["id"] = o.Id
	}
	if true {
		toSerialize["created"] = o.Created
	}
	if true {
		toSerialize["updated"] = o.Updated
	}
	if o.Okta != nil {
		toSerialize["okta"] = o.Okta
	}
	return json.Marshal(toSerialize)
}

type NullableSource struct {
	value *Source
	isSet bool
}

func (v NullableSource) Get() *Source {
	return v.value
}

func (v *NullableSource) Set(val *Source) {
	v.value = val
	v.isSet = true
}

func (v NullableSource) IsSet() bool {
	return v.isSet
}

func (v *NullableSource) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableSource(val *Source) *NullableSource {
	return &NullableSource{value: val, isSet: true}
}

func (v NullableSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableSource) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

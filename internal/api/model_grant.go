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

// Grant struct for Grant
type Grant struct {
	ID string `json:"id"`
	// created time in seconds since 1970-01-01
	Created int64 `json:"created"`
	// updated time in seconds since 1970-01-01
	Updated     int64            `json:"updated"`
	Kind        GrantKind        `json:"kind"`
	Destination Destination      `json:"destination"`
	Kubernetes  *GrantKubernetes `json:"kubernetes,omitempty"`
	Users       *[]User          `json:"users,omitempty"`
	Groups      *[]Group         `json:"groups,omitempty"`
}

// NewGrant instantiates a new Grant object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewGrant(id string, created int64, updated int64, kind GrantKind, destination Destination) *Grant {
	this := Grant{}
	this.ID = id
	this.Created = created
	this.Updated = updated
	this.Kind = kind
	this.Destination = destination
	return &this
}

// NewGrantWithDefaults instantiates a new Grant object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewGrantWithDefaults() *Grant {
	this := Grant{}
	return &this
}

// GetID returns the ID field value
func (o *Grant) GetID() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.ID
}

// GetIDOK returns a tuple with the ID field value
// and a boolean to check if the value has been set.
func (o *Grant) GetIDOK() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ID, true
}

// SetID sets field value
func (o *Grant) SetID(v string) {
	o.ID = v
}

// GetCreated returns the Created field value
func (o *Grant) GetCreated() int64 {
	if o == nil {
		var ret int64
		return ret
	}

	return o.Created
}

// GetCreatedOK returns a tuple with the Created field value
// and a boolean to check if the value has been set.
func (o *Grant) GetCreatedOK() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Created, true
}

// SetCreated sets field value
func (o *Grant) SetCreated(v int64) {
	o.Created = v
}

// GetUpdated returns the Updated field value
func (o *Grant) GetUpdated() int64 {
	if o == nil {
		var ret int64
		return ret
	}

	return o.Updated
}

// GetUpdatedOK returns a tuple with the Updated field value
// and a boolean to check if the value has been set.
func (o *Grant) GetUpdatedOK() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Updated, true
}

// SetUpdated sets field value
func (o *Grant) SetUpdated(v int64) {
	o.Updated = v
}

// GetKind returns the Kind field value
func (o *Grant) GetKind() GrantKind {
	if o == nil {
		var ret GrantKind
		return ret
	}

	return o.Kind
}

// GetKindOK returns a tuple with the Kind field value
// and a boolean to check if the value has been set.
func (o *Grant) GetKindOK() (*GrantKind, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Kind, true
}

// SetKind sets field value
func (o *Grant) SetKind(v GrantKind) {
	o.Kind = v
}

// GetDestination returns the Destination field value
func (o *Grant) GetDestination() Destination {
	if o == nil {
		var ret Destination
		return ret
	}

	return o.Destination
}

// GetDestinationOK returns a tuple with the Destination field value
// and a boolean to check if the value has been set.
func (o *Grant) GetDestinationOK() (*Destination, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Destination, true
}

// SetDestination sets field value
func (o *Grant) SetDestination(v Destination) {
	o.Destination = v
}

// GetKubernetes returns the Kubernetes field value if set, zero value otherwise.
func (o *Grant) GetKubernetes() GrantKubernetes {
	if o == nil || o.Kubernetes == nil {
		var ret GrantKubernetes
		return ret
	}
	return *o.Kubernetes
}

// GetKubernetesOK returns a tuple with the Kubernetes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Grant) GetKubernetesOK() (*GrantKubernetes, bool) {
	if o == nil || o.Kubernetes == nil {
		return nil, false
	}
	return o.Kubernetes, true
}

// HasKubernetes returns a boolean if a field has been set.
func (o *Grant) HasKubernetes() bool {
	if o != nil && o.Kubernetes != nil {
		return true
	}

	return false
}

// SetKubernetes gets a reference to the given GrantKubernetes and assigns it to the Kubernetes field.
func (o *Grant) SetKubernetes(v GrantKubernetes) {
	o.Kubernetes = &v
}

// GetUsers returns the Users field value if set, zero value otherwise.
func (o *Grant) GetUsers() []User {
	if o == nil || o.Users == nil {
		var ret []User
		return ret
	}
	return *o.Users
}

// GetUsersOK returns a tuple with the Users field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Grant) GetUsersOK() (*[]User, bool) {
	if o == nil || o.Users == nil {
		return nil, false
	}
	return o.Users, true
}

// HasUsers returns a boolean if a field has been set.
func (o *Grant) HasUsers() bool {
	if o != nil && o.Users != nil {
		return true
	}

	return false
}

// SetUsers gets a reference to the given []User and assigns it to the Users field.
func (o *Grant) SetUsers(v []User) {
	o.Users = &v
}

// GetGroups returns the Groups field value if set, zero value otherwise.
func (o *Grant) GetGroups() []Group {
	if o == nil || o.Groups == nil {
		var ret []Group
		return ret
	}
	return *o.Groups
}

// GetGroupsOK returns a tuple with the Groups field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Grant) GetGroupsOK() (*[]Group, bool) {
	if o == nil || o.Groups == nil {
		return nil, false
	}
	return o.Groups, true
}

// HasGroups returns a boolean if a field has been set.
func (o *Grant) HasGroups() bool {
	if o != nil && o.Groups != nil {
		return true
	}

	return false
}

// SetGroups gets a reference to the given []Group and assigns it to the Groups field.
func (o *Grant) SetGroups(v []Group) {
	o.Groups = &v
}

func (o Grant) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["id"] = o.ID
	}
	if true {
		toSerialize["created"] = o.Created
	}
	if true {
		toSerialize["updated"] = o.Updated
	}
	if true {
		toSerialize["kind"] = o.Kind
	}
	if true {
		toSerialize["destination"] = o.Destination
	}
	if o.Kubernetes != nil {
		toSerialize["kubernetes"] = o.Kubernetes
	}
	if o.Users != nil {
		toSerialize["users"] = o.Users
	}
	if o.Groups != nil {
		toSerialize["groups"] = o.Groups
	}
	return json.Marshal(toSerialize)
}

type NullableGrant struct {
	value *Grant
	isSet bool
}

func (v NullableGrant) Get() *Grant {
	return v.value
}

func (v *NullableGrant) Set(val *Grant) {
	v.value = val
	v.isSet = true
}

func (v NullableGrant) IsSet() bool {
	return v.isSet
}

func (v *NullableGrant) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableGrant(val *Grant) *NullableGrant {
	return &NullableGrant{value: val, isSet: true}
}

func (v NullableGrant) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableGrant) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

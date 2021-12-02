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

// Role struct for Role
type Role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// created time in seconds since 1970-01-01
	Created int64 `json:"created"`
	// updated time in seconds since 1970-01-01
	Updated     int64       `json:"updated"`
	Kind        RoleKind    `json:"kind"`
	Namespace   string      `json:"namespace"`
	Users       *[]User     `json:"users,omitempty"`
	Groups      *[]Group    `json:"groups,omitempty"`
	Destination Destination `json:"destination"`
}

// NewRole instantiates a new Role object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewRole(id string, name string, created int64, updated int64, kind RoleKind, namespace string, destination Destination) *Role {
	this := Role{}
	this.ID = id
	this.Name = name
	this.Created = created
	this.Updated = updated
	this.Kind = kind
	this.Namespace = namespace
	this.Destination = destination
	return &this
}

// NewRoleWithDefaults instantiates a new Role object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewRoleWithDefaults() *Role {
	this := Role{}
	var kind RoleKind = ROLEKIND_ROLE
	this.Kind = kind
	return &this
}

// GetID returns the ID field value
func (o *Role) GetID() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.ID
}

// GetIDOK returns a tuple with the ID field value
// and a boolean to check if the value has been set.
func (o *Role) GetIDOK() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ID, true
}

// SetID sets field value
func (o *Role) SetID(v string) {
	o.ID = v
}

// GetName returns the Name field value
func (o *Role) GetName() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Name
}

// GetNameOK returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *Role) GetNameOK() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value
func (o *Role) SetName(v string) {
	o.Name = v
}

// GetCreated returns the Created field value
func (o *Role) GetCreated() int64 {
	if o == nil {
		var ret int64
		return ret
	}

	return o.Created
}

// GetCreatedOK returns a tuple with the Created field value
// and a boolean to check if the value has been set.
func (o *Role) GetCreatedOK() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Created, true
}

// SetCreated sets field value
func (o *Role) SetCreated(v int64) {
	o.Created = v
}

// GetUpdated returns the Updated field value
func (o *Role) GetUpdated() int64 {
	if o == nil {
		var ret int64
		return ret
	}

	return o.Updated
}

// GetUpdatedOK returns a tuple with the Updated field value
// and a boolean to check if the value has been set.
func (o *Role) GetUpdatedOK() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Updated, true
}

// SetUpdated sets field value
func (o *Role) SetUpdated(v int64) {
	o.Updated = v
}

// GetKind returns the Kind field value
func (o *Role) GetKind() RoleKind {
	if o == nil {
		var ret RoleKind
		return ret
	}

	return o.Kind
}

// GetKindOK returns a tuple with the Kind field value
// and a boolean to check if the value has been set.
func (o *Role) GetKindOK() (*RoleKind, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Kind, true
}

// SetKind sets field value
func (o *Role) SetKind(v RoleKind) {
	o.Kind = v
}

// GetNamespace returns the Namespace field value
func (o *Role) GetNamespace() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Namespace
}

// GetNamespaceOK returns a tuple with the Namespace field value
// and a boolean to check if the value has been set.
func (o *Role) GetNamespaceOK() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Namespace, true
}

// SetNamespace sets field value
func (o *Role) SetNamespace(v string) {
	o.Namespace = v
}

// GetUsers returns the Users field value if set, zero value otherwise.
func (o *Role) GetUsers() []User {
	if o == nil || o.Users == nil {
		var ret []User
		return ret
	}
	return *o.Users
}

// GetUsersOK returns a tuple with the Users field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Role) GetUsersOK() (*[]User, bool) {
	if o == nil || o.Users == nil {
		return nil, false
	}
	return o.Users, true
}

// HasUsers returns a boolean if a field has been set.
func (o *Role) HasUsers() bool {
	if o != nil && o.Users != nil {
		return true
	}

	return false
}

// SetUsers gets a reference to the given []User and assigns it to the Users field.
func (o *Role) SetUsers(v []User) {
	o.Users = &v
}

// GetGroups returns the Groups field value if set, zero value otherwise.
func (o *Role) GetGroups() []Group {
	if o == nil || o.Groups == nil {
		var ret []Group
		return ret
	}
	return *o.Groups
}

// GetGroupsOK returns a tuple with the Groups field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Role) GetGroupsOK() (*[]Group, bool) {
	if o == nil || o.Groups == nil {
		return nil, false
	}
	return o.Groups, true
}

// HasGroups returns a boolean if a field has been set.
func (o *Role) HasGroups() bool {
	if o != nil && o.Groups != nil {
		return true
	}

	return false
}

// SetGroups gets a reference to the given []Group and assigns it to the Groups field.
func (o *Role) SetGroups(v []Group) {
	o.Groups = &v
}

// GetDestination returns the Destination field value
func (o *Role) GetDestination() Destination {
	if o == nil {
		var ret Destination
		return ret
	}

	return o.Destination
}

// GetDestinationOK returns a tuple with the Destination field value
// and a boolean to check if the value has been set.
func (o *Role) GetDestinationOK() (*Destination, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Destination, true
}

// SetDestination sets field value
func (o *Role) SetDestination(v Destination) {
	o.Destination = v
}

func (o Role) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["id"] = o.ID
	}
	if true {
		toSerialize["name"] = o.Name
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
		toSerialize["namespace"] = o.Namespace
	}
	if o.Users != nil {
		toSerialize["users"] = o.Users
	}
	if o.Groups != nil {
		toSerialize["groups"] = o.Groups
	}
	if true {
		toSerialize["destination"] = o.Destination
	}
	return json.Marshal(toSerialize)
}

type NullableRole struct {
	value *Role
	isSet bool
}

func (v NullableRole) Get() *Role {
	return v.value
}

func (v *NullableRole) Set(val *Role) {
	v.value = val
	v.isSet = true
}

func (v NullableRole) IsSet() bool {
	return v.isSet
}

func (v *NullableRole) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableRole(val *Role) *NullableRole {
	return &NullableRole{value: val, isSet: true}
}

func (v NullableRole) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableRole) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

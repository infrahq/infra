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

// Destination struct for Destination
type Destination struct {
	ID     string          `json:"id"`
	NodeID string          `json:"nodeID"`
	Name   string          `json:"name"`
	Kind   DestinationKind `json:"kind"`
	// created time in seconds since 1970-01-01
	Created int64 `json:"created"`
	// updated time in seconds since 1970-01-01
	Updated    int64                  `json:"updated"`
	Labels     []string               `json:"labels"`
	Kubernetes *DestinationKubernetes `json:"kubernetes,omitempty"`
}

// NewDestination instantiates a new Destination object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewDestination(id string, nodeID string, name string, kind DestinationKind, created int64, updated int64, labels []string) *Destination {
	this := Destination{}
	this.ID = id
	this.NodeID = nodeID
	this.Name = name
	this.Kind = kind
	this.Created = created
	this.Updated = updated
	this.Labels = labels
	return &this
}

// NewDestinationWithDefaults instantiates a new Destination object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewDestinationWithDefaults() *Destination {
	this := Destination{}
	return &this
}

// GetID returns the ID field value
func (o *Destination) GetID() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.ID
}

// GetIDOK returns a tuple with the ID field value
// and a boolean to check if the value has been set.
func (o *Destination) GetIDOK() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ID, true
}

// SetID sets field value
func (o *Destination) SetID(v string) {
	o.ID = v
}

// GetNodeID returns the NodeID field value
func (o *Destination) GetNodeID() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.NodeID
}

// GetNodeIDOK returns a tuple with the NodeID field value
// and a boolean to check if the value has been set.
func (o *Destination) GetNodeIDOK() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.NodeID, true
}

// SetNodeID sets field value
func (o *Destination) SetNodeID(v string) {
	o.NodeID = v
}

// GetName returns the Name field value
func (o *Destination) GetName() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Name
}

// GetNameOK returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *Destination) GetNameOK() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value
func (o *Destination) SetName(v string) {
	o.Name = v
}

// GetKind returns the Kind field value
func (o *Destination) GetKind() DestinationKind {
	if o == nil {
		var ret DestinationKind
		return ret
	}

	return o.Kind
}

// GetKindOK returns a tuple with the Kind field value
// and a boolean to check if the value has been set.
func (o *Destination) GetKindOK() (*DestinationKind, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Kind, true
}

// SetKind sets field value
func (o *Destination) SetKind(v DestinationKind) {
	o.Kind = v
}

// GetCreated returns the Created field value
func (o *Destination) GetCreated() int64 {
	if o == nil {
		var ret int64
		return ret
	}

	return o.Created
}

// GetCreatedOK returns a tuple with the Created field value
// and a boolean to check if the value has been set.
func (o *Destination) GetCreatedOK() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Created, true
}

// SetCreated sets field value
func (o *Destination) SetCreated(v int64) {
	o.Created = v
}

// GetUpdated returns the Updated field value
func (o *Destination) GetUpdated() int64 {
	if o == nil {
		var ret int64
		return ret
	}

	return o.Updated
}

// GetUpdatedOK returns a tuple with the Updated field value
// and a boolean to check if the value has been set.
func (o *Destination) GetUpdatedOK() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Updated, true
}

// SetUpdated sets field value
func (o *Destination) SetUpdated(v int64) {
	o.Updated = v
}

// GetLabels returns the Labels field value
func (o *Destination) GetLabels() []string {
	if o == nil {
		var ret []string
		return ret
	}

	return o.Labels
}

// GetLabelsOK returns a tuple with the Labels field value
// and a boolean to check if the value has been set.
func (o *Destination) GetLabelsOK() (*[]string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Labels, true
}

// SetLabels sets field value
func (o *Destination) SetLabels(v []string) {
	o.Labels = v
}

// GetKubernetes returns the Kubernetes field value if set, zero value otherwise.
func (o *Destination) GetKubernetes() DestinationKubernetes {
	if o == nil || o.Kubernetes == nil {
		var ret DestinationKubernetes
		return ret
	}
	return *o.Kubernetes
}

// GetKubernetesOK returns a tuple with the Kubernetes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Destination) GetKubernetesOK() (*DestinationKubernetes, bool) {
	if o == nil || o.Kubernetes == nil {
		return nil, false
	}
	return o.Kubernetes, true
}

// HasKubernetes returns a boolean if a field has been set.
func (o *Destination) HasKubernetes() bool {
	if o != nil && o.Kubernetes != nil {
		return true
	}

	return false
}

// SetKubernetes gets a reference to the given DestinationKubernetes and assigns it to the Kubernetes field.
func (o *Destination) SetKubernetes(v DestinationKubernetes) {
	o.Kubernetes = &v
}

func (o Destination) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["id"] = o.ID
	}
	if true {
		toSerialize["nodeID"] = o.NodeID
	}
	if true {
		toSerialize["name"] = o.Name
	}
	if true {
		toSerialize["kind"] = o.Kind
	}
	if true {
		toSerialize["created"] = o.Created
	}
	if true {
		toSerialize["updated"] = o.Updated
	}
	if true {
		toSerialize["labels"] = o.Labels
	}
	if o.Kubernetes != nil {
		toSerialize["kubernetes"] = o.Kubernetes
	}
	return json.Marshal(toSerialize)
}

type NullableDestination struct {
	value *Destination
	isSet bool
}

func (v NullableDestination) Get() *Destination {
	return v.value
}

func (v *NullableDestination) Set(val *Destination) {
	v.value = val
	v.isSet = true
}

func (v NullableDestination) IsSet() bool {
	return v.isSet
}

func (v *NullableDestination) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableDestination(val *Destination) *NullableDestination {
	return &NullableDestination{value: val, isSet: true}
}

func (v NullableDestination) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableDestination) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

# DestinationCreateRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** |  | 
**Kubernetes** | Pointer to [**DestinationKubernetes**](DestinationKubernetes.md) |  | [optional] 

## Methods

### NewDestinationCreateRequest

`func NewDestinationCreateRequest(name string, ) *DestinationCreateRequest`

NewDestinationCreateRequest instantiates a new DestinationCreateRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDestinationCreateRequestWithDefaults

`func NewDestinationCreateRequestWithDefaults() *DestinationCreateRequest`

NewDestinationCreateRequestWithDefaults instantiates a new DestinationCreateRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *DestinationCreateRequest) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *DestinationCreateRequest) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *DestinationCreateRequest) SetName(v string)`

SetName sets Name field to given value.


### GetKubernetes

`func (o *DestinationCreateRequest) GetKubernetes() DestinationKubernetes`

GetKubernetes returns the Kubernetes field if non-nil, zero value otherwise.

### GetKubernetesOk

`func (o *DestinationCreateRequest) GetKubernetesOk() (*DestinationKubernetes, bool)`

GetKubernetesOk returns a tuple with the Kubernetes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKubernetes

`func (o *DestinationCreateRequest) SetKubernetes(v DestinationKubernetes)`

SetKubernetes sets Kubernetes field to given value.

### HasKubernetes

`func (o *DestinationCreateRequest) HasKubernetes() bool`

HasKubernetes returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)



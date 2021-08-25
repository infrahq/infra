# Source

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **string** |  | 
**Created** | **int64** |  | 
**Updated** | **int64** |  | 
**Okta** | Pointer to [**SourceOkta**](SourceOkta.md) |  | [optional] 

## Methods

### NewSource

`func NewSource(id string, created int64, updated int64, ) *Source`

NewSource instantiates a new Source object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSourceWithDefaults

`func NewSourceWithDefaults() *Source`

NewSourceWithDefaults instantiates a new Source object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *Source) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *Source) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *Source) SetId(v string)`

SetId sets Id field to given value.


### GetCreated

`func (o *Source) GetCreated() int64`

GetCreated returns the Created field if non-nil, zero value otherwise.

### GetCreatedOk

`func (o *Source) GetCreatedOk() (*int64, bool)`

GetCreatedOk returns a tuple with the Created field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreated

`func (o *Source) SetCreated(v int64)`

SetCreated sets Created field to given value.


### GetUpdated

`func (o *Source) GetUpdated() int64`

GetUpdated returns the Updated field if non-nil, zero value otherwise.

### GetUpdatedOk

`func (o *Source) GetUpdatedOk() (*int64, bool)`

GetUpdatedOk returns a tuple with the Updated field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdated

`func (o *Source) SetUpdated(v int64)`

SetUpdated sets Updated field to given value.


### GetOkta

`func (o *Source) GetOkta() SourceOkta`

GetOkta returns the Okta field if non-nil, zero value otherwise.

### GetOktaOk

`func (o *Source) GetOktaOk() (*SourceOkta, bool)`

GetOktaOk returns a tuple with the Okta field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOkta

`func (o *Source) SetOkta(v SourceOkta)`

SetOkta sets Okta field to given value.

### HasOkta

`func (o *Source) HasOkta() bool`

HasOkta returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)



# LoginRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Infra** | Pointer to [**LoginRequestInfra**](LoginRequestInfra.md) |  | [optional] 
**Okta** | Pointer to [**LoginRequestOkta**](LoginRequestOkta.md) |  | [optional] 

## Methods

### NewLoginRequest

`func NewLoginRequest() *LoginRequest`

NewLoginRequest instantiates a new LoginRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewLoginRequestWithDefaults

`func NewLoginRequestWithDefaults() *LoginRequest`

NewLoginRequestWithDefaults instantiates a new LoginRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetInfra

`func (o *LoginRequest) GetInfra() LoginRequestInfra`

GetInfra returns the Infra field if non-nil, zero value otherwise.

### GetInfraOk

`func (o *LoginRequest) GetInfraOk() (*LoginRequestInfra, bool)`

GetInfraOk returns a tuple with the Infra field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInfra

`func (o *LoginRequest) SetInfra(v LoginRequestInfra)`

SetInfra sets Infra field to given value.

### HasInfra

`func (o *LoginRequest) HasInfra() bool`

HasInfra returns a boolean if a field has been set.

### GetOkta

`func (o *LoginRequest) GetOkta() LoginRequestOkta`

GetOkta returns the Okta field if non-nil, zero value otherwise.

### GetOktaOk

`func (o *LoginRequest) GetOktaOk() (*LoginRequestOkta, bool)`

GetOktaOk returns a tuple with the Okta field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOkta

`func (o *LoginRequest) SetOkta(v LoginRequestOkta)`

SetOkta sets Okta field to given value.

### HasOkta

`func (o *LoginRequest) HasOkta() bool`

HasOkta returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)



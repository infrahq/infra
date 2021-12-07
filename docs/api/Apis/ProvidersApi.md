# ProvidersApi

All URIs are relative to *http://localhost/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getProvider**](ProvidersApi.md#getProvider) | **GET** /providers/{id} | Get provider by ID
[**listProviders**](ProvidersApi.md#listProviders) | **GET** /providers | List providers


<a name="getProvider"></a>
# **getProvider**
> Provider getProvider(id)

Get provider by ID

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | [**UUID**](../Models/.md)| Provider ID | [default to null]

### Return type

[**Provider**](../Models/Provider.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

<a name="listProviders"></a>
# **listProviders**
> List listProviders(kind)

List providers

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **kind** | [**ProviderKind**](../Models/.md)| Filter providers by kind | [optional] [default to null] [enum: okta]

### Return type

[**List**](../Models/Provider.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


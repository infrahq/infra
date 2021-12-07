# ApiKeysApi

All URIs are relative to *http://localhost/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**createAPIKey**](ApiKeysApi.md#createAPIKey) | **POST** /api-keys | Create an API key
[**deleteAPIKey**](ApiKeysApi.md#deleteAPIKey) | **DELETE** /api-keys/{id} | Delete an API key
[**listAPIKeys**](ApiKeysApi.md#listAPIKeys) | **GET** /api-keys | List API keys


<a name="createAPIKey"></a>
# **createAPIKey**
> InfraAPIKeyCreateResponse createAPIKey(body)

Create an API key

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**InfraAPIKeyCreateRequest**](../Models/InfraAPIKeyCreateRequest.md)|  |

### Return type

[**InfraAPIKeyCreateResponse**](../Models/InfraAPIKeyCreateResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

<a name="deleteAPIKey"></a>
# **deleteAPIKey**
> deleteAPIKey(id)

Delete an API key

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | [**UUID**](../Models/.md)| API key ID | [default to null]

### Return type

null (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

<a name="listAPIKeys"></a>
# **listAPIKeys**
> List listAPIKeys(name)

List API keys

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **String**| Filter results by the API key name | [optional] [default to null]

### Return type

[**List**](../Models/InfraAPIKey.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


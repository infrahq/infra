# GrantsApi

All URIs are relative to *http://localhost/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getGrant**](GrantsApi.md#getGrant) | **GET** /grants/{id} | Get grant
[**listGrants**](GrantsApi.md#listGrants) | **GET** /grants | List grants


<a name="getGrant"></a>
# **getGrant**
> Grant getGrant(id)

Get grant

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | [**UUID**](../Models/.md)| Grant ID | [default to null]

### Return type

[**Grant**](../Models/Grant.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

<a name="listGrants"></a>
# **listGrants**
> List listGrants(name, kind, destination)

List grants

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **String**| Filter results by name | [optional] [default to null]
 **kind** | [**GrantKind**](../Models/.md)| Filter results by kind | [optional] [default to null] [enum: kubernetes]
 **destination** | **String**| Filter results by destination | [optional] [default to null]

### Return type

[**List**](../Models/Grant.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


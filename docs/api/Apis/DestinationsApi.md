# DestinationsApi

All URIs are relative to *http://localhost/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**createDestination**](DestinationsApi.md#createDestination) | **POST** /destinations | Create a destination
[**getDestination**](DestinationsApi.md#getDestination) | **GET** /destinations/{id} | Get destination by ID
[**listDestinations**](DestinationsApi.md#listDestinations) | **GET** /destinations | List destinations


<a name="createDestination"></a>
# **createDestination**
> Destination createDestination(body)

Create a destination

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**DestinationCreateRequest**](../Models/DestinationCreateRequest.md)|  |

### Return type

[**Destination**](../Models/Destination.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

<a name="getDestination"></a>
# **getDestination**
> Destination getDestination(id)

Get destination by ID

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | [**UUID**](../Models/.md)| Destination ID | [default to null]

### Return type

[**Destination**](../Models/Destination.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

<a name="listDestinations"></a>
# **listDestinations**
> List listDestinations(name, kind)

List destinations

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **String**| Filter destinations by name | [optional] [default to null]
 **kind** | [**DestinationKind**](../Models/.md)| Filter destinations by kind | [optional] [default to null] [enum: kubernetes]

### Return type

[**List**](../Models/Destination.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


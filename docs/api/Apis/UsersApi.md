# UsersApi

All URIs are relative to *http://localhost/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getUser**](UsersApi.md#getUser) | **GET** /users/{id} | Get user by ID
[**listUsers**](UsersApi.md#listUsers) | **GET** /users | List users


<a name="getUser"></a>
# **getUser**
> User getUser(id)

Get user by ID

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | [**UUID**](../Models/.md)| User ID | [default to null]

### Return type

[**User**](../Models/User.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

<a name="listUsers"></a>
# **listUsers**
> List listUsers(email)

List users

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **email** | [**String**](../Models/.md)| Filter results by user email | [optional] [default to null]

### Return type

[**List**](../Models/User.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


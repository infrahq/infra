# AuthApi

All URIs are relative to *http://localhost/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**login**](AuthApi.md#login) | **POST** /login | Login to Infra
[**logout**](AuthApi.md#logout) | **POST** /logout | Logout of Infra


<a name="login"></a>
# **login**
> LoginResponse login(body)

Login to Infra

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**LoginRequest**](../Models/LoginRequest.md)|  |

### Return type

[**LoginResponse**](../Models/LoginResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

<a name="logout"></a>
# **logout**
> logout()

Logout of Infra

### Parameters
This endpoint does not need any parameter.

### Return type

null (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


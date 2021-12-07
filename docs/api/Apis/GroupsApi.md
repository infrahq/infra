# GroupsApi

All URIs are relative to *http://localhost/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getGroup**](GroupsApi.md#getGroup) | **GET** /groups/{id} | Get group by ID
[**listGroups**](GroupsApi.md#listGroups) | **GET** /groups | List groups


<a name="getGroup"></a>
# **getGroup**
> Group getGroup(id)

Get group by ID

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | [**UUID**](../Models/.md)| Group ID | [default to null]

### Return type

[**Group**](../Models/Group.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

<a name="listGroups"></a>
# **listGroups**
> List listGroups(name, active)

List groups

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **String**| Filter groups by name | [optional] [default to null]
 **active** | **Boolean**| Filter groups by active state | [optional] [default to null]

### Return type

[**List**](../Models/Group.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


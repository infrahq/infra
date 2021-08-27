# .DestinationsApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**createDestination**](DestinationsApi.md#createDestination) | **POST** /destinations | Register a destination
[**listDestinations**](DestinationsApi.md#listDestinations) | **GET** /destinations | List destinations


# **createDestination**
> Destination createDestination(body)


### Example


```typescript
import {  } from '';
import * as fs from 'fs';

const configuration = .createConfiguration();
const apiInstance = new .DestinationsApi(configuration);

let body:.DestinationsApiCreateDestinationRequest = {
  // DestinationCreateRequest
  body: {
    name: "name_example",
    kubernetes: {
      ca: "ca_example",
      endpoint: "endpoint_example",
      namespace: "namespace_example",
      saToken: "saToken_example",
    },
  },
};

apiInstance.createDestination(body).then((data:any) => {
  console.log('API called successfully. Returned data: ' + data);
}).catch((error:any) => console.error(error));
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | **DestinationCreateRequest**|  |


### Return type

**Destination**

### Authorization

[bearerAuth](README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | create destination response |  -  |
**0** | unexpected error |  -  |

[[Back to top]](#) [[Back to API list]](README.md#documentation-for-api-endpoints) [[Back to Model list]](README.md#documentation-for-models) [[Back to README]](README.md)

# **listDestinations**
> Array<Destination> listDestinations()


### Example


```typescript
import {  } from '';
import * as fs from 'fs';

const configuration = .createConfiguration();
const apiInstance = new .DestinationsApi(configuration);

let body:any = {};

apiInstance.listDestinations(body).then((data:any) => {
  console.log('API called successfully. Returned data: ' + data);
}).catch((error:any) => console.error(error));
```


### Parameters
This endpoint does not need any parameter.


### Return type

**Array<Destination>**

### Authorization

[bearerAuth](README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | destinations response |  -  |
**0** | unexpected error |  -  |

[[Back to top]](#) [[Back to API list]](README.md#documentation-for-api-endpoints) [[Back to Model list]](README.md#documentation-for-models) [[Back to README]](README.md)



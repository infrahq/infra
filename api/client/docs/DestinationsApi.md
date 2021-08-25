# \DestinationsApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateDestination**](DestinationsApi.md#CreateDestination) | **Post** /destinations | Register a destination
[**ListDestinations**](DestinationsApi.md#ListDestinations) | **Get** /destinations | List destinations



## CreateDestination

> Destination CreateDestination(ctx).Body(body).Execute()

Register a destination

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {
    body := *openapiclient.NewDestinationCreateRequest("Name_example", *openapiclient.NewDestinationKubernetes("Ca_example", "Endpoint_example", "Namespace_example", "SaToken_example")) // DestinationCreateRequest | 

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.DestinationsApi.CreateDestination(context.Background()).Body(body).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `DestinationsApi.CreateDestination``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `CreateDestination`: Destination
    fmt.Fprintf(os.Stdout, "Response from `DestinationsApi.CreateDestination`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateDestinationRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**DestinationCreateRequest**](DestinationCreateRequest.md) |  | 

### Return type

[**Destination**](Destination.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListDestinations

> []Destination ListDestinations(ctx).Execute()

List destinations

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.DestinationsApi.ListDestinations(context.Background()).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `DestinationsApi.ListDestinations``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `ListDestinations`: []Destination
    fmt.Fprintf(os.Stdout, "Response from `DestinationsApi.ListDestinations`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListDestinationsRequest struct via the builder pattern


### Return type

[**[]Destination**](Destination.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


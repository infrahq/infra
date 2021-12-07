# Documentation for Infra API

<a name="documentation-for-api-endpoints"></a>
## Documentation for API Endpoints

All URIs are relative to *http://localhost/v1*

Class | Method | HTTP request | Description
------------ | ------------- | ------------- | -------------
*ApiKeysApi* | [**createAPIKey**](Apis/ApiKeysApi.md#createapikey) | **POST** /api-keys | Create an API key
*ApiKeysApi* | [**deleteAPIKey**](Apis/ApiKeysApi.md#deleteapikey) | **DELETE** /api-keys/{id} | Delete an API key
*ApiKeysApi* | [**listAPIKeys**](Apis/ApiKeysApi.md#listapikeys) | **GET** /api-keys | List API keys
*AuthApi* | [**login**](Apis/AuthApi.md#login) | **POST** /login | Login to Infra
*AuthApi* | [**logout**](Apis/AuthApi.md#logout) | **POST** /logout | Logout of Infra
*DestinationsApi* | [**createDestination**](Apis/DestinationsApi.md#createdestination) | **POST** /destinations | Create a destination
*DestinationsApi* | [**getDestination**](Apis/DestinationsApi.md#getdestination) | **GET** /destinations/{id} | Get destination by ID
*DestinationsApi* | [**listDestinations**](Apis/DestinationsApi.md#listdestinations) | **GET** /destinations | List destinations
*GrantsApi* | [**getGrant**](Apis/GrantsApi.md#getgrant) | **GET** /grants/{id} | Get grant
*GrantsApi* | [**listGrants**](Apis/GrantsApi.md#listgrants) | **GET** /grants | List grants
*GroupsApi* | [**getGroup**](Apis/GroupsApi.md#getgroup) | **GET** /groups/{id} | Get group by ID
*GroupsApi* | [**listGroups**](Apis/GroupsApi.md#listgroups) | **GET** /groups | List groups
*ProvidersApi* | [**getProvider**](Apis/ProvidersApi.md#getprovider) | **GET** /providers/{id} | Get provider by ID
*ProvidersApi* | [**listProviders**](Apis/ProvidersApi.md#listproviders) | **GET** /providers | List providers
*TokensApi* | [**createToken**](Apis/TokensApi.md#createtoken) | **POST** /tokens | Create an infrastructure destination token
*UsersApi* | [**getUser**](Apis/UsersApi.md#getuser) | **GET** /users/{id} | Get user by ID
*UsersApi* | [**listUsers**](Apis/UsersApi.md#listusers) | **GET** /users | List users
*VersionApi* | [**version**](Apis/VersionApi.md#version) | **GET** /version | Get version information


<a name="documentation-for-models"></a>
## Documentation for Models

 - [Destination](./Models/Destination.md)
 - [DestinationCreateRequest](./Models/DestinationCreateRequest.md)
 - [DestinationKind](./Models/DestinationKind.md)
 - [DestinationKubernetes](./Models/DestinationKubernetes.md)
 - [Error](./Models/Error.md)
 - [Grant](./Models/Grant.md)
 - [GrantKind](./Models/GrantKind.md)
 - [GrantKubernetes](./Models/GrantKubernetes.md)
 - [GrantKubernetesKind](./Models/GrantKubernetesKind.md)
 - [Group](./Models/Group.md)
 - [InfraAPIKey](./Models/InfraAPIKey.md)
 - [InfraAPIKeyCreateRequest](./Models/InfraAPIKeyCreateRequest.md)
 - [InfraAPIKeyCreateResponse](./Models/InfraAPIKeyCreateResponse.md)
 - [LoginRequest](./Models/LoginRequest.md)
 - [LoginRequestOkta](./Models/LoginRequestOkta.md)
 - [LoginResponse](./Models/LoginResponse.md)
 - [Provider](./Models/Provider.md)
 - [ProviderKind](./Models/ProviderKind.md)
 - [Token](./Models/Token.md)
 - [TokenRequest](./Models/TokenRequest.md)
 - [User](./Models/User.md)
 - [Version](./Models/Version.md)


<a name="documentation-for-authorization"></a>
## Documentation for Authorization

<a name="bearerAuth"></a>
### bearerAuth

- **Type**: HTTP basic authentication


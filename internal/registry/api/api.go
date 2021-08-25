/*
 * Infra API
 *
 * Infra REST API
 *
 * API version: 0.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package api

import (
	"context"
	"net/http"
)



// AuthApiRouter defines the required methods for binding the api requests to a responses for the AuthApi
// The AuthApiRouter implementation should parse necessary information from the http request, 
// pass the data to a AuthApiServicer to perform the required actions, then write the service results to the http response.
type AuthApiRouter interface { 
	Login(http.ResponseWriter, *http.Request)
	Logout(http.ResponseWriter, *http.Request)
	Signup(http.ResponseWriter, *http.Request)
}
// CredsApiRouter defines the required methods for binding the api requests to a responses for the CredsApi
// The CredsApiRouter implementation should parse necessary information from the http request, 
// pass the data to a CredsApiServicer to perform the required actions, then write the service results to the http response.
type CredsApiRouter interface { 
	CreateCred(http.ResponseWriter, *http.Request)
}
// DestinationsApiRouter defines the required methods for binding the api requests to a responses for the DestinationsApi
// The DestinationsApiRouter implementation should parse necessary information from the http request, 
// pass the data to a DestinationsApiServicer to perform the required actions, then write the service results to the http response.
type DestinationsApiRouter interface { 
	CreateDestination(http.ResponseWriter, *http.Request)
	ListDestinations(http.ResponseWriter, *http.Request)
}
// GroupsApiRouter defines the required methods for binding the api requests to a responses for the GroupsApi
// The GroupsApiRouter implementation should parse necessary information from the http request, 
// pass the data to a GroupsApiServicer to perform the required actions, then write the service results to the http response.
type GroupsApiRouter interface { 
	ListGroups(http.ResponseWriter, *http.Request)
}
// InfoApiRouter defines the required methods for binding the api requests to a responses for the InfoApi
// The InfoApiRouter implementation should parse necessary information from the http request, 
// pass the data to a InfoApiServicer to perform the required actions, then write the service results to the http response.
type InfoApiRouter interface { 
	Status(http.ResponseWriter, *http.Request)
	Version(http.ResponseWriter, *http.Request)
}
// RolesApiRouter defines the required methods for binding the api requests to a responses for the RolesApi
// The RolesApiRouter implementation should parse necessary information from the http request, 
// pass the data to a RolesApiServicer to perform the required actions, then write the service results to the http response.
type RolesApiRouter interface { 
	ListRoles(http.ResponseWriter, *http.Request)
}
// SourcesApiRouter defines the required methods for binding the api requests to a responses for the SourcesApi
// The SourcesApiRouter implementation should parse necessary information from the http request, 
// pass the data to a SourcesApiServicer to perform the required actions, then write the service results to the http response.
type SourcesApiRouter interface { 
	ListSources(http.ResponseWriter, *http.Request)
}
// UsersApiRouter defines the required methods for binding the api requests to a responses for the UsersApi
// The UsersApiRouter implementation should parse necessary information from the http request, 
// pass the data to a UsersApiServicer to perform the required actions, then write the service results to the http response.
type UsersApiRouter interface { 
	ListUsers(http.ResponseWriter, *http.Request)
}


// AuthApiServicer defines the api actions for the AuthApi service
// This interface intended to stay up to date with the openapi yaml used to generate it, 
// while the service implementation can ignored with the .openapi-generator-ignore file 
// and updated with the logic required for the API.
type AuthApiServicer interface { 
	Login(context.Context, LoginRequest) (ImplResponse, error)
	Logout(context.Context) (ImplResponse, error)
	Signup(context.Context, SignupRequest) (ImplResponse, error)
}


// CredsApiServicer defines the api actions for the CredsApi service
// This interface intended to stay up to date with the openapi yaml used to generate it, 
// while the service implementation can ignored with the .openapi-generator-ignore file 
// and updated with the logic required for the API.
type CredsApiServicer interface { 
	CreateCred(context.Context) (ImplResponse, error)
}


// DestinationsApiServicer defines the api actions for the DestinationsApi service
// This interface intended to stay up to date with the openapi yaml used to generate it, 
// while the service implementation can ignored with the .openapi-generator-ignore file 
// and updated with the logic required for the API.
type DestinationsApiServicer interface { 
	CreateDestination(context.Context, DestinationCreateRequest) (ImplResponse, error)
	ListDestinations(context.Context) (ImplResponse, error)
}


// GroupsApiServicer defines the api actions for the GroupsApi service
// This interface intended to stay up to date with the openapi yaml used to generate it, 
// while the service implementation can ignored with the .openapi-generator-ignore file 
// and updated with the logic required for the API.
type GroupsApiServicer interface { 
	ListGroups(context.Context) (ImplResponse, error)
}


// InfoApiServicer defines the api actions for the InfoApi service
// This interface intended to stay up to date with the openapi yaml used to generate it, 
// while the service implementation can ignored with the .openapi-generator-ignore file 
// and updated with the logic required for the API.
type InfoApiServicer interface { 
	Status(context.Context) (ImplResponse, error)
	Version(context.Context) (ImplResponse, error)
}


// RolesApiServicer defines the api actions for the RolesApi service
// This interface intended to stay up to date with the openapi yaml used to generate it, 
// while the service implementation can ignored with the .openapi-generator-ignore file 
// and updated with the logic required for the API.
type RolesApiServicer interface { 
	ListRoles(context.Context, string) (ImplResponse, error)
}


// SourcesApiServicer defines the api actions for the SourcesApi service
// This interface intended to stay up to date with the openapi yaml used to generate it, 
// while the service implementation can ignored with the .openapi-generator-ignore file 
// and updated with the logic required for the API.
type SourcesApiServicer interface { 
	ListSources(context.Context) (ImplResponse, error)
}


// UsersApiServicer defines the api actions for the UsersApi service
// This interface intended to stay up to date with the openapi yaml used to generate it, 
// while the service implementation can ignored with the .openapi-generator-ignore file 
// and updated with the logic required for the API.
type UsersApiServicer interface { 
	ListUsers(context.Context) (ImplResponse, error)
}

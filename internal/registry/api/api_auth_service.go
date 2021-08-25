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
	"errors"
	"net/http"
)

// AuthApiService is a service that implents the logic for the AuthApiServicer
// This service should implement the business logic for every endpoint for the AuthApi API.
// Include any external packages or services that will be required by this service.
type AuthApiService struct {
}

// NewAuthApiService creates a default api service
func NewAuthApiService() AuthApiServicer {
	return &AuthApiService{}
}

// Login - Log in to Infra and get an API token for a user
func (s *AuthApiService) Login(ctx context.Context, body LoginRequest) (ImplResponse, error) {
	// TODO - update Login with the required logic for this service method.
	// Add api_auth_service.go to the .openapi-generator-ignore to avoid overwriting this service implementation when updating open api generation.

	//TODO: Uncomment the next line to return response Response(200, AuthResponse{}) or use other options such as http.Ok ...
	//return Response(200, AuthResponse{}), nil

	//TODO: Uncomment the next line to return response Response(0, Error{}) or use other options such as http.Ok ...
	//return Response(0, Error{}), nil

	return Response(http.StatusNotImplemented, nil), errors.New("Login method not implemented")
}

// Logout - Log out of Infra
func (s *AuthApiService) Logout(ctx context.Context) (ImplResponse, error) {
	// TODO - update Logout with the required logic for this service method.
	// Add api_auth_service.go to the .openapi-generator-ignore to avoid overwriting this service implementation when updating open api generation.

	//TODO: Uncomment the next line to return response Response(200, {}) or use other options such as http.Ok ...
	//return Response(200, nil),nil

	//TODO: Uncomment the next line to return response Response(0, Error{}) or use other options such as http.Ok ...
	//return Response(0, Error{}), nil

	return Response(http.StatusNotImplemented, nil), errors.New("Logout method not implemented")
}

// Signup - Sign up Infra&#39;s admin user and get an API token for a user
func (s *AuthApiService) Signup(ctx context.Context, body SignupRequest) (ImplResponse, error) {
	// TODO - update Signup with the required logic for this service method.
	// Add api_auth_service.go to the .openapi-generator-ignore to avoid overwriting this service implementation when updating open api generation.

	//TODO: Uncomment the next line to return response Response(200, AuthResponse{}) or use other options such as http.Ok ...
	//return Response(200, AuthResponse{}), nil

	//TODO: Uncomment the next line to return response Response(0, Error{}) or use other options such as http.Ok ...
	//return Response(0, Error{}), nil

	return Response(http.StatusNotImplemented, nil), errors.New("Signup method not implemented")
}

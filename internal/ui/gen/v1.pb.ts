/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "./fetch.pb"
import * as GoogleProtobufEmpty from "google-protobuf/google/protobuf/empty_pb"

export enum SourceType {
  INFRA = "INFRA",
  OKTA = "OKTA",
}

export enum DestinationType {
  KUBERNETES = "KUBERNETES",
}

export enum KubernetesRoleType {
  ROLE = "ROLE",
  CLUSTER_ROLE = "CLUSTER_ROLE",
}

export type User = {
  id?: string
  created?: string
  updated?: string
  email?: string
  admin?: boolean
}

export type ListUsersRequest = {
  email?: string
}

export type ListUsersResponse = {
  users?: User[]
}

export type CreateUserRequest = {
  email?: string
  password?: string
}

export type DeleteUserRequest = {
  id?: string
}

export type SourceOkta = {
  domain?: string
  clientId?: string
}

export type Source = {
  id?: string
  created?: string
  updated?: string
  type?: SourceType
  okta?: SourceOkta
}

export type ListSourcesResponse = {
  sources?: Source[]
}

export type CreateSourceRequestOkta = {
  domain?: string
  clientId?: string
  clientSecret?: string
  apiToken?: string
}

export type CreateSourceRequest = {
  type?: SourceType
  okta?: CreateSourceRequestOkta
}

export type DeleteSourceRequest = {
  id?: string
}

export type DestinationKubernetes = {
  ca?: string
  endpoint?: string
  namespace?: string
  saToken?: string
}

export type Destination = {
  id?: string
  created?: string
  updated?: string
  name?: string
  type?: DestinationType
  kubernetes?: DestinationKubernetes
}

export type ListDestinationsResponse = {
  destinations?: Destination[]
}

export type CreateDestinationRequestKubernetes = {
  ca?: string
  endpoint?: string
  namespace?: string
  saToken?: string
}

export type CreateDestinationRequest = {
  name?: string
  type?: DestinationType
  kubernetes?: CreateDestinationRequestKubernetes
}

export type Role = {
  id?: string
  created?: string
  updated?: string
  user?: User
  destination?: Destination
  name?: string
  kind?: KubernetesRoleType
}

export type ListRolesRequest = {
  destinationId?: string
}

export type ListRolesResponse = {
  roles?: Role[]
}

export type CreateCredResponse = {
  token?: string
  expires?: string
}

export type ApiKey = {
  id?: string
  created?: string
  updated?: string
  name?: string
  key?: string
}

export type ListApiKeyResponse = {
  apiKeys?: ApiKey[]
}

export type LoginRequestInfra = {
  email?: string
  password?: string
}

export type LoginRequestOkta = {
  domain?: string
  code?: string
}

export type LoginRequest = {
  type?: SourceType
  infra?: LoginRequestInfra
  okta?: LoginRequestOkta
}

export type LoginResponse = {
  token?: string
}

export type SignupRequest = {
  email?: string
  password?: string
}

export type StatusResponse = {
  admin?: boolean
}

export type VersionResponse = {
  version?: string
}

export type Error = {
  message?: string
  details?: ErrorDetails[]
}

export type ErrorDetails = {
  name?: string
  description?: string
}

export class V1 {
  static ListUsers(req: ListUsersRequest, initReq?: fm.InitReq): Promise<ListUsersResponse> {
    return fm.fetchReq<ListUsersRequest, ListUsersResponse>(`/v1/users?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static CreateUser(req: CreateUserRequest, initReq?: fm.InitReq): Promise<User> {
    return fm.fetchReq<CreateUserRequest, User>(`/v1/users`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static DeleteUser(req: DeleteUserRequest, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<DeleteUserRequest, GoogleProtobufEmpty.Empty>(`/v1/users/${req["id"]}`, {...initReq, method: "DELETE"})
  }
  static ListDestinations(req: GoogleProtobufEmpty.Empty, initReq?: fm.InitReq): Promise<ListDestinationsResponse> {
    return fm.fetchReq<GoogleProtobufEmpty.Empty, ListDestinationsResponse>(`/v1.V1/ListDestinations`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static CreateDestination(req: CreateDestinationRequest, initReq?: fm.InitReq): Promise<Destination> {
    return fm.fetchReq<CreateDestinationRequest, Destination>(`/v1.V1/CreateDestination`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static ListSources(req: GoogleProtobufEmpty.Empty, initReq?: fm.InitReq): Promise<ListSourcesResponse> {
    return fm.fetchReq<GoogleProtobufEmpty.Empty, ListSourcesResponse>(`/v1.V1/ListSources`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static CreateSource(req: CreateSourceRequest, initReq?: fm.InitReq): Promise<Source> {
    return fm.fetchReq<CreateSourceRequest, Source>(`/v1.V1/CreateSource`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static DeleteSource(req: DeleteSourceRequest, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<DeleteSourceRequest, GoogleProtobufEmpty.Empty>(`/v1.V1/DeleteSource`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static ListRoles(req: ListRolesRequest, initReq?: fm.InitReq): Promise<ListRolesResponse> {
    return fm.fetchReq<ListRolesRequest, ListRolesResponse>(`/v1.V1/ListRoles`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static CreateCred(req: GoogleProtobufEmpty.Empty, initReq?: fm.InitReq): Promise<CreateCredResponse> {
    return fm.fetchReq<GoogleProtobufEmpty.Empty, CreateCredResponse>(`/v1.V1/CreateCred`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static ListApiKeys(req: GoogleProtobufEmpty.Empty, initReq?: fm.InitReq): Promise<ListApiKeyResponse> {
    return fm.fetchReq<GoogleProtobufEmpty.Empty, ListApiKeyResponse>(`/v1.V1/ListApiKeys`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static Login(req: LoginRequest, initReq?: fm.InitReq): Promise<LoginResponse> {
    return fm.fetchReq<LoginRequest, LoginResponse>(`/v1/login`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static Logout(req: GoogleProtobufEmpty.Empty, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<GoogleProtobufEmpty.Empty, GoogleProtobufEmpty.Empty>(`/v1/logout`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static Signup(req: SignupRequest, initReq?: fm.InitReq): Promise<LoginResponse> {
    return fm.fetchReq<SignupRequest, LoginResponse>(`/v1/signup`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static Status(req: GoogleProtobufEmpty.Empty, initReq?: fm.InitReq): Promise<StatusResponse> {
    return fm.fetchReq<GoogleProtobufEmpty.Empty, StatusResponse>(`/v1/status?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static Version(req: GoogleProtobufEmpty.Empty, initReq?: fm.InitReq): Promise<VersionResponse> {
    return fm.fetchReq<GoogleProtobufEmpty.Empty, VersionResponse>(`/v1.V1/Version`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
}
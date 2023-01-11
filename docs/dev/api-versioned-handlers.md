## API Versioned Handlers

API versioned handlers (previously known as API migrations) are API handlers that are
called when the request identifies a previous version with the `Infra-Version` HTTP request
header. These handlers keep our API endpoints backwards compatible. When we make a
breaking change to our API, we create a versioned handler that implements the existing API
endpoint.

This document will guide you through the steps of making breaking change to an API
endpoint.


1. Find the appropriate `API.addPreviousVersionHandlers<Type>` function in
   `internal/server`. If one does not exist for the type you are changing, create it and
   call it from the same place in `server.GenerateRoutes`.
2. If the request or response struct are changing, create a copy of that struct with
   the current infra version as a suffix. For example, if the `api.Grant` struct is
   being changed, and the current version is `v0.20.2`, the copy of the existing struct
   should be named `grantV0_20_2`.
3. Look for all the existing places where the struct is used. Any use of the struct
   as a field in other `api` types will require a versioned handler for all of the
   API endpoints that use the struct. Any existing versioned handlers that reference
   the api struct directly also need to be updated to use the new versioned struct.
4. Once all the existing handlers have been updated, it's time to create the new
   versioned handler. Use `addVersionHandler` to register the handler. The version argument to
   `addVersionHandler` must be the current API version (in other words, the last version
   of the API that used the old types).
5. Finally, now that the old behaviour of all API endpoints have been preserved by
   versioned handlers you are free to modify the `api` types and API endpoint behaviour.

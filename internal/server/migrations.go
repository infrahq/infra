package server

import "github.com/infrahq/infra/api"

func addRewrites() {
	// all migrations go here
	//
	//    http method ---v           v--- path      v--- last version that supports the old response
	addResponseRewrite("get", "/v1/access-keys", "0.11.0", func(newResponse *api.ListResponse[api.AccessKey]) []api.AccessKey {
		return newResponse.Items
	})
	addResponseRewrite("get", "/v1/identities", "0.11.0", func(newResponse *api.ListResponse[api.Identity]) []api.Identity {
		return newResponse.Items
	})
	addResponseRewrite("get", "/v1/grants", "0.11.0", func(newResponse *api.ListResponse[api.Grant]) []api.Grant {
		return newResponse.Items
	})
	addResponseRewrite("get", "/v1/identities/:id/grants", "0.11.0", func(newResponse *api.ListResponse[api.Grant]) []api.Grant {
		return newResponse.Items
	})
	addResponseRewrite("get", "/v1/identities/:id/groups", "0.11.0", func(newResponse *api.ListResponse[api.Group]) []api.Group {
		return newResponse.Items
	})
	addResponseRewrite("get", "/v1/groups", "0.11.0", func(newResponse *api.ListResponse[api.Group]) []api.Group {
		return newResponse.Items
	})
	addResponseRewrite("get", "/v1/groups/:id/grants", "0.11.0", func(newResponse *api.ListResponse[api.Grant]) []api.Grant {
		return newResponse.Items
	})
	addResponseRewrite("get", "/v1/providers", "0.11.0", func(newResponse *api.ListResponse[api.Provider]) []api.Provider {
		return newResponse.Items
	})
	addResponseRewrite("get", "/v1/destinations", "0.11.0", func(newResponse *api.ListResponse[api.Destination]) []api.Destination {
		return newResponse.Items
	})
	// next migration ...
}

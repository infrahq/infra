package server

func (a *API) addRequestRewrites() {
	// all request migrations go here
}

func (a *API) addResponseRewrites() {
	// all response migrations go here
}

func (a *API) addRewrites() {
	a.addRequestRewrites()
	a.addResponseRewrites()
}

// addRedirects for API endpoints that have moved to a different path
func (a *API) addRedirects() {
}

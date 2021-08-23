package api

import "net/http"

func HandlerWithBaseURL(si ServerInterface, baseURL string) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{BaseURL: baseURL})
}

//go:build !tei_repair_only && !tei_api_only

package main

import "net/http"

// The existing HTML form remains available to the historical and rendering
// build variants. An API-only binary does not register this route.
func registerOrderPresentationRoute(mux *http.ServeMux, server *webTargetServer) {
	mux.HandleFunc("/custom-submit", server.handleCustomSubmit)
}

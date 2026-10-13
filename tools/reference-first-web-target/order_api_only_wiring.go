//go:build tei_api_only && tei_order_only && !tei_repair_only

package main

import "net/http"

// The API executable owns only transport and business Runner entrypoints.
// The companion Browser asset is built separately from the full reference.
func registerCombinedScenarioRoutes(mux *http.ServeMux, server *webTargetServer) {
	registerOrderEntryScenarioRoutes(mux, server)
}

// Deliberately absent: GET/POST /custom-submit and server-side HTML templates.
func registerOrderPresentationRoute(_ *http.ServeMux, _ *webTargetServer) {}

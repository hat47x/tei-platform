//go:build !tei_order_only && !tei_repair_only

package main

import "net/http"

// Preserve the Gate C combined reference's historical route surface. Future
// Application-only targets can supply a different function with the same
// signature and cannot accidentally register the excluded Application.
func registerCombinedScenarioRoutes(mux *http.ServeMux, server *webTargetServer) {
	registerOrderEntryScenarioRoutes(mux, server)
	registerRepairIntakeScenarioRoutes(mux, server)
}

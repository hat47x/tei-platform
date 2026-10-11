//go:build tei_repair_only && !tei_order_only

package main

import (
	"html/template"
	"net/http"
)

// The repair_intake binary does not parse or mount order_entry's custom UI.
// Repair-only links must be navigable and must not advertise the excluded
// order_entry Command and custom UI as though they existed.
func referenceIndexFooterHTML() string {
	return `<footer>JSON projection: <a href="/manifest.json">/manifest.json</a> · Customer lookup: <a href="/customer-lookup">/customer-lookup</a> · health: <a href="/healthz">/healthz</a></footer>`
}

func referenceScenarioTemplates() (*template.Template, *template.Template, error) {
	repairPage, err := template.New("customer-lookup").Parse(customerLookupHTML)
	if err != nil {
		return nil, nil, err
	}
	return nil, repairPage, nil
}

func registerCombinedScenarioRoutes(mux *http.ServeMux, server *webTargetServer) {
	registerRepairIntakeScenarioRoutes(mux, server)
}

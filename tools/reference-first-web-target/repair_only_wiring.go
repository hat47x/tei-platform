//go:build tei_repair_only && !tei_order_only

package main

import (
	"html/template"
	"net/http"
)

// The repair_intake binary does not parse or mount order_entry's custom UI.
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

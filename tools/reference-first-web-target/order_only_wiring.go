//go:build tei_order_only && !tei_repair_only && !tei_api_only

package main

import (
	"html/template"
	"net/http"
)

// The order_entry binary does not parse or mount repair_intake's page.
func referenceIndexFooterHTML() string {
	return orderScenarioIndexFooterHTML
}

func referenceScenarioTemplates() (*template.Template, *template.Template, error) {
	orderPage, err := template.New("custom-submit").Parse(customSubmitHTML)
	if err != nil {
		return nil, nil, err
	}
	return orderPage, nil, nil
}

func registerCombinedScenarioRoutes(mux *http.ServeMux, server *webTargetServer) {
	registerOrderEntryScenarioRoutes(mux, server)
}

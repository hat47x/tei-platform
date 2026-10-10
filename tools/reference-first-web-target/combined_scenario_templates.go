//go:build !tei_order_only && !tei_repair_only

package main

import "html/template"

// Keep legacy combined-mode template construction in one provider function.
// Other compile-time entries can replace this with only their own templates
// without changing the shared transport constructor.
func referenceScenarioTemplates() (*template.Template, *template.Template, error) {
	customSubmit, err := template.New("custom-submit").Parse(customSubmitHTML)
	if err != nil {
		return nil, nil, err
	}
	customerLookupPage, err := template.New("customer-lookup").Parse(customerLookupHTML)
	if err != nil {
		return nil, nil, err
	}
	return customSubmit, customerLookupPage, nil
}

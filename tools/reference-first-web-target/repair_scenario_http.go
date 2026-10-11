//go:build !tei_order_only

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Keep repair_intake's HTTP routes and response formatting isolated from
// its Driver-backed business Runner. These are still part of the combined
// diagnostic package until a compile-time target entry is supplied.
// Keep repair_intake route installation adjacent to the lookup handler;
// preserve the legacy 503/405 responses for routes without runners.
func registerRepairIntakeScenarioRoutes(mux *http.ServeMux, s *webTargetServer) {
	mux.HandleFunc("/scenario/customer-lookup", s.handleCustomerLookupScenario)
	mux.HandleFunc("/customer-lookup", s.handleCustomerLookupPage)
}

type webCustomerLookupResponse struct {
	HandoverBaselineID         string               `json:"handoverBaselineId"`
	BaselineRepositoryRevision string               `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string               `json:"targetBuildRevision"`
	TargetAdapter              string               `json:"targetAdapter"`
	Host                       string               `json:"host"`
	Result                     customerLookupResult `json:"result"`
}

func (s *webTargetServer) handleCustomerLookupScenario(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.customerLookup == nil {
		http.Error(w, "customer lookup scenario is not configured", http.StatusServiceUnavailable)
		return
	}
	result, err := s.customerLookup.RunCustomerLookup(r.Context(), customerLookupInput{
		CustomerID: r.URL.Query().Get("customerId"),
	})
	if err != nil {
		status := http.StatusUnprocessableEntity
		if errors.Is(err, errCustomerLookupInput) {
			status = http.StatusBadRequest
		}
		http.Error(w, "execute customer lookup scenario: "+err.Error(), status)
		return
	}
	response := webCustomerLookupResponse{
		HandoverBaselineID:         s.projection.HandoverBaselineID,
		BaselineRepositoryRevision: s.projection.BaselineRepositoryRevision,
		TargetBuildRevision:        s.projection.TargetBuildRevision,
		TargetAdapter:              s.projection.TargetAdapter,
		Host:                       s.projection.Host,
		Result:                     result,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(response); err != nil {
		http.Error(w, "encode customer lookup result", http.StatusInternalServerError)
	}
}

type customerLookupPageData struct {
	Projection webTargetProjection
	CustomerID string
	Result     *customerLookupResult
	Error      string
}

// handleCustomerLookupPage renders the same lookup as an HTML screen. It calls the same runner as
// the JSON endpoint, so the two never disagree about the result.
func (s *webTargetServer) handleCustomerLookupPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.customerLookup == nil {
		http.Error(w, "customer lookup scenario is not configured", http.StatusServiceUnavailable)
		return
	}
	data := customerLookupPageData{
		Projection: s.projection,
		CustomerID: r.URL.Query().Get("customerId"),
	}
	status := http.StatusOK
	if r.URL.Query().Has("customerId") {
		result, err := s.customerLookup.RunCustomerLookup(r.Context(), customerLookupInput{CustomerID: data.CustomerID})
		switch {
		case err == nil:
			data.Result = &result
		case errors.Is(err, errCustomerLookupInput):
			status = http.StatusBadRequest
			data.Error = err.Error()
		default:
			status = http.StatusUnprocessableEntity
			data.Error = "customer lookup could not complete: " + err.Error()
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := s.customerLookupPage.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("render customer lookup UI: %v", err), http.StatusInternalServerError)
	}
}

const customerLookupHTML = `<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>TEI Customer Lookup — {{.Projection.ApplicationID}}</title>
<style>
:root { font-family: system-ui, -apple-system, "Segoe UI", sans-serif; color-scheme: light dark; }
body { margin: 0; line-height: 1.55; }
main { max-width: 960px; margin: 0 auto; padding: 32px 24px 56px; }
.card { border: 1px solid currentColor; padding: 18px; margin-top: 20px; }
label { display: grid; gap: 5px; max-width: 320px; }
input { font: inherit; padding: 8px 9px; }
button { font: inherit; padding: 9px 15px; margin-top: 16px; }
table { border-collapse: collapse; width: 100%; }
th, td { border: 1px solid currentColor; padding: 6px 10px; text-align: left; overflow-wrap: anywhere; }
.error { border: 1px solid currentColor; border-left-width: 4px; padding: 10px 14px; margin: 18px 0; }
code { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; overflow-wrap: anywhere; }
ul { padding-left: 20px; }
</style>
</head>
<body>
<main>
<h1>顧客照会 — <code>{{.Projection.ApplicationID}}</code></h1>
<p>Target <code>{{.Projection.TargetAdapter}}</code> / baseline <code>{{.Projection.BaselineRepositoryRevision}}</code> / build <code>{{.Projection.TargetBuildRevision}}</code></p>
<div class="card">
<form method="get" action="/customer-lookup">
<label>Customer ID
<input name="customerId" value="{{.CustomerID}}" inputmode="numeric" autocomplete="off">
</label>
<button type="submit">照会する</button>
</form>
</div>
{{if .Error}}<div class="error" role="alert">{{.Error}}</div>{{end}}
{{with .Result}}
<div class="card">
<h2>顧客</h2>
<table>
<thead><tr><th>attribute</th><th>value</th></tr></thead>
<tbody>
{{range .CustomerRows}}{{range .Attributes}}<tr><td><code>{{.Attribute}}</code></td><td>{{.Value}}</td></tr>{{end}}{{end}}
</tbody>
</table>
<h3>実行の根拠</h3>
<ul>
{{range .RuntimeBindings}}<li><code>{{.Pattern}}</code> → Environment <code>{{.Environment}}</code> / instance <code>{{.Instance}}</code> / <code>{{.ContractQuery}}</code></li>{{end}}
</ul>
<h3>この結果が主張しないこと</h3>
<ul>
{{range .AuthorityLimits}}<li><code>{{.}}</code></li>{{end}}
</ul>
<h3>未解決</h3>
<ul>
{{range .Unresolved}}<li><code>{{.}}</code></li>{{end}}
</ul>
</div>
{{end}}
</main>
</body>
</html>
`

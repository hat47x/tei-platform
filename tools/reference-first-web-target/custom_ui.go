package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type customSubmitPageData struct {
	Projection webTargetProjection
	Result     *commandScenarioResult
	ResultJSON string
	Error      string
}

func (s *webTargetServer) handleCustomSubmit(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderCustomSubmit(w, http.StatusOK, customSubmitPageData{
			Projection: s.projection,
		})
	case http.MethodPost:
		if s.commandScenario == nil {
			http.Error(w, "command scenario is not configured", http.StatusServiceUnavailable)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		if err := r.ParseForm(); err != nil {
			s.renderCustomSubmit(w, http.StatusBadRequest, customSubmitPageData{
				Projection: s.projection,
				Error:      "form input could not be parsed",
			})
			return
		}

		rowIndex := 0
		if raw := r.Form.Get("row"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 0 {
				s.renderCustomSubmit(w, http.StatusBadRequest, customSubmitPageData{
					Projection: s.projection,
					Error:      "row must be a non-negative integer",
				})
				return
			}
			rowIndex = parsed
		}
		from := r.Form.Get("from")
		if from == "" {
			from = "2026-09-01T00:00:00Z"
		}

		result, err := s.commandScenario.RunCommand(r.Context(), commandScenarioInput{
			LoginID:    r.Form.Get("loginId"),
			Credential: r.Form.Get("credential"),
			From:       from,
			RowIndex:   rowIndex,
		})

		encoded, encodeErr := json.MarshalIndent(result, "", "  ")
		if encodeErr != nil {
			http.Error(w, "encode custom UI result", http.StatusInternalServerError)
			return
		}

		data := customSubmitPageData{
			Projection: s.projection,
			Result:     &result,
			ResultJSON: string(encoded),
		}
		status := http.StatusOK
		if err != nil {
			status = http.StatusUnprocessableEntity
			data.Error = "command scenario could not complete: " + err.Error()
		}
		s.renderCustomSubmit(w, status, data)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *webTargetServer) renderCustomSubmit(
	w http.ResponseWriter,
	status int,
	data customSubmitPageData,
) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := s.customSubmit.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("render custom submit UI: %v", err), http.StatusInternalServerError)
	}
}

const customSubmitHTML = `<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>TEI Custom Submit UI — {{.Projection.ApplicationID}}</title>
<style>
:root { font-family: system-ui, -apple-system, "Segoe UI", sans-serif; color-scheme: light dark; }
body { margin: 0; line-height: 1.55; }
main { max-width: 1080px; margin: 0 auto; padding: 32px 24px 56px; }
h1 { margin-bottom: 6px; }
h2 { margin-top: 0; }
h3 { margin-bottom: 8px; }
.meta, .summary-grid, .result-grid { display: grid; grid-template-columns: repeat(auto-fit,minmax(220px,1fr)); gap: 12px 18px; }
.meta { margin: 20px 0; }
.card { border: 1px solid currentColor; padding: 18px; margin-top: 20px; }
.subcard { border: 1px solid currentColor; padding: 14px; min-width: 0; }
.form-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 14px; }
label { display: grid; gap: 5px; }
input { font: inherit; padding: 8px 9px; }
button { font: inherit; padding: 9px 15px; margin-top: 16px; }
.notice, .stage, .error { border-left: 4px solid currentColor; padding: 10px 14px; margin: 18px 0; }
.error { border: 1px solid currentColor; border-left-width: 4px; }
code, pre { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; overflow-wrap: anywhere; }
pre { overflow: auto; padding: 14px; border: 1px solid currentColor; }
table { width: 100%; border-collapse: collapse; margin-top: 8px; }
th, td { text-align: left; vertical-align: top; border-bottom: 1px solid currentColor; padding: 7px 8px; }
th { width: 42%; }
.status-line { display: flex; flex-wrap: wrap; gap: 7px; align-items: baseline; }
.pill { display: inline-block; border: 1px solid currentColor; padding: 1px 7px; font-size: .84rem; }
.flow { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
.flow span { border: 1px solid currentColor; padding: 5px 8px; }
.muted { opacity: .72; }
details { margin-top: 20px; }
@media (max-width: 640px) {
  .form-grid { grid-template-columns: 1fr; }
  th { width: auto; }
}
</style>
</head>
<body>
<main>
<p><a href="/">← Projection</a></p>
<h1>Custom Submit UI</h1>
<p>この画面固有なのはform・layout・HTML renderingだけです。業務判断はJSON APIと同じ <code>commandScenarioRunner</code> を利用します。</p>

<div class="meta">
<div><small>Application</small><br><code>{{.Projection.ApplicationID}}</code></div>
<div><small>Handover</small><br><code>{{.Projection.HandoverBaselineID}}</code></div>
<div><small>Baseline R</small><br><code>{{.Projection.BaselineRepositoryRevision}}</code></div>
<div><small>Target T</small><br><code>{{.Projection.TargetBuildRevision}}</code></div>
</div>

<div class="notice">
Reference専用UIです。credentialはruntime inputとしてのみ使い、結果へ保持しません。
</div>

<section class="card">
<h2>Submit input</h2>
<form method="post" action="/custom-submit" autocomplete="off">
<div class="form-grid">
<label>Login ID
<input name="loginId" required placeholder="legacy-order-editor">
</label>
<label>Credential
<input name="credential" type="password" required>
</label>
<label>Browse from
<input name="from" value="2026-09-18T08:30:00Z" required>
</label>
<label>Selected row
<input name="row" type="number" min="0" value="0" required>
</label>
</div>
<button type="submit">Submit selected order</button>
</form>
</section>

{{if .Result}}
<section class="card">
<div class="status-line">
<h2>Scenario state</h2>
<span class="pill">{{.Result.Stage}}</span>
</div>
<div class="flow" aria-label="Scenario flow">
<span>Authentication</span><span>→</span>
<span>Read</span><span>→</span>
<span>Validation</span><span>→</span>
<span>Authorization</span><span>→</span>
<span>Command</span><span>→</span>
<span>Observation</span>
</div>
</section>

<section class="result-grid">
{{if .Result.Authentication}}
<article class="subcard" id="authentication-result">
<h3>Authentication</h3>
<p class="status-line"><code>{{.Result.Authentication.Capability}}</code>
{{if .Result.Authentication.Outcome}}<span class="pill">{{.Result.Authentication.Outcome}}</span>{{end}}</p>
{{if .Result.Authentication.Payload}}
<table>
{{range $key, $value := .Result.Authentication.Payload}}<tr><th><code>{{$key}}</code></th><td>{{$value}}</td></tr>{{end}}
</table>
{{end}}
</article>
{{end}}

{{if .Result.Authorization}}
<article class="subcard" id="authorization-result">
<h3>Authorization</h3>
<p class="status-line"><code>{{.Result.Authorization.Capability}}</code>
{{if .Result.Authorization.Outcome}}<span class="pill">{{.Result.Authorization.Outcome}}</span>{{end}}</p>
{{if .Result.Authorization.Payload}}
<table>
{{range $key, $value := .Result.Authorization.Payload}}<tr><th><code>{{$key}}</code></th><td>{{$value}}</td></tr>{{end}}
</table>
{{end}}
</article>
{{end}}
</section>

{{if .Result.Read}}
<section class="card" id="read-result">
<h2>Selected order and relations</h2>
<div class="summary-grid">
<article class="subcard">
<h3>Selected order</h3>
<table>
{{range .Result.Read.SelectedOrder.Attributes}}<tr><th><code>{{.Attribute}}</code></th><td>{{.Value}}</td></tr>{{end}}
</table>
</article>

<article class="subcard">
<h3>Customer reference</h3>
{{range .Result.Read.CustomerRows}}
<table>
{{range .Attributes}}<tr><th><code>{{.Attribute}}</code></th><td>{{.Value}}</td></tr>{{end}}
</table>
{{else}}<p class="muted">No related customer row.</p>{{end}}
</article>
</div>

<article class="subcard" id="order-lines">
<h3>Order lines</h3>
{{range $index, $row := .Result.Read.OrderLineRows}}
<p><strong>Line {{$index}}</strong></p>
<table>
{{range $row.Attributes}}<tr><th><code>{{.Attribute}}</code></th><td>{{.Value}}</td></tr>{{end}}
</table>
{{else}}<p class="muted">No order-line rows.</p>{{end}}
</article>
</section>
{{end}}

{{if .Result.Validations}}
<section class="card" id="validation-results">
<h2>Canonical Validation</h2>
<div class="result-grid">
{{range .Result.Validations}}
<article class="subcard">
<h3><code>{{.Validation}}</code></h3>
<p><span class="pill">{{.Result}}</span></p>
</article>
{{end}}
</div>
</section>
{{end}}

<section class="result-grid">
{{if .Result.Command}}
<article class="subcard" id="command-result">
<h3>Command Business Result</h3>
<p class="status-line"><code>{{.Result.Command.Capability}}</code>
<span class="pill">{{.Result.Command.Mode}}</span>
{{if .Result.Command.Outcome}}<span class="pill">{{.Result.Command.Outcome}}</span>{{end}}</p>
{{if .Result.Command.Payload}}
<table>
{{range $key, $value := .Result.Command.Payload}}<tr><th><code>{{$key}}</code></th><td>{{$value}}</td></tr>{{end}}
</table>
{{end}}
</article>
{{end}}

{{if .Result.Observation}}
<article class="subcard" id="observation-result">
<h3>Separate Observation</h3>
<table>
<tr><th>Read identity</th><td><code>{{.Result.Observation.ReadIdentity}}</code></td></tr>
<tr><th>Evidence nature</th><td><span class="pill">{{.Result.Observation.Nature}}</span></td></tr>
<tr><th>Observed result</th><td>{{.Result.Observation.Result}}</td></tr>
</table>
</article>
{{end}}
</section>
{{end}}

{{if .Error}}
<section class="error" id="execution-error">
<h2>Execution error</h2>
<p>{{.Error}}</p>
<p class="muted">Execution ErrorはAuthorization denialやValidation not-satisfiedとは別のTarget状態として表示しています。</p>
</section>
{{end}}

{{if .ResultJSON}}
<details>
<summary>Diagnostic raw Scenario result</summary>
<pre>{{.ResultJSON}}</pre>
</details>
{{end}}
</main>
</body>
</html>
`

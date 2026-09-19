package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type customSubmitPageData struct {
	Projection webTargetProjection
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
		if err != nil {
			s.renderCustomSubmit(w, http.StatusUnprocessableEntity, customSubmitPageData{
				Projection: s.projection,
				Error:      "command scenario could not complete: " + err.Error(),
			})
			return
		}

		encoded, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			http.Error(w, "encode custom UI result", http.StatusInternalServerError)
			return
		}
		s.renderCustomSubmit(w, http.StatusOK, customSubmitPageData{
			Projection: s.projection,
			ResultJSON: string(encoded),
		})
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
main { max-width: 880px; margin: 0 auto; padding: 32px 24px 56px; }
h1 { margin-bottom: 6px; }
.meta { display: grid; grid-template-columns: repeat(auto-fit,minmax(210px,1fr)); gap: 8px 18px; margin: 20px 0; }
.card { border: 1px solid currentColor; padding: 18px; margin-top: 20px; }
.form-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 14px; }
label { display: grid; gap: 5px; }
input { font: inherit; padding: 8px 9px; }
button { font: inherit; padding: 9px 15px; margin-top: 16px; }
.notice { border-left: 4px solid currentColor; padding: 10px 14px; margin: 18px 0; }
.error { border: 1px solid currentColor; padding: 10px 14px; margin-top: 18px; }
pre { overflow: auto; padding: 14px; border: 1px solid currentColor; }
code, pre { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; }
@media (max-width: 640px) { .form-grid { grid-template-columns: 1fr; } }
</style>
</head>
<body>
<main>
<p><a href="/">← Projection</a></p>
<h1>Custom Submit UI</h1>
<p>この画面固有なのはformとHTML表示だけです。業務判断はJSON APIと同じ <code>commandScenarioRunner</code> を利用します。</p>

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

{{if .Error}}
<div class="error"><strong>Execution error</strong><br>{{.Error}}</div>
{{end}}

{{if .ResultJSON}}
<section class="card">
<h2>Scenario result</h2>
<pre>{{.ResultJSON}}</pre>
</section>
{{end}}
</main>
</body>
</html>
`

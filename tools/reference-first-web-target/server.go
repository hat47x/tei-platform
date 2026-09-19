package main

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"strconv"
)

type webTargetServer struct {
	projection      webTargetProjection
	readScenario    readScenarioRunner
	commandScenario commandScenarioRunner
	index           *template.Template
	customSubmit    *template.Template
}

type webReadScenarioResponse struct {
	HandoverBaselineID         string             `json:"handoverBaselineId"`
	BaselineRepositoryRevision string             `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string             `json:"targetBuildRevision"`
	TargetAdapter              string             `json:"targetAdapter"`
	Host                       string             `json:"host"`
	Result                     readScenarioResult `json:"result"`
}

type webCommandScenarioRequest struct {
	LoginID    string `json:"loginId"`
	Credential string `json:"credential"`
	From       string `json:"from"`
	RowIndex   int    `json:"row"`
}

type webCommandScenarioResponse struct {
	HandoverBaselineID         string                `json:"handoverBaselineId"`
	BaselineRepositoryRevision string                `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string                `json:"targetBuildRevision"`
	TargetAdapter              string                `json:"targetAdapter"`
	Host                       string                `json:"host"`
	Result                     commandScenarioResult `json:"result"`
}

type webCommandScenarioErrorResponse struct {
	HandoverBaselineID         string                `json:"handoverBaselineId"`
	BaselineRepositoryRevision string                `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string                `json:"targetBuildRevision"`
	TargetAdapter              string                `json:"targetAdapter"`
	Host                       string                `json:"host"`
	Result                     commandScenarioResult `json:"result"`
	ExecutionError             string                `json:"executionError"`
}

func newWebTargetServer(
	projection webTargetProjection,
	readScenario readScenarioRunner,
	commandScenario commandScenarioRunner,
) (*webTargetServer, error) {
	index, err := template.New("index").Parse(indexHTML)
	if err != nil {
		return nil, err
	}
	customSubmit, err := template.New("custom-submit").Parse(customSubmitHTML)
	if err != nil {
		return nil, err
	}
	return &webTargetServer{
		projection:      projection,
		readScenario:    readScenario,
		commandScenario: commandScenario,
		index:           index,
		customSubmit:    customSubmit,
	}, nil
}

func (s *webTargetServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/manifest.json", s.handleManifest)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/scenario/read", s.handleReadScenario)
	mux.HandleFunc("/scenario/submit", s.handleCommandScenario)
	mux.HandleFunc("/custom-submit", s.handleCustomSubmit)
	return mux
}

func (s *webTargetServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.index.Execute(w, s.projection); err != nil {
		http.Error(w, "render target projection", http.StatusInternalServerError)
	}
}

func (s *webTargetServer) handleManifest(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(s.projection); err != nil {
		http.Error(w, "encode target manifest", http.StatusInternalServerError)
	}
}

func (s *webTargetServer) handleReadScenario(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.readScenario == nil {
		http.Error(w, "read scenario is not configured", http.StatusServiceUnavailable)
		return
	}

	rowIndex := 0
	if raw := r.URL.Query().Get("row"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			http.Error(w, "row must be a non-negative integer", http.StatusBadRequest)
			return
		}
		rowIndex = parsed
	}
	from := r.URL.Query().Get("from")
	if from == "" {
		from = "2026-09-01T00:00:00Z"
	}

	result, err := s.readScenario.Run(r.Context(), readScenarioInput{
		From:     from,
		RowIndex: rowIndex,
	})
	if err != nil {
		http.Error(w, "execute read scenario: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	response := webReadScenarioResponse{
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
		http.Error(w, "encode read scenario result", http.StatusInternalServerError)
	}
}

func (s *webTargetServer) handleCommandScenario(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.commandScenario == nil {
		http.Error(w, "command scenario is not configured", http.StatusServiceUnavailable)
		return
	}

	var request webCommandScenarioRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid command request", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "command request must contain one JSON document", http.StatusBadRequest)
		return
	}
	if request.From == "" {
		request.From = "2026-09-01T00:00:00Z"
	}

	result, err := s.commandScenario.RunCommand(r.Context(), commandScenarioInput{
		LoginID:    request.LoginID,
		Credential: request.Credential,
		From:       request.From,
		RowIndex:   request.RowIndex,
	})
	if err != nil {
		response := webCommandScenarioErrorResponse{
			HandoverBaselineID:         s.projection.HandoverBaselineID,
			BaselineRepositoryRevision: s.projection.BaselineRepositoryRevision,
			TargetBuildRevision:        s.projection.TargetBuildRevision,
			TargetAdapter:              s.projection.TargetAdapter,
			Host:                       s.projection.Host,
			Result:                     result,
			ExecutionError:             err.Error(),
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(response); encodeErr != nil {
			http.Error(w, "encode command scenario execution error", http.StatusInternalServerError)
		}
		return
	}

	response := webCommandScenarioResponse{
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
		http.Error(w, "encode command scenario result", http.StatusInternalServerError)
	}
}

func (s *webTargetServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

const indexHTML = `<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>TEI First Web Target — {{.ApplicationID}}</title>
<style>
:root { font-family: system-ui, -apple-system, "Segoe UI", sans-serif; color-scheme: light dark; }
body { margin: 0; line-height: 1.55; }
header, main { max-width: 1120px; margin: 0 auto; padding: 24px; }
header { border-bottom: 1px solid currentColor; }
h1, h2, h3 { line-height: 1.2; }
.meta, .limits { display: grid; grid-template-columns: repeat(auto-fit,minmax(220px,1fr)); gap: 8px 20px; }
code { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; }
.patterns { display: grid; grid-template-columns: repeat(auto-fit,minmax(300px,1fr)); gap: 16px; }
.pattern { border: 1px solid currentColor; padding: 16px; }
.kv { display: grid; grid-template-columns: 130px 1fr; gap: 5px 12px; }
.kv dt { opacity: .72; }
.kv dd { margin: 0; overflow-wrap: anywhere; }
.badge { display: inline-block; border: 1px solid currentColor; padding: 1px 7px; margin-right: 6px; font-size: .82rem; }
.notice { border-left: 4px solid currentColor; padding: 10px 14px; margin: 18px 0; }
ul.compact { margin-top: 6px; padding-left: 20px; }
footer { max-width: 1120px; margin: 0 auto; padding: 0 24px 40px; opacity: .72; }
</style>
</head>
<body>
<header>
<p><strong>TEI / First Web Target Reference</strong></p>
<h1>{{.ApplicationID}}</h1>
<div class="meta">
<div><small>Handover baseline</small><br><code>{{.HandoverBaselineID}}</code></div>
<div><small>Baseline revision R</small><br><code>{{.BaselineRepositoryRevision}}</code></div>
<div><small>Target build revision T</small><br><code>{{.TargetBuildRevision}}</code></div>
<div><small>Adapter / Host</small><br><code>{{.TargetAdapter}} / {{.Host}}</code></div>
</div>
<div class="notice">
<strong>Stage: {{.Stage}}</strong><br>
この画面はCanonical Applicationから導出したReference projectionです。Web側の業務正本ではありません。
</div>
</header>

<main>
<section>
<h2>Application meaning</h2>
<div class="limits">
<div>
<h3>Semantic uses</h3>
<ul class="compact">{{range .SemanticUses}}<li><code>{{.}}</code></li>{{end}}</ul>
</div>
<div>
<h3>Validations</h3>
<ul class="compact">{{range .Validations}}<li><code>{{.}}</code></li>{{else}}<li>none declared at Application level</li>{{end}}</ul>
</div>
</div>
</section>

<section>
<h2>Patterns</h2>
<div class="patterns">
{{range .Patterns}}
<article class="pattern">
<p><span class="badge">{{.Kind}}</span><span class="badge">web: {{.PresentationCandidate}}</span></p>
<h3><code>{{.ID}}</code></h3>
<dl class="kv">
{{if .Data}}<dt>Data</dt><dd><code>{{.Data}}</code></dd>{{end}}
{{if .Relation}}<dt>Relation</dt><dd><code>{{.Relation.ID}}</code><br><small><code>{{.Relation.Source}} → {{.Relation.Target}}</code></small></dd>{{end}}
{{if .Query}}<dt>Query</dt><dd><code>{{.Query}}</code></dd>{{end}}
{{if .Datasource}}<dt>Datasource</dt><dd><code>{{.Datasource.Alias}}</code> → <code>{{.Datasource.Target}}</code> (<code>{{.Datasource.Access}}</code>)</dd>{{end}}
{{if .Capability}}<dt>Capability</dt><dd><code>{{.Capability}}</code></dd>{{end}}
{{if .ContractQuery}}
<dt>Contract Query</dt><dd><code>{{.ContractQuery.Contract}} v{{.ContractQuery.ContractVersion}} / {{.ContractQuery.Query}}</code><br>
<small>result <code>{{.ContractQuery.Result}}</code> / <code>{{.ContractQuery.ResultCardinality}}</code></small>
{{if .ContractQuery.Inputs}}<ul class="compact">{{range .ContractQuery.Inputs}}<li><code>{{.SemanticAttribute}}</code> → <code>{{.QueryInput}}</code> <small>({{.QueryInputType}})</small></li>{{end}}</ul>{{end}}
</dd>{{end}}
</dl>
</article>
{{end}}
</div>
</section>

<section>
<h2>Unresolved in this increment</h2>
<ul class="compact">{{range .Unresolved}}<li><code>{{.}}</code></li>{{end}}</ul>
</section>

<section>
<h2>Authority limits</h2>
<ul class="compact">{{range .AuthorityLimits}}<li><code>{{.}}</code></li>{{end}}</ul>
</section>
</main>
<footer>JSON projection: <a href="/manifest.json">/manifest.json</a> · read-only Driver slice: <a href="/scenario/read">/scenario/read</a> · submit Command: <code>POST /scenario/submit</code> · Custom UI: <a href="/custom-submit">/custom-submit</a> · health: <a href="/healthz">/healthz</a></footer>
</body>
</html>
`

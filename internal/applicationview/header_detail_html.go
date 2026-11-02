package applicationview

import (
	"bytes"
	"html/template"
)

var headerDetailTemplate = template.Must(template.New("header-detail-view").Parse(`<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
{{if .SemanticFingerprint}}<meta name="tei-semantic-fingerprint" content="{{.SemanticFingerprint}}">{{end}}
<title>{{.Pattern}}</title>
<style>
body{font-family:system-ui,sans-serif;max-width:56rem;margin:2rem auto;padding:0 1rem;color:#222}
h1{font-size:1.35rem;margin-bottom:.25rem}
small{font-weight:normal;color:#666}
.entry{display:grid;grid-template-columns:12rem 1fr;gap:.35rem 1rem;align-items:center;margin:.35rem 0}
.entry label{text-align:right;color:#333}
.entry input,.entry select{max-width:22rem;padding:.25rem}
table{width:100%;border-collapse:collapse;margin-top:1.5rem}
th,td{border:1px solid #bbb;padding:.35rem .5rem;text-align:left;font-size:.9rem}
th{background:#f3f3f3}
h2{font-size:1.05rem;margin-top:1.75rem}
</style>
</head>
<body>
<h1>{{.Pattern}} <small>({{.Kind}})</small></h1>
<p>relation: {{.Relation}} — {{.Header}} → {{.Detail}}</p>
{{if .Packages}}<p>packages: {{range .Packages}}{{.ID}} ^{{.Version}} {{end}}</p>{{end}}
<h2>見出し</h2>
{{range .HeaderItems}}<div class="entry">
<label for="{{.Attribute}}">{{.Label}}</label>
{{if .Values}}<select id="{{.Attribute}}" name="{{.Attribute}}"{{if .Immutable}} disabled{{end}}>{{range .Values}}<option value="{{.}}">{{.}}</option>{{end}}</select>{{else}}<input id="{{.Attribute}}" name="{{.Attribute}}" type="{{.Input}}"{{if .Required}} required{{end}}{{if .Immutable}} readonly{{end}}>{{end}}
</div>
{{end}}
<h2>明細</h2>
<table>
<thead><tr>{{range .DetailColumns}}<th>{{.Label}}</th>{{end}}</tr></thead>
<tbody><tr>{{range .DetailColumns}}<td></td>{{end}}</tr></tbody>
</table>
</body>
</html>
`))

// RenderHeaderDetailHTMLは、解決済みのheader-detailビューをHTMLへ投影する。
func RenderHeaderDetailHTML(view HeaderDetailView) (string, error) {
	var buf bytes.Buffer
	if err := headerDetailTemplate.Execute(&buf, view); err != nil {
		return "", err
	}
	return buf.String(), nil
}

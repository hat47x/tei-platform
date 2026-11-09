package applicationview

import (
	"bytes"
	"html/template"
)

var printTemplate = template.Must(template.New("print-view").Parse(`<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
{{if .SemanticFingerprint}}<meta name="tei-semantic-fingerprint" content="{{.SemanticFingerprint}}">{{end}}
<title>{{.Pattern}}</title>
<style>
@page{size:A4;margin:15mm}
body{font-family:system-ui,sans-serif;color:#000}
h1{font-size:1.2rem;border-bottom:2px solid #000;padding-bottom:2mm}
table{width:100%;border-collapse:collapse;margin-top:4mm}
th,td{border:1px solid #000;padding:2mm 3mm;text-align:left;vertical-align:top}
th{width:30%;background:#eee;font-weight:600}
footer{margin-top:6mm;font-size:.75rem;color:#444;border-top:1px solid #999;padding-top:2mm}
</style>
</head>
<body>
<h1>{{.Pattern}}</h1>
<table>
{{range .Items}}<tr><th>{{.Label}}</th><td></td></tr>
{{end}}</table>
{{if .Hooks}}<section><h2>フック</h2><ul>{{range .Hooks}}<li>{{.Phase}}:{{range .Reads}} reads {{.}}{{end}}{{range .Uses}} uses {{.}}{{end}}{{range .Produces}} produces {{.}}{{end}}{{if .Failure}} on-failure {{.Failure}}{{end}}</li>{{end}}</ul></section>{{end}}
<footer>pattern: {{.Pattern}} / semantics: {{.SemanticFingerprint}} / format: tei.print/v0{{if .PrintTemplate}} template: {{.PrintTemplate}}{{end}}</footer>
</body>
</html>
`))

var headerDetailPrintTemplate = template.Must(template.New("header-detail-print").Parse(`<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
{{if .SemanticFingerprint}}<meta name="tei-semantic-fingerprint" content="{{.SemanticFingerprint}}">{{end}}
<title>{{.Pattern}}</title>
<style>
@page{size:A4;margin:15mm}
body{font-family:system-ui,sans-serif;color:#000}
h1{font-size:1.2rem;border-bottom:2px solid #000;padding-bottom:2mm}
h2{font-size:1rem;margin-top:6mm}
table{width:100%;border-collapse:collapse;margin-top:3mm}
th,td{border:1px solid #000;padding:2mm 3mm;text-align:left;vertical-align:top}
th{background:#eee;font-weight:600}
.header th{width:30%}
</style>
</head>
<body>
<h1>{{.Pattern}}</h1>
<table class="header">
{{range .HeaderItems}}<tr><th>{{.Label}}</th><td></td></tr>
{{end}}</table>
<h2>明細</h2>
{{if .PageBreak}}<p>{{.PageBreak.Rows}}行/ページ{{if .PageBreak.RepeatHeader}}、見出し繰返し{{end}}</p>{{end}}
<table>
<thead><tr>{{range .DetailColumns}}<th>{{.Label}}</th>{{end}}</tr></thead>
<tbody>
{{range .DetailRows}}<tr>{{range $.DetailColumns}}<td></td>{{end}}</tr>
{{end}}</tbody>
</table>
{{if .Aggregates}}<h2>集計</h2>
<table>
{{range .Aggregates}}<tr><th>{{.Name}}</th><td>{{.Function}} {{.Label}} ({{.Type}})</td></tr>
{{end}}</table>{{end}}
<footer>pattern: {{.Pattern}} / semantics: {{.SemanticFingerprint}} / format: tei.print/v0{{if .PrintTemplate}} template: {{.PrintTemplate}}{{end}}</footer>
</body>
</html>
`))

// RenderPrintHTMLは、recordビューを印刷用の用紙面へ投影する。
func RenderPrintHTML(view View) (string, error) {
	var buf bytes.Buffer
	if err := printTemplate.Execute(&buf, view); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderHeaderDetailPrintHTMLは、header-detailビューを印刷用の用紙面へ投影する。
func RenderHeaderDetailPrintHTML(view HeaderDetailView) (string, error) {
	var buf bytes.Buffer
	if err := headerDetailPrintTemplate.Execute(&buf, view); err != nil {
		return "", err
	}
	return buf.String(), nil
}

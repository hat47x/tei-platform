package applicationview

import (
	"bytes"
	"html/template"
)

var browseTemplate = template.Must(template.New("browse-view").Parse(`<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
{{if .SemanticFingerprint}}<meta name="tei-semantic-fingerprint" content="{{.SemanticFingerprint}}">{{end}}
<title>{{.Pattern}}</title>
<style>
body{font-family:system-ui,sans-serif;max-width:60rem;margin:2rem auto;padding:0 1rem;color:#222}
h1{font-size:1.35rem;margin-bottom:.25rem}
small{font-weight:normal;color:#666}
table{width:100%;border-collapse:collapse;margin-top:1rem}
th,td{border:1px solid #bbb;padding:.4rem .5rem;text-align:left;font-size:.9rem}
th{background:#f3f3f3}
.filters{display:flex;gap:1rem;flex-wrap:wrap;margin:1rem 0}
.filters label{display:flex;gap:.35rem;align-items:center}
</style>
</head>
<body>
<h1>{{.Pattern}} <small>({{.Kind}})</small></h1>
<p>data: {{.Data}} — query: {{.Query}}</p>
{{if .Packages}}<p>packages: {{range .Packages}}{{.ID}} ^{{.Version}} {{end}}</p>{{end}}
{{if .PageSize}}<p>page-size: {{.PageSize}}</p>{{end}}
{{if .Sorts}}<p>並び順: {{range .Sorts}}{{.Label}} {{.Direction}}{{end}}</p>{{end}}
{{if .Filters}}<form class="filters">{{range .Filters}}<label>{{.Label}}{{if .Parameter}} ({{.Parameter}}){{end}} <select>{{range .Operators}}<option>{{.}}</option>{{end}}</select><input type="{{.Input}}" name="{{.Attribute}}"></label>{{end}}</form>{{end}}
<table>
<thead><tr>{{range .Columns}}<th>{{.Label}}</th>{{end}}</tr></thead>
<tbody>
<tr>{{range .Columns}}<td></td>{{end}}</tr>
<tr>{{range .Columns}}<td></td>{{end}}</tr>
<tr>{{range .Columns}}<td></td>{{end}}</tr>
<tr>{{range .Columns}}<td></td>{{end}}</tr>
<tr>{{range .Columns}}<td></td>{{end}}</tr>
</tbody>
</table>
<footer>pattern: {{.Pattern}} / semantics: {{.SemanticFingerprint}} / format: tei.print/v0{{if .PrintTemplate}} template: {{.PrintTemplate}}{{end}}</footer>
</body>
</html>
`))

var browsePrintTemplate = template.Must(template.New("browse-print").Parse(`<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
{{if .SemanticFingerprint}}<meta name="tei-semantic-fingerprint" content="{{.SemanticFingerprint}}">{{end}}
<title>{{.Pattern}}</title>
<style>
@page{size:A4 landscape;margin:12mm}
body{font-family:system-ui,sans-serif;color:#000}
h1{font-size:1.1rem;border-bottom:2px solid #000;padding-bottom:2mm}
table{width:100%;border-collapse:collapse;margin-top:4mm}
th,td{border:1px solid #000;padding:1.5mm 2mm;text-align:left;font-size:.85rem}
th{background:#eee}
footer{margin-top:6mm;font-size:.75rem;color:#444;border-top:1px solid #999;padding-top:2mm}
</style>
</head>
<body>
<h1>{{.Pattern}}</h1>
<table>
<thead><tr>{{range .Columns}}<th>{{.Label}}</th>{{end}}</tr></thead>
<tbody>
<tr>{{range .Columns}}<td></td>{{end}}</tr>
<tr>{{range .Columns}}<td></td>{{end}}</tr>
<tr>{{range .Columns}}<td></td>{{end}}</tr>
<tr>{{range .Columns}}<td></td>{{end}}</tr>
<tr>{{range .Columns}}<td></td>{{end}}</tr>
<tr>{{range .Columns}}<td></td>{{end}}</tr>
<tr>{{range .Columns}}<td></td>{{end}}</tr>
<tr>{{range .Columns}}<td></td>{{end}}</tr>
</tbody>
</table>
<footer>pattern: {{.Pattern}} / semantics: {{.SemanticFingerprint}} / format: tei.print/v0{{if .PrintTemplate}} template: {{.PrintTemplate}}{{end}}</footer>
</body>
</html>
`))

// RenderBrowseHTMLは、browseビューを一覧のHTMLへ投影する。
func RenderBrowseHTML(view BrowseView) (string, error) {
	var buf bytes.Buffer
	if err := browseTemplate.Execute(&buf, view); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderBrowsePrintHTMLは、browseビューを横向きA4の一覧へ投影する。
func RenderBrowsePrintHTML(view BrowseView) (string, error) {
	var buf bytes.Buffer
	if err := browsePrintTemplate.Execute(&buf, view); err != nil {
		return "", err
	}
	return buf.String(), nil
}

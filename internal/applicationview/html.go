package applicationview

import (
	"bytes"
	"html/template"
)

var htmlTemplate = template.Must(template.New("application-view").Parse(`<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<title>{{.Pattern}}</title>
<style>
body{font-family:system-ui,sans-serif;max-width:52rem;margin:2rem auto;padding:0 1rem;color:#222}
h1{font-size:1.35rem;margin-bottom:.25rem}
small{font-weight:normal;color:#666}
.entry{display:grid;grid-template-columns:14rem 1fr;gap:.35rem 1rem;align-items:center;margin:.4rem 0}
.entry label{text-align:right;color:#333}
.entry input{max-width:24rem;padding:.25rem}
.lookup{grid-column:2;color:#666;font-size:.85rem}
nav{margin-top:2rem;border-top:1px solid #ddd;padding-top:1rem}
</style>
</head>
<body>
<h1>{{.Pattern}} <small>({{.Kind}})</small></h1>
<p>data: {{.Data}}</p>
<form>
{{range .Items}}<div class="entry">
<label for="{{.Attribute}}">{{.Label}}</label>
<input id="{{.Attribute}}" name="{{.Attribute}}" type="{{.Input}}"{{if .Required}} required{{end}}{{if or .Immutable .Lookup}} readonly{{end}}>
{{if .Lookup}}<span class="lookup">参照: {{.Lookup.Pattern}} → {{.Lookup.TargetAttribute}} ({{.Lookup.TargetType}})</span>{{end}}
</div>
{{end}}</form>
{{if .Navigations}}<nav>
<h2>遷移元</h2>
<ul>{{range .Navigations}}<li>{{.From}} ({{.Kind}}){{if .Data}} — data: {{.Data}}{{end}}</li>{{end}}</ul>
</nav>{{end}}
</body>
</html>
`))

// RenderHTMLは、解決済みの対話ビューを、自己完結したHTMLのフォームへ投影する。
// 表示名・入力部品・参照先はViewが持つ解決結果を使い、ここで意味を推測しない。
func RenderHTML(view View) (string, error) {
	var buf bytes.Buffer
	if err := htmlTemplate.Execute(&buf, view); err != nil {
		return "", err
	}
	return buf.String(), nil
}

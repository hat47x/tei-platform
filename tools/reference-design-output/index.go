package designoutput

import (
	"bytes"
	"html/template"
)

const indexTemplate = `<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<title>TEI design output</title>
</head>
<body>
<h1>TEI design output</h1>
<p>sourceRevision: <code>{{.SourceRevision}}</code></p>
<p>provider: <code>{{.Provider}}</code></p>
<p>このページは成果物の一覧です。意味の正本ではなく、成果物を編集してもWorkspaceへ反映されません。</p>
<h2>Artifacts</h2>
<ul>
{{range .Artifacts}}<li><a href="{{.Path}}">{{.Name}}</a> ({{.Kind}}, {{.Bytes}} bytes, <code>{{.Digest}}</code>)</li>
{{end}}</ul>
{{if .Unsupported}}<h2>Unsupported</h2>
<ul>
{{range .Unsupported}}<li>{{.Name}}: {{.Reason}}</li>
{{end}}</ul>
{{end}}
<h2>Notes</h2>
<ul>
{{range .Notes}}<li>{{.}}</li>
{{end}}</ul>
<p><a href="manifest.json">manifest.json</a></p>
</body>
</html>
`

func renderIndex(manifest Manifest) []byte {
	var buffer bytes.Buffer
	if err := template.Must(template.New("index").Parse(indexTemplate)).Execute(&buffer, manifest); err != nil {
		return []byte("<!doctype html><html lang=\"ja\"><body><p>index generation failed</p></body></html>\n")
	}
	return buffer.Bytes()
}

package designoutput

import (
	"bytes"
	"html/template"
)

const indexTemplate = `<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<title>TEI 設計成果物</title>
</head>
<body>
<h1>TEI 設計成果物</h1>
<p>sourceRevision: <code>{{.SourceRevision}}</code></p>
<p>provider: <code>{{.Provider}}</code></p>
<p>このページは成果物の一覧です。意味の正本ではなく、成果物を編集してもWorkspaceへ反映されません。</p>
<p>この一覧の保存は、bundle全体の承認を意味しません。</p>
<h2>受理済みWorldの宣言から導出した文書・DDL</h2>
<p>文書の内容はモデルの宣言です。実際の副作用や実行結果を観測したものではありません。DDLは新規作成用であり、既存DBへの移行文ではありません。</p>
{{template "artifacts" .Documents}}
{{if .Candidates}}<h2>未選抜コード候補と参照用契約</h2>
<p>コード候補はbuild・実行・選抜していません。契約の複合規則はreference-localであり、業務意味の受理や文書との意味的一致、実行権限を証明しません。</p>
<p>生成器ID・宣言版・strategy・各digestは <code>generated/candidate.json</code>、入力契約の原文は <code>generated/contract.json</code> で確認できます。宣言版は生成器の現在の資格を示しません。</p>
{{template "artifacts" .Candidates}}
{{end}}
{{if .Other}}<h2>その他の成果物（種別を確認してください）</h2>
{{template "artifacts" .Other}}
{{end}}
{{if .Unsupported}}<h2>出力できない項目（Unsupported）</h2>
<ul>
{{range .Unsupported}}<li>{{.Name}}: {{.Reason}}</li>
{{end}}</ul>
{{end}}
<h2>確認範囲と注意点</h2>
<ul>
{{range .Notes}}<li>{{.}}</li>
{{end}}</ul>
<p><a href="manifest.json">manifest.json</a></p>
</body>
</html>
{{define "artifacts"}}<ul>
{{range .}}<li><a href="{{.Path}}">{{.Name}}</a>
{{if .Unresolved}}<p>未解決：</p><ul>{{range .Unresolved}}<li>{{.}}</li>{{end}}</ul>{{end}}
<details><summary>版とファイル情報</summary>
<p>種別: <code>{{.Kind}}</code> / {{.Bytes}} bytes</p>
<p>digest: <code>{{.Digest}}</code></p>
{{if .StructureRevision}}<p>structureRevision: <code>{{.StructureRevision}}</code></p>{{end}}
</details></li>
{{end}}</ul>{{end}}
`

func renderIndex(manifest Manifest) []byte {
	view := struct {
		Manifest
		Documents, Candidates, Other []Artifact
	}{Manifest: manifest}
	for _, artifact := range manifest.Artifacts {
		switch artifact.Kind {
		case "table-definition-document", "function-definition-document", "declared-access-matrix", "data-access-matrix", "ddl-script":
			view.Documents = append(view.Documents, artifact)
		case "unqualified-source-candidate", "generator-candidate-manifest", "reference-generator-contract":
			view.Candidates = append(view.Candidates, artifact)
		default:
			view.Other = append(view.Other, artifact)
		}
	}
	var buffer bytes.Buffer
	if err := template.Must(template.New("index").Parse(indexTemplate)).Execute(&buffer, view); err != nil {
		return []byte("<!doctype html><html lang=\"ja\"><body><p>index generation failed</p></body></html>\n")
	}
	return buffer.Bytes()
}

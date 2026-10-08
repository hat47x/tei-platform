package main

import (
	"bytes"
	"html/template"
)

// renderReviewHTMLは、Review ProjectionをA1〜A8の順に並べた単一のHTML documentとして描画する。
//
// これは人が読むための描画であり、承認そのものでもAcceptance authorityでもない。
// 表示する値はcandidate・workspace・Evidenceに由来するため、すべてhtml/templateで文脈に応じてescapeする。
// 外部のscript / stylesheet / font / imageは使わず、JavaScriptなしで読める。
// mapを走査せず、時刻や乱数も含めないため、同じProjectionからは同じbyte列を返す。
func renderReviewHTML(review ReviewProjection) (string, error) {
	var out bytes.Buffer
	if err := reviewHTMLTemplate.Execute(&out, review); err != nil {
		return "", err
	}
	return out.String(), nil
}

var reviewHTMLTemplate = template.Must(template.New("review").Funcs(template.FuncMap{
	"coverageLabel": reviewCoverageLabel,
}).Parse(reviewHTMLSource))

const reviewHTMLSource = `<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Shared Acceptance Review</title>
<style>
:root {
  color-scheme: light dark;
  --bg: #f5f5f2;
  --surface: #ffffff;
  --text: #1f1f1c;
  --muted: #5c5c57;
  --line: #d7d6cf;
  --code-bg: #eeede8;
  --ok: #1d6b3b;
  --partial: #7a5200;
  --missing: #8a2b2b;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #161714;
    --surface: #1f201d;
    --text: #e8e7e1;
    --muted: #aeada5;
    --line: #3a3b36;
    --code-bg: #2a2b27;
    --ok: #8fd3a6;
    --partial: #e8c27a;
    --missing: #f0a3a3;
  }
}
* { box-sizing: border-box; }
body {
  margin: 0;
  background: var(--bg);
  color: var(--text);
  font-family: system-ui, -apple-system, "Segoe UI", "Hiragino Sans", "Noto Sans JP", sans-serif;
  line-height: 1.65;
}
main { max-width: 56rem; margin: 0 auto; padding: 16px; }
h1 { font-size: 1.5rem; margin: 0.4em 0 0.6em; }
h2 { font-size: 1.05rem; margin: 0 0 0.5em; overflow-wrap: anywhere; }
p { margin: 0.5em 0; }
code {
  font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace;
  font-size: 0.88em;
  background: var(--code-bg);
  padding: 0.05em 0.3em;
  border-radius: 4px;
  overflow-wrap: anywhere;
  word-break: break-all;
}
.notice {
  background: var(--surface);
  border: 1px solid var(--line);
  border-left: 4px solid var(--partial);
  border-radius: 6px;
  padding: 10px 14px;
  color: var(--muted);
}
.card {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: 8px;
  padding: 14px 16px;
  margin: 14px 0;
}
dl.identity {
  display: grid;
  grid-template-columns: minmax(9rem, max-content) minmax(0, 1fr);
  gap: 8px 16px;
  margin: 0;
}
dl.identity dt { color: var(--muted); }
dl.identity dd { margin: 0; overflow-wrap: anywhere; }
dl.identity dd ul { margin: 0.2em 0; }
.note { color: var(--muted); font-size: 0.93em; }
.badge {
  display: inline-block;
  font-size: 0.78rem;
  font-weight: 600;
  padding: 0 0.6em;
  margin-left: 0.4em;
  border: 1px solid currentColor;
  border-radius: 999px;
  vertical-align: 0.1em;
  white-space: nowrap;
}
[data-coverage="available"] .badge { color: var(--ok); }
[data-coverage="partial"] .badge { color: var(--partial); }
[data-coverage="not-evaluated"] .badge { color: var(--missing); }
ul { margin: 0.4em 0; padding-left: 1.3em; }
li { margin: 0.25em 0; overflow-wrap: anywhere; }
.summary { font-weight: 600; }
.sources, .section-note { color: var(--muted); font-size: 0.93em; }
.toc { display: flex; flex-wrap: wrap; gap: 6px 14px; padding: 0; list-style: none; margin: 0.6em 0; }
.toc a { color: inherit; }
footer { color: var(--muted); font-size: 0.85rem; margin: 24px 0 8px; overflow-wrap: anywhere; }
@media (max-width: 36rem) {
  dl.identity { grid-template-columns: 1fr; gap: 2px 0; }
  dl.identity dt { margin-top: 8px; }
}
</style>
</head>
<body>
<main>
<header>
<h1>Shared Acceptance Review</h1>
<p class="notice">このページは、Shared Acceptance Previewの結果を人が読めるように並べ直した表示です。承認そのものではなく、受入（<code>accept</code>）の権限もありません。受入は別のコマンドで行い、このページからは行えません。</p>
</header>

<section class="card" aria-labelledby="identity-title">
<h2 id="identity-title">対象と機械判定</h2>
<dl class="identity">
{{with .CandidateID}}<dt>candidate</dt><dd>{{.}}</dd>{{end}}
{{with .Intent}}<dt>intent</dt><dd>{{.}}</dd>{{end}}
{{with .CandidateFingerprint}}<dt>candidate fingerprint</dt><dd><code>{{.}}</code></dd>{{end}}
{{with .EvidenceFingerprint}}<dt>evidence fingerprint</dt><dd><code>{{.}}</code></dd>{{end}}
{{with .ReviewFingerprint}}<dt>review fingerprint</dt><dd><code data-field="review-fingerprint">{{.}}</code><p class="note">review fingerprintは、この表示の内容へ受理を結び付けるidentityです。一致しても、人が実際に読んで承認したことを証明するものではありません。</p></dd>{{end}}
<dt>machine preview</dt><dd>{{if .PreviewAcceptable}}受入可能{{else}}受入不可{{end}}（Shared Acceptanceの機械判定だけを表します）</dd>
<dt>coverage</dt><dd>表示可能 {{.Coverage.Available}} / 一部 {{.Coverage.Partial}} / 未評価 {{.Coverage.NotEvaluated}}</dd>
{{with .Coverage.Incomplete}}<dt>確認が残るsection</dt><dd><ul>{{range .}}<li>{{.Code}} {{.Title}} [{{coverageLabel .Coverage}}]</li>{{end}}</ul></dd>{{end}}
</dl>
</section>

<nav aria-label="A1からA8への目次">
<ol class="toc">
{{range .Sections}}<li><a href="#section-{{.Code}}">{{.Code}} {{.Title}}</a> <span>[{{coverageLabel .Coverage}}]</span></li>
{{end}}</ol>
</nav>

{{range .Sections}}
<section class="card" id="section-{{.Code}}" data-coverage="{{.Coverage}}">
<h2>{{.Code}} {{.Title}}<span class="badge">{{coverageLabel .Coverage}}</span></h2>
{{with .Summary}}<p class="summary">要約: {{.}}</p>{{end}}
{{with .Items}}<ul>{{range .}}<li>{{.}}</li>{{end}}</ul>{{end}}
{{with .Sources}}<p class="sources">根拠: {{range $i, $source := .}}{{if $i}}, {{end}}<code>{{$source}}</code>{{end}}</p>{{end}}
{{with .Note}}<p class="section-note">注記: {{.}}</p>{{end}}
</section>
{{end}}

<footer>
<p>この表示は読み取り専用のreview commandが生成しました。承認の記録ではありません。</p>
<p>projection schema: {{.Schema}}</p>
</footer>
</main>
</body>
</html>
`

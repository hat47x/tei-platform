package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"sort"
)

// destinationReadHTML is the self-contained page `tei view -environment
// -format html` returns: one row block per detail row, each answered rule
// printed beside it. It exists because plan/NOW.md's road item 2 asks for the
// per-row answers to be visible in a browser rather than only in JSON or a
// terminal (#1119, #2586).
//
// The page is a rendering of the same destinationRead the text and JSON forms
// print. It adds no rule, no coverage claim and no value: every answer and
// every undecided reason comes from the read.
var destinationReadTemplate = template.Must(template.New("destination-read").Parse(`<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>TEI view -environment — {{.Read.Pattern}}</title>
<style>
:root{color-scheme:light dark;font-family:system-ui,-apple-system,"Segoe UI",sans-serif}
body{margin:0;line-height:1.55}
main{max-width:72rem;margin:0 auto;padding:2rem 1.5rem 4rem}
h1{font-size:1.35rem;margin-bottom:.25rem}
h2{font-size:1.05rem;margin-top:2rem}
.meta{display:grid;grid-template-columns:repeat(auto-fit,minmax(14rem,1fr));gap:.35rem 1.25rem;margin:1rem 0}
.meta span{color:#666;font-variant-numeric:tabular-nums}
table{width:100%;border-collapse:collapse;margin-top:.75rem}
th,td{border:1px solid #bbb;padding:.35rem .6rem;text-align:left;vertical-align:top;font-size:.9rem}
th{background:rgba(127,127,127,.12)}
tr.answer-satisfied td:first-child{font-weight:600}
.empty{color:#666;font-style:italic}
</style>
</head>
<body>
<main>
<h1>{{.Read.Pattern}}</h1>
<p class="empty">{{.Read.Application}}</p>
<div class="meta">
<span>relation: {{.Read.Relation}}</span>
<span>environment: {{.Read.Environment}}</span>
<span>coverage: {{.Read.Coverage}}</span>
<span>rows: {{len .Read.Rows}}</span>
</div>
{{if .Declaration}}<p><strong>declaration:</strong> {{.Declaration}}</p>{{end}}
<h2>行ごとの答え</h2>
{{if .Rows}}
{{range .Rows}}
<h3>{{.Key}}</h3>
{{if .Answers}}
<table>
<thead><tr><th>rule</th><th>answer</th><th>reason</th></tr></thead>
<tbody>
{{range .Answers}}<tr class="answer-{{.Class}}"><td>{{.Rule}}</td><td>{{.Answer}}</td><td>{{.Reason}}</td></tr>
{{end}}</tbody>
</table>
{{else}}<p class="empty">この行に答えはない。</p>{{end}}
{{end}}
{{else}}<p class="empty">明細の行はない。</p>{{end}}
{{if .RelationAnswers}}
<h2>関係全体の答え</h2>
<table>
<thead><tr><th>rule</th><th>answer</th><th>reason</th></tr></thead>
<tbody>
{{range .RelationAnswers}}<tr class="answer-{{.Class}}"><td>{{.Rule}}</td><td>{{.Answer}}</td><td>{{.Reason}}</td></tr>
{{end}}</tbody>
</table>
{{end}}
</main>
</body>
</html>
`))

// destinationReadPage is the template's data: the read plus the rule answers
// already ordered, so the template holds no logic and a test can compare the
// rendered page against the read.
type destinationReadPage struct {
	Read            destinationRead
	Rows            []destinationReadRowPage
	RelationAnswers []destinationReadAnswerPage
	Declaration     string
}

type destinationReadRowPage struct {
	Key     string
	Answers []destinationReadAnswerPage
}

type destinationReadAnswerPage struct {
	Rule   string
	Answer string
	Reason string
	// Class is "satisfied", "not-satisfied", "undecided" or "other", taken
	// from the answer the evaluation recorded. It exists for styling only and
	// is never inferred from the reason.
	Class string
}

func destinationReadPageOf(read destinationRead) destinationReadPage {
	page := destinationReadPage{Read: read}
	if read.Declaration != nil {
		// A binding that states both "every row" and "only some rows" is
		// self-contradictory. The static validator rejects it, but this path
		// runs no validator, so the page must not pick one side and hide the
		// other: printing only the relation would read as a clean declaration
		// when the binding contradicts itself. The text form says the same
		// thing, and both renderings of one read must agree.
		var declared string
		switch {
		case read.Declaration.CoversRelation != "" && read.Declaration.CoversSome:
			declared = "covers " + string(read.Declaration.CoversRelation) + " and covers some (contradictory)"
		case read.Declaration.CoversRelation != "":
			declared = "covers " + string(read.Declaration.CoversRelation)
		default:
			declared = "covers some"
		}
		checked := "unchecked"
		if read.Declaration.Checked {
			checked = "checked"
		}
		page.Declaration = declared + " (" + checked + ")"
	}
	keys := make([]string, 0, len(read.Rows))
	for _, row := range read.Rows {
		keys = append(keys, row.Key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		page.Rows = append(page.Rows, destinationReadRowPage{
			Key:     key,
			Answers: destinationReadAnswers(read.PerRow[key], read.PerRowUndecided[key]),
		})
	}
	page.RelationAnswers = destinationReadAnswers(read.PerRelation, read.PerRelationUndecided)
	return page
}

// destinationReadAnswers orders the union of the answered and the undecided
// rules. A rule that has only a reason is reported as undecided, not as a blank
// answer, for the same reason the text form does so: silence would read as
// satisfied.
func destinationReadAnswers(answered, undecided map[string]string) []destinationReadAnswerPage {
	seen := map[string]bool{}
	rules := make([]string, 0, len(answered)+len(undecided))
	for rule := range answered {
		if !seen[rule] {
			seen[rule] = true
			rules = append(rules, rule)
		}
	}
	for rule := range undecided {
		if !seen[rule] {
			seen[rule] = true
			rules = append(rules, rule)
		}
	}
	sort.Strings(rules)
	answers := make([]destinationReadAnswerPage, 0, len(rules))
	for _, rule := range rules {
		answer := answered[rule]
		reason := undecided[rule]
		if answer == "" && reason != "" {
			answer = "undecided"
		}
		answers = append(answers, destinationReadAnswerPage{
			Rule:   rule,
			Answer: answer,
			Reason: reason,
			Class:  destinationReadAnswerClass(answer),
		})
	}
	return answers
}

func destinationReadAnswerClass(answer string) string {
	switch answer {
	case "satisfied":
		return "satisfied"
	case "not-satisfied":
		return "not-satisfied"
	case "undecided":
		return "undecided"
	}
	return "other"
}

// writeDestinationReadHTML renders the page. Every value goes through
// html/template, so a rule name or a field value from the Workspace cannot
// become markup.
func writeDestinationReadHTML(out io.Writer, read destinationRead) (int, error) {
	var buf bytes.Buffer
	if err := destinationReadTemplate.Execute(&buf, destinationReadPageOf(read)); err != nil {
		return 2, fmt.Errorf("render view -environment as HTML: %w", err)
	}
	if _, err := out.Write(buf.Bytes()); err != nil {
		return 2, err
	}
	return 0, nil
}

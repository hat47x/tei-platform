package main

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
)

// destinationHTMLCell is a presentation-only projection. It never changes
// the observed value state or attempts to decide a business rule.
type destinationHTMLCell struct {
	ID    string
	State string
	Text  string
}

type destinationHTMLDecision struct {
	ID     string
	Answer string
	Reason string
}

type destinationHTMLRow struct {
	ID        string
	Fields    []destinationHTMLCell
	Decisions []destinationHTMLDecision
}

type destinationHTMLRelationRule struct {
	ID       string
	Decision destinationHTMLDecision
}

type destinationHTMLPage struct {
	Application   string
	Pattern       string
	Environment   string
	Relation      string
	KeyAttributes []string
	Coverage      string
	FieldIDs      []string
	RuleIDs       []string
	Rows          []destinationHTMLRow
	RelationRules []destinationHTMLRelationRule
}

var destinationHTMLTemplate = template.Must(template.New("environment-rows").
	Funcs(template.FuncMap{"add": func(a, b int) int { return a + b }}).
	Parse(`<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>取得した行の判定 — {{.Pattern}}</title>
<style>
:root { font-family: system-ui, sans-serif; color-scheme: light dark; }
body { max-width: 80rem; margin: 2rem auto; padding: 0 1rem; line-height: 1.5; }
h1 { font-size: 1.4rem; }
.table-scroll { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; margin: 1rem 0; }
caption { text-align: left; font-weight: 600; }
th, td { border: 1px solid #9998; padding: .45rem .65rem; text-align: left; vertical-align: top; }
th { background: #7772; }
code { overflow-wrap: anywhere; }
[data-state="not-fetched"], [data-answer="undecided"] { font-style: italic; }
small { display: block; }
</style>
</head>
<body>
<main>
<h1>取得した行の判定</h1>
<p>Application: <code>{{.Application}}</code> / Pattern: <code>{{.Pattern}}</code> / Environment: <code>{{.Environment}}</code></p>
<p>Relation: <code>{{.Relation}}</code> / 主キー: {{range $i, $id := .KeyAttributes}}{{if $i}}, {{end}}<code>{{$id}}</code>{{else}}未定義{{end}}</p>
<p>取得範囲: <strong data-coverage="{{.Coverage}}">{{.Coverage}}</strong>{{if eq .Coverage "partial"}}（関係全体の取得は未確認です。未確定の判定を成立とみなさないでください。）{{end}}</p>
<h2>取得行と規則の判定</h2>
<div class="table-scroll"><table>
<caption>取得した行（{{len .Rows}}行）</caption>
<thead><tr><th scope="col">行キー</th>{{range .FieldIDs}}<th scope="col"><code>{{.}}</code></th>{{end}}{{range .RuleIDs}}<th scope="col"><code>{{.}}</code></th>{{end}}</tr></thead>
<tbody>
{{range .Rows}}<tr data-row-key="{{.ID}}"><th scope="row"><code>{{.ID}}</code></th>{{range .Fields}}<td data-attribute-id="{{.ID}}" data-state="{{.State}}">{{.Text}}</td>{{end}}{{range .Decisions}}<td data-rule-id="{{.ID}}" data-answer="{{.Answer}}">{{.Answer}}{{if .Reason}}<small>理由: {{.Reason}}</small>{{end}}</td>{{end}}</tr>
{{else}}<tr><td colspan="{{add (add 1 (len .FieldIDs)) (len .RuleIDs)}}">取得した行はありません。取得範囲が確定しているとは限りません。</td></tr>{{end}}
</tbody></table></div>
<h2>関係全体の判定</h2>
<div class="table-scroll"><table>
<thead><tr><th scope="col">規則</th><th scope="col">判定</th></tr></thead>
<tbody>{{range .RelationRules}}<tr><th scope="row"><code>{{.ID}}</code></th><td data-rule-id="{{.ID}}" data-answer="{{.Decision.Answer}}">{{.Decision.Answer}}{{if .Decision.Reason}}<small>理由: {{.Decision.Reason}}</small>{{end}}</td></tr>{{else}}<tr><td colspan="2">関係全体の規則はありません。</td></tr>{{end}}</tbody>
</table></div>
</main>
</body>
</html>
`))

// renderEnvironmentRowsHTML renders the existing CLI read-only result.
// Neither the Contract Query nor its rule evaluation is repeated here.
func renderEnvironmentRowsHTML(result destinationRead) (string, error) {
	fields := map[string]struct{}{}
	rules := map[string]struct{}{}
	for _, row := range result.Rows {
		for id := range row.Fields {
			fields[id] = struct{}{}
		}
	}
	for _, perRow := range result.PerRow {
		for id := range perRow {
			rules[id] = struct{}{}
		}
	}
	page := destinationHTMLPage{
		Application: string(result.Application),
		Pattern:     string(result.Pattern),
		Environment: string(result.Environment),
		Relation:    string(result.Relation),
		Coverage:    result.Coverage,
	}
	for _, id := range result.KeyAttributes {
		page.KeyAttributes = append(page.KeyAttributes, string(id))
	}
	for id := range fields {
		page.FieldIDs = append(page.FieldIDs, id)
	}
	for id := range rules {
		page.RuleIDs = append(page.RuleIDs, id)
	}
	sort.Strings(page.FieldIDs)
	sort.Strings(page.RuleIDs)
	for _, row := range result.Rows {
		item := destinationHTMLRow{ID: row.Key}
		for _, id := range page.FieldIDs {
			field, exists := row.Fields[id]
			cell := destinationHTMLCell{ID: id, State: "not-fetched", Text: "未取得"}
			if exists {
				cell.State = field.State
				switch field.State {
				case "present":
					cell.Text = fmt.Sprint(field.Value)
				case "null":
					cell.Text = "NULL"
				case "not-fetched":
					cell.Text = "未取得"
				default:
					return "", fmt.Errorf("unknown field state %q", field.State)
				}
			}
			item.Fields = append(item.Fields, cell)
		}
		for _, id := range page.RuleIDs {
			answer, exists := result.PerRow[row.Key][id]
			if !exists {
				return "", fmt.Errorf("missing decision for row %s and rule %s", row.Key, id)
			}
			item.Decisions = append(item.Decisions, destinationHTMLDecision{
				ID: id, Answer: answer, Reason: result.PerRowUndecided[row.Key][id],
			})
		}
		page.Rows = append(page.Rows, item)
	}
	for id, answer := range result.PerRelation {
		page.RelationRules = append(page.RelationRules, destinationHTMLRelationRule{
			ID: id,
			Decision: destinationHTMLDecision{
				ID: id, Answer: answer, Reason: result.PerRelationUndecided[id],
			},
		})
	}
	sort.Slice(page.RelationRules, func(i, j int) bool { return page.RelationRules[i].ID < page.RelationRules[j].ID })
	var out bytes.Buffer
	if err := destinationHTMLTemplate.Execute(&out, page); err != nil {
		return "", err
	}
	return out.String(), nil
}

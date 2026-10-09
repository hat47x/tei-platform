package main

import (
	"sort"
	"strings"

	"tei/internal/application"
	"tei/internal/ir"
)

// validationScopeReviewRow は、Application 全体の評価宣言と業務手順の明示参照を
// 比較するための読み取り専用の表示情報である。集合の差は不整合の証拠ではない。
// validationReferenceIssue は比較を保留した具体的な理由を指す。
// 適用範囲や実行時の成否の判定ではない。
type validationReferenceIssue struct {
	Side   string `json:"side"`
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

type validationScopeReviewRow struct {
	Step             string                   `json:"step"`
	Application      string                   `json:"applicationContext,omitempty"`
	ProcedureSource  string                   `json:"procedureSource,omitempty"`
	Status           string                   `json:"status"`
	ApplicationOnly  []string                 `json:"applicationOnly,omitempty"`
	ProcedureOnly    []string                 `json:"procedureOnly,omitempty"`
	Reason           string                   `json:"reason"`
	SourceEvidence   []validationReviewSource `json:"sourceEvidence,omitempty"`
	UnresolvedSource []string                 `json:"unresolvedSource,omitempty"`
	ReferenceIssues  []validationReferenceIssue `json:"referenceIssues,omitempty"`
}

// compareValidationScopes は業務手順の validation-support に限って比較する。
// fixture.Application は比較文脈の申告であり、Procedure の所有者や適用範囲を
// 確定する情報ではない。明示されていない Application を名前から推測しない。
func compareValidationScopes(world ir.World, f fixture) []validationScopeReviewRow {
	var rows []validationScopeReviewRow
	_, _, validationCounts, _ := semanticCounts(world)
	for _, s := range f.Steps {
		if s.Realization != "validation-support" {
			continue
		}
		row := validationScopeReviewRow{
			Step:        s.ID,
			Application: f.Application,
		}
		if strings.TrimSpace(f.Application) == "" {
			row.Status = "unresolved-context"
			row.Reason = "比較するApplication文脈が指定されていません。業務手順から所有Applicationを推測しません"
			rows = append(rows, row)
			continue
		}

		var matched application.Application
		count := 0
		for _, app := range world.Applications {
			if string(app.ID) == f.Application {
				matched = app
				count++
			}
		}
		if count != 1 {
			row.Status = "unresolved-context"
			row.Reason = "比較文脈のApplicationを正確に1件へ解決できません"
			rows = append(rows, row)
			continue
		}

		// 未解決参照に加え、Applicationや手順の宣言重複を集合化で消さない。
		// 参照の個数は暗黙に意味のある検査順序を担う可能性がある。
		inApplication := make(map[string]struct{}, len(matched.Validations))
		inProcedure := make(map[string]struct{}, len(s.Validations))
		issueSet := make(map[validationReferenceIssue]struct{})
		recordIssue := func(side, id, reason string) {
			issue := validationReferenceIssue{Side: side, ID: id, Reason: reason}
			if _, found := issueSet[issue]; found {
				return
			}
			issueSet[issue] = struct{}{}
			row.ReferenceIssues = append(row.ReferenceIssues, issue)
		}
		checkDefinition := func(side, id string) {
			switch validationCounts[id] {
			case 0:
				recordIssue(side, id, "missing-definition")
			case 1:
				// Unique source declaration.
			default:
				recordIssue(side, id, "ambiguous-definition")
			}
		}
		for _, id := range s.Validations {
			if _, seen := inProcedure[id]; seen {
				recordIssue("procedure", id, "duplicate-declaration")
			}
			inProcedure[id] = struct{}{}
			checkDefinition("procedure", id)
		}
		for _, id := range matched.Validations {
			name := string(id)
			if _, seen := inApplication[name]; seen {
				recordIssue("application", name, "duplicate-declaration")
			}
			inApplication[name] = struct{}{}
			checkDefinition("application", name)
		}
		if len(row.ReferenceIssues) > 0 {
			sort.Slice(row.ReferenceIssues, func(a, b int) bool {
				left, right := row.ReferenceIssues[a], row.ReferenceIssues[b]
				if left.Side != right.Side {
					return left.Side < right.Side
				}
				if left.ID != right.ID {
					return left.ID < right.ID
				}
				return left.Reason < right.Reason
			})
			row.Status = "unresolved-reference"
			row.Reason = "比較を保留したValidation参照とその理由をreferenceIssuesに示します。未解決の参照から集合差を確定しません"
			rows = append(rows, row)
			continue
		}
		for id := range inApplication {
			if _, ok := inProcedure[id]; !ok {
				row.ApplicationOnly = append(row.ApplicationOnly, id)
			}
		}
		for id := range inProcedure {
			if _, ok := inApplication[id]; !ok {
				row.ProcedureOnly = append(row.ProcedureOnly, id)
			}
		}
		sort.Strings(row.ApplicationOnly)
		sort.Strings(row.ProcedureOnly)
		if len(row.ApplicationOnly)+len(row.ProcedureOnly) > 0 {
			row.Status = "review-needed"
			row.Reason = "Application全体の宣言と業務手順の明示参照に差があります。適用範囲が同じかを人が確認する必要があり、差だけで不整合とは判定しません"
		} else {
			row.Status = "no-set-difference"
			row.Reason = "明示されたValidation集合に差はありません。業務上の適用範囲や実行結果の一致までは保証しません"
		}
		rows = append(rows, row)
	}
	return rows
}

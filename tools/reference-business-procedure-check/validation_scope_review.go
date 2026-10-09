package main

import (
	"sort"
	"strings"

	"tei/internal/application"
	"tei/internal/ir"
)

// validationScopeReviewRow は、Application 全体の評価宣言と業務手順の明示参照を
// 比較するための読み取り専用の表示情報である。集合の差は不整合の証拠ではない。
type validationScopeReviewRow struct {
	Step            string   `json:"step"`
	Application     string   `json:"applicationContext,omitempty"`
	Status          string   `json:"status"`
	ApplicationOnly []string `json:"applicationOnly,omitempty"`
	ProcedureOnly   []string `json:"procedureOnly,omitempty"`
	Reason          string   `json:"reason"`
	SourceEvidence   []validationReviewSource `json:"sourceEvidence,omitempty"`
	UnresolvedSource []string `json:"unresolvedSource,omitempty"`
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
		unresolved := false
		for _, id := range s.Validations {
			if _, seen := inProcedure[id]; seen {
				unresolved = true
			}
			inProcedure[id] = struct{}{}
			if validationCounts[id] != 1 {
				unresolved = true
			}
		}
		for _, id := range matched.Validations {
			name := string(id)
			if _, seen := inApplication[name]; seen {
				unresolved = true
			}
			inApplication[name] = struct{}{}
			if validationCounts[name] != 1 {
				unresolved = true
			}
		}
		if unresolved {
			row.Status = "unresolved-reference"
			row.Reason = "Validation参照が重複しているか正確に1件へ解決できません。集合差の判定を保留します"
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

package main

import (
	"encoding/json"
	"reflect"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/workspace"
)

func TestValidationScopeReviewKeepsOriginalCheckerAndWorkspaceUnchanged(t *testing.T) {
	workspacePath, fixturePath, probePath := ref01Paths()
	before := ref01TreeDigest(t, workspacePath)

	baseline, err := run(workspacePath, fixturePath, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(baseline.ValidationScopeReview) != 0 {
		t.Fatalf("任意表示を指定しないbaselineに比較結果が混入しました: %+v", baseline.ValidationScopeReview)
	}

	reviewed, err := runWithOptions(workspacePath, fixturePath, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reviewed.Diagnostics, baseline.Diagnostics) {
		t.Fatalf("比較表示が既存のblocking判定を書き換えました: before=%+v after=%+v", baseline.Diagnostics, reviewed.Diagnostics)
	}
	if len(reviewed.ValidationScopeReview) != 1 {
		t.Fatalf("validation-supportの比較は1件のはずです: %+v", reviewed.ValidationScopeReview)
	}
	row := reviewed.ValidationScopeReview[0]
	if row.Step != "BP-05" || row.Application != "order_entry" || row.Status != "review-needed" {
		t.Fatalf("比較文脈または判定が不正です: %+v", row)
	}
	if !reflect.DeepEqual(row.ApplicationOnly, []string{"sales.order_line.line_number_positive"}) || len(row.ProcedureOnly) != 0 {
		t.Fatalf("受理済みValidationと業務手順の差を検出できません: %+v", row)
	}

	probed, err := runWithOptions(workspacePath, fixturePath, probePath, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkExpectedDiagnostics(probed.Diagnostics, "BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT"); err != nil {
		t.Fatalf("Customer編集のblocking判定が変わりました: %v", err)
	}
	if !reflect.DeepEqual(probed.ValidationScopeReview, reviewed.ValidationScopeReview) {
		t.Fatalf("別のProbeでValidation比較結果が変わりました: %+v", probed.ValidationScopeReview)
	}
	if after := ref01TreeDigest(t, workspacePath); after != before {
		t.Fatal("読み取り専用の比較でReference Workspaceを書き換えました")
	}
}

func TestValidationScopeReviewNeverInfersMissingOrAmbiguousApplication(t *testing.T) {
	workspacePath, fixturePath, _ := ref01Paths()
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}

	missingContext := f
	missingContext.Application = ""
	rows := compareValidationScopes(loaded.World, missingContext)
	if len(rows) != 1 || rows[0].Status != "unresolved-context" || len(rows[0].ApplicationOnly) != 0 {
		t.Fatalf("Application未指定から意味を推測しました: %+v", rows)
	}

	ambiguousWorld := loaded.World
	for _, app := range loaded.World.Applications {
		if string(app.ID) == f.Application {
			ambiguousWorld.Applications = append(append([]application.Application(nil), loaded.World.Applications...), app)
			break
		}
	}
	rows = compareValidationScopes(ambiguousWorld, f)
	if len(rows) != 1 || rows[0].Status != "unresolved-context" || len(rows[0].ApplicationOnly) != 0 {
		t.Fatalf("Application ID重複から任意の一件を選びました: %+v", rows)
	}
}

func TestValidationScopeReviewTreatsUnknownReferenceAsUnresolved(t *testing.T) {
	workspacePath, fixturePath, _ := ref01Paths()
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	for i := range f.Steps {
		if f.Steps[i].ID == "BP-05" {
			f.Steps[i].Validations = append(f.Steps[i].Validations, "sales.nonexistent.validation")
		}
	}
	rows := compareValidationScopes(loaded.World, f)
	if len(rows) != 1 || rows[0].Status != "unresolved-reference" ||
		len(rows[0].ApplicationOnly) != 0 || len(rows[0].ProcedureOnly) != 0 {
		t.Fatalf("未解決参照を集合差の確定結果にしました: %+v", rows)
	}
}

func TestValidationScopeReviewEqualSetsDoNotClaimBusinessEquivalence(t *testing.T) {
	workspacePath, fixturePath, _ := ref01Paths()
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	for i := range f.Steps {
		if f.Steps[i].ID == "BP-05" {
			f.Steps[i].Validations = append(f.Steps[i].Validations, "sales.order_line.line_number_positive")
		}
	}
	rows := compareValidationScopes(loaded.World, f)
	if len(rows) != 1 || rows[0].Status != "no-set-difference" {
		t.Fatalf("同じ参照集合の扱いが不正です: %+v", rows)
	}
	if rows[0].Reason == "" {
		t.Fatal("意味の適用範囲は別途確認が必要である旨を表示していません")
	}
}

func TestValidationScopeReviewDoesNotHideDuplicateDeclarations(t *testing.T) {
	workspacePath, fixturePath, _ := ref01Paths()
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("procedure", func(t *testing.T) {
		duplicate := f
		duplicate.Steps = append([]step(nil), f.Steps...)
		found := false
		for i := range duplicate.Steps {
			if duplicate.Steps[i].ID == "BP-05" {
				original := duplicate.Steps[i].Validations
				duplicate.Steps[i].Validations = append(append([]string(nil), original...), original[0])
				found = true
			}
		}
		if !found {
			t.Fatal("検証対象BP-05がありません")
		}
		rows := compareValidationScopes(loaded.World, duplicate)
		if len(rows) != 1 || rows[0].Status != "unresolved-reference" {
			t.Fatalf("Procedure内の重複を集合で消しました: %+v", rows)
		}
	})

	t.Run("application", func(t *testing.T) {
		duplicate := loaded.World
		duplicate.Applications = append([]application.Application(nil), loaded.World.Applications...)
		found := false
		for i := range duplicate.Applications {
			if string(duplicate.Applications[i].ID) == f.Application {
				values := duplicate.Applications[i].Validations
				duplicate.Applications[i].Validations = append(values[:len(values):len(values)], values[0])
				found = true
			}
		}
		if !found {
			t.Fatal("比較対象Applicationがありません")
		}
		rows := compareValidationScopes(duplicate, f)
		if len(rows) != 1 || rows[0].Status != "unresolved-reference" {
			t.Fatalf("Application内の重複を集合で消しました: %+v", rows)
		}
	})
}

func TestValidationScopeReviewShowsBothDifferenceDirections(t *testing.T) {
	workspacePath, fixturePath, _ := ref01Paths()
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	world := loaded.World
	world.Applications = append([]application.Application(nil), world.Applications...)
	found := false
	for i := range world.Applications {
		if string(world.Applications[i].ID) == f.Application {
			values := world.Applications[i].Validations
			world.Applications[i].Validations = append([]core.CanonicalID(nil), values[:2]...)
			found = true
		}
	}
	if !found {
		t.Fatal("比較対象Applicationがありません")
	}
	for i := range f.Steps {
		if f.Steps[i].ID == "BP-05" {
			f.Steps[i].Validations = []string{
				"sales.order.lines.line_number_unique",
				"sales.order_line.line_number_positive",
			}
		}
	}
	rows := compareValidationScopes(world, f)
	if len(rows) != 1 || rows[0].Status != "review-needed" ||
		!reflect.DeepEqual(rows[0].ApplicationOnly, []string{"sales.order_line.quantity_positive"}) ||
		!reflect.DeepEqual(rows[0].ProcedureOnly, []string{"sales.order_line.line_number_positive"}) {
		t.Fatalf("双方向のValidation参照差を説明できません: %+v", rows)
	}
}

func TestValidationScopeReviewJSONIsOptIn(t *testing.T) {
	workspacePath, fixturePath, _ := ref01Paths()
	baseline, err := run(workspacePath, fixturePath, "")
	if err != nil {
		t.Fatal(err)
	}
	existingJSON, err := json.Marshal(baseline)
	if err != nil {
		t.Fatal(err)
	}
	var existing map[string]json.RawMessage
	if err := json.Unmarshal(existingJSON, &existing); err != nil {
		t.Fatal(err)
	}
	if _, ok := existing["validationScopeReview"]; ok {
		t.Fatal("明示指定なしで比較結果が出力されました")
	}
	reviewed, err := runWithOptions(workspacePath, fixturePath, "", true)
	if err != nil {
		t.Fatal(err)
	}
	reviewedJSON, err := json.Marshal(reviewed)
	if err != nil {
		t.Fatal(err)
	}
	var withReview map[string]json.RawMessage
	if err := json.Unmarshal(reviewedJSON, &withReview); err != nil {
		t.Fatal(err)
	}
	if len(withReview["validationScopeReview"]) == 0 {
		t.Fatal("明示指定時にValidation比較結果が出力されません")
	}
	delete(withReview, "validationScopeReview")
	if !reflect.DeepEqual(withReview, existing) {
		t.Fatalf("任意表示が従来のJSON項目を書き換えました: before=%v after=%v", existing, withReview)
	}
}

package changeanalysis

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

// referenceWorkspaceCopyBeforeSecondApplicationは、第二Application `repair_intake` を受理する前の
// Reference Workspaceを返す。受理後のReferenceにはそのsourceが含まれるため、「第二Applicationを追加する」
// 差分を作るtestはこちらを起点にする。
func referenceWorkspaceCopyBeforeSecondApplication(t *testing.T) string {
	t.Helper()
	dir := referenceWorkspaceCopy(t)
	for _, rel := range []string{
		filepath.Join("semantic", "repair-intake"),
		filepath.Join("applications", "repair-intake"),
	} {
		if err := os.RemoveAll(filepath.Join(dir, rel)); err != nil {
			t.Fatal(err)
		}
	}
	// S7のEnvironment bindingも外す。Applicationを消してbindingだけ残すと、
	// 存在しないApplicationへbindしたWorldという別の不整合になる。
	for _, rel := range []string{
		filepath.Join("environments", "development", "environment.tei"),
		filepath.Join("environments", "production", "environment.tei"),
	} {
		path := filepath.Join(dir, rel)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var kept []string
		for _, line := range strings.Split(string(content), "\n") {
			if !strings.Contains(line, "bind repair_intake.customer_master") {
				kept = append(kept, line)
			}
		}
		if err := os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func referenceWorkspaceCopy(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source pathを取得できません")
	}
	src := filepath.Join(filepath.Dir(file), "..", "..", "..", "reference", "workspace")
	dst := filepath.Join(t.TempDir(), "workspace")
	err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o644)
	})
	if err != nil {
		t.Fatalf("Reference workspaceを複製できません: %v", err)
	}
	return dst
}

func loadWorld(t *testing.T, dir string) workspace.Result {
	t.Helper()
	result, err := workspace.Load(dir)
	if err != nil {
		t.Fatalf("workspaceを読み込めません: %v", err)
	}
	return result
}

func appendToFile(t *testing.T, path, text string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(content, []byte(text)...), 0o644); err != nil {
		t.Fatal(err)
	}
}

func worldJSON(t *testing.T, w ir.World) string {
	t.Helper()
	raw, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func keys(changes []SubjectChange) []string {
	var out []string
	for _, c := range changes {
		out = append(out, c.Change+" "+c.Key())
	}
	sort.Strings(out)
	return out
}

func TestFileMoveIsASemanticNoOpButALogicalRenameIsASemanticChange(t *testing.T) {
	baseline := loadWorld(t, referenceWorkspaceCopy(t))
	const block = "validation sales.order_line.quantity_positive positive {\n    attribute sales.order_line.quantity\n}\n"

	moved := referenceWorkspaceCopy(t)
	validationFile := filepath.Join(moved, "semantic", "sales", "validation.tei")
	content, err := os.ReadFile(validationFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), block+"\n") {
		t.Fatalf("前提のsourceが変わっています:\n%s", content)
	}
	if err := os.WriteFile(validationFile, []byte(strings.Replace(string(content), block+"\n", "", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moved, "semantic", "sales", "quantity.tei"), []byte("namespace sales\n\n"+block), 0o644); err != nil {
		t.Fatal(err)
	}
	movedResult := loadWorld(t, moved)

	diff, err := Diff(baseline.World, movedResult.World)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff) != 0 {
		t.Fatalf("同じ意味を別fileへ移しただけなのにsemantic差分が出ています: %+v", diff)
	}
	if worldJSON(t, baseline.World) != worldJSON(t, movedResult.World) {
		t.Fatal("file移動でCanonical Worldが変わりました")
	}
	if got := movedResult.Sources["semantic-validation:sales.order_line.quantity_positive"].File; got != "semantic/sales/quantity.tei" {
		t.Fatalf("Source locatorだけが新しいfileへ追随する必要があります: %s", got)
	}
	if got := baseline.Sources["semantic-validation:sales.order_line.quantity_positive"].File; got != "semantic/sales/validation.tei" {
		t.Fatalf("baselineのlocator=%s", got)
	}

	renamed := referenceWorkspaceCopy(t)
	renamedFile := filepath.Join(renamed, "semantic", "sales", "validation.tei")
	renamedContent, err := os.ReadFile(renamedFile)
	if err != nil {
		t.Fatal(err)
	}
	renamedText := strings.ReplaceAll(string(renamedContent), "sales.order_line.quantity_positive", "sales.order_line.quantity_gt_zero")
	if err := os.WriteFile(renamedFile, []byte(renamedText), 0o644); err != nil {
		t.Fatal(err)
	}
	renameDiff, err := Diff(baseline.World, loadWorld(t, renamed).World)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"added validation:sales.order_line.quantity_gt_zero",
		"removed validation:sales.order_line.quantity_positive",
	}
	if got := keys(renameDiff); !reflect.DeepEqual(got, want) {
		t.Fatalf("論理的な改名がsemantic変更として現れていません（暗黙のno-opにしない）: %v", got)
	}
}

func TestDiffReportsAddedRemovedAndChangedSubjects(t *testing.T) {
	before := loadWorld(t, referenceWorkspaceCopy(t)).World
	after := loadWorld(t, referenceWorkspaceCopy(t)).World

	for i := range after.Semantics {
		if after.Semantics[i].ID != "sales" {
			continue
		}
		ns := &after.Semantics[i]
		ns.Capabilities = nil
		for d := range ns.Data {
			if ns.Data[d].ID != "sales.order_line" {
				continue
			}
			for a := range ns.Data[d].Attributes {
				if ns.Data[d].Attributes[a].ID == "sales.order_line.item_code" {
					ns.Data[d].Attributes[a].TypeRef = "std.text"
				}
			}
			ns.Data[d].Attributes = append(ns.Data[d].Attributes, semantic.Attribute{ID: "sales.order_line.note", Symbol: "note", TypeRef: "std.text"})
		}
		ns.Validations = append(ns.Validations, semantic.Validation{
			ID: "sales.order_line.unit_price_positive", Symbol: "unit_price_positive",
			Kind: semantic.ValidationPositive, Attribute: "sales.order_line.unit_price",
		})
	}

	diff, err := Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"added attribute:sales.order_line.note",
		"added validation:sales.order_line.unit_price_positive",
		"changed attribute:sales.order_line.item_code",
		"removed capability:sales.order.submit",
	}
	if got := keys(diff); !reflect.DeepEqual(got, want) {
		t.Fatalf("diff=%v want %v", got, want)
	}
	for _, c := range diff {
		switch c.Change {
		case "added":
			if len(c.After) == 0 || len(c.Before) != 0 {
				t.Fatalf("added subjectはafterだけを持ちます: %+v", c)
			}
		case "removed":
			if len(c.Before) == 0 || len(c.After) != 0 {
				t.Fatalf("removed subjectはbeforeだけを持ちます: %+v", c)
			}
		case "changed":
			if len(c.Before) == 0 || len(c.After) == 0 || string(c.Before) == string(c.After) {
				t.Fatalf("changed subjectはbefore / afterが異なります: %+v", c)
			}
		}
	}
	if same, err := Diff(before, before); err != nil || len(same) != 0 {
		t.Fatalf("同一Worldに差分が出ています: %v %v", same, err)
	}
}

func TestDiffRefusesASubjectDeclaredTwice(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{
		{ID: "a", Data: []semantic.Data{{ID: "a.thing"}}},
		{ID: "b", Data: []semantic.Data{{ID: "a.thing"}}},
	}}
	if _, err := Diff(world, world); err == nil {
		t.Fatal("同じsubjectが複数回宣言されたWorldを差分できてしまいます（どちらが正か決められない）")
	}
}

func TestCollectionDiffSeesChangesOutsideTheSemanticLayer(t *testing.T) {
	world := loadWorld(t, referenceWorkspaceCopy(t)).World
	other := world
	other.Applications = nil
	changed, unchanged, err := CollectionDiff(world, other)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(changed, []string{"applications"}) || !contains(unchanged, "semantics") {
		t.Fatalf("changed=%v unchanged=%v", changed, unchanged)
	}
	if same, _, err := CollectionDiff(world, world); err != nil || len(same) != 0 {
		t.Fatalf("同一Worldに差分が出ています: %v %v", same, err)
	}
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func addValidation(t *testing.T, dir, text string) ir.World {
	t.Helper()
	appendToFile(t, filepath.Join(dir, "semantic", "sales", "validation.tei"), "\n"+text)
	return loadWorld(t, dir).World
}

func TestImpactOfAnAddedValidationTracesExactDependencies(t *testing.T) {
	before := loadWorld(t, referenceWorkspaceCopy(t)).World
	after := addValidation(t, referenceWorkspaceCopy(t),
		"validation sales.order_line.unit_price_positive positive {\n    attribute sales.order_line.unit_price\n}\n")
	diff, err := Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	edges, unresolved := Impact(after, diff)

	want := [][3]string{
		{"attribute:sales.order_line.unit_price", "data:sales.order_line", "semantic.data.attribute"},
		{"data:sales.order_line", "relation:sales.order.lines", "semantic.relation.target"},
		{"relation:sales.order.lines", "application-pattern:order_entry.order_lines", "application.pattern.relation"},
		{"validation:sales.order_line.unit_price_positive", "application:order_entry", "application.uses-semantic-namespace"},
		{"validation:sales.order_line.unit_price_positive", "attribute:sales.order_line.unit_price", "semantic.validation.attribute"},
	}
	var got [][3]string
	for _, e := range edges {
		if e.Reason == "" {
			t.Fatalf("辺に導出理由がありません: %+v", e)
		}
		got = append(got, [3]string{e.From, e.To, e.Via})
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("edges=%v want %v", got, want)
	}
	var codes []string
	for _, u := range unresolved {
		codes = append(codes, u.Code)
	}
	wantCodes := []string{"APPLICATION_VALIDATION_SELECTION_NOT_DECLARED", "DRIVER_SCENARIO_NOT_RUN", "TARGET_PROJECTION_NOT_EVALUATED"}
	if !reflect.DeepEqual(codes, wantCodes) {
		t.Fatalf("unresolved=%v want %v", codes, wantCodes)
	}
}

func TestImpactOfARelationUniqueValidationIncludesItsRelation(t *testing.T) {
	before := loadWorld(t, referenceWorkspaceCopy(t)).World
	after := addValidation(t, referenceWorkspaceCopy(t),
		"validation sales.order.lines.item_code_unique relation-unique {\n    relation sales.order.lines\n    attribute sales.order_line.item_code\n}\n")
	diff, err := Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	edges, _ := Impact(after, diff)
	found := false
	for _, e := range edges {
		if e.From == "validation:sales.order.lines.item_code_unique" && e.To == "relation:sales.order.lines" && e.Via == "semantic.validation.relation" {
			found = true
		}
	}
	if !found {
		t.Fatalf("relation-unique Validationからscope Relationへの辺がありません: %+v", edges)
	}
}

func TestAnApplicationThatDeclaresTheValidationIsNotFlaggedAsUnresolved(t *testing.T) {
	before := loadWorld(t, referenceWorkspaceCopy(t)).World
	after := addValidation(t, referenceWorkspaceCopy(t),
		"validation sales.order_line.unit_price_positive positive {\n    attribute sales.order_line.unit_price\n}\n")
	for i := range after.Applications {
		if after.Applications[i].ID == "order_entry" {
			after.Applications[i].Validations = append(after.Applications[i].Validations, core.CanonicalID("sales.order_line.unit_price_positive"))
		}
	}
	diff, err := Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	_, unresolved := Impact(after, diff)
	for _, u := range unresolved {
		if u.Code == "APPLICATION_VALIDATION_SELECTION_NOT_DECLARED" {
			t.Fatalf("Applicationが宣言済みなのに未解決として残っています: %+v", u)
		}
	}
}

func TestARemovalProducesNoEdgesButNeverClaimsNoImpact(t *testing.T) {
	before := loadWorld(t, referenceWorkspaceCopy(t)).World
	after := loadWorld(t, referenceWorkspaceCopy(t)).World
	for i := range after.Semantics {
		if after.Semantics[i].ID == "sales" {
			after.Semantics[i].Validations = nil
		}
	}
	diff, err := Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	edges, unresolved := Impact(after, diff)
	if len(edges) != 0 {
		t.Fatalf("削除されたValidationから辺を作っています: %+v", edges)
	}
	if len(unresolved) < 2 {
		t.Fatalf("辺が無くても、解析できなかった範囲(unresolved)は残す必要があります: %+v", unresolved)
	}
}

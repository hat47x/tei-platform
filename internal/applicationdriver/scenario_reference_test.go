package applicationdriver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/application"
	"tei/internal/applicationdriver"
	"tei/internal/core"
	"tei/internal/workspace"
)

func TestReferenceOrderEntryPatternScenarioIsAssembledFromCanonicalIdentity(t *testing.T) {
	root := applicationDriverRepoRoot(t)
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	patternIDs := []core.CanonicalID{
		"order_entry.recent_orders",
		"order_entry.order_record",
		"order_entry.customer_reference",
		"order_entry.order_lines",
		"order_entry.submit_order",
	}
	plan, err := applicationdriver.AssemblePatternScenario(loaded.World, "order_entry", patternIDs)
	if err != nil {
		t.Fatalf("order_entry Scenarioのassemblyに失敗しました: %v", err)
	}
	if plan.Application.ID != "order_entry" {
		t.Fatalf("application=%s", plan.Application.ID)
	}

	wantKinds := []application.PatternKind{
		application.PatternBrowse,
		application.PatternRecord,
		application.PatternReference,
		application.PatternHeaderDetail,
		application.PatternCommand,
	}
	if len(plan.Patterns) != len(wantKinds) {
		t.Fatalf("patterns=%#v", plan.Patterns)
	}
	for i, pattern := range plan.Patterns {
		if pattern.ID != patternIDs[i] || pattern.Kind != wantKinds[i] {
			t.Fatalf("pattern[%d]=%#v want id=%s kind=%s", i, pattern, patternIDs[i], wantKinds[i])
		}
	}

	browse := plan.Patterns[0]
	if browse.Data != "sales.order" || browse.Query != "order_entry.recent_orders" || browse.Datasource != "" {
		t.Fatalf("browse PatternのCanonicalな意味が保持されていません: %#v", browse)
	}

	reference := plan.Patterns[2]
	if reference.Relation != "sales.order.customer" || reference.Datasource != "master" {
		t.Fatalf("reference PatternのCanonicalな意味が保持されていません: %#v", reference)
	}

	headerDetail := plan.Patterns[3]
	if headerDetail.Relation != "sales.order.lines" || headerDetail.Data != "" || headerDetail.Datasource != "" {
		t.Fatalf("header-detail PatternがRelation-only契約を保持していません: %#v", headerDetail)
	}

	command := plan.Patterns[4]
	if command.Capability != "sales.order.submit" || command.Data != "" || command.Datasource != "" {
		t.Fatalf("command PatternがCapability-only契約を保持していません: %#v", command)
	}
}

func applicationDriverRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test fileの位置を取得できません")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

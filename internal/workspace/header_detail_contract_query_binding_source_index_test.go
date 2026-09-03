package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/internal/validation"
)

func TestLoadIndexesHeaderDetailContractQueryBindingSource(t *testing.T) {
	root := t.TempDir()
	writeHeaderDetailBindingWorkspaceFile(t, root, "tei.workspace.yaml", `workspace:
  sources:
    - applications
`)
	writeHeaderDetailBindingWorkspaceFile(t, root, "applications/order-entry/application.tei", `application order_entry {
}`)
	writeHeaderDetailBindingWorkspaceFile(t, root, "applications/order-entry/patterns/order-lines.tei", `pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}`)
	writeHeaderDetailBindingWorkspaceFile(t, root, "applications/order-entry/realizations/header-detail/order-lines.tei", `header-detail-contract-query-binding {
    application order_entry
    pattern order_entry.order_lines
    datasource sales
    database sales_core
    contract sales_core.order_line
    query find_by_order
    input sales.order.order_id to order_id
}`)

	result, err := Load(root)
	if err != nil {
		t.Fatalf("Workspaceを読み込めません: %v", err)
	}
	if len(result.World.HeaderDetailContractQueryBindings) != 1 {
		t.Fatalf("通常Load入口からBindingを取得できません: %#v", result.World.HeaderDetailContractQueryBindings)
	}
	binding := result.World.HeaderDetailContractQueryBindings[0]
	if binding.Identity() != "order_entry/order_entry.order_lines" {
		t.Fatalf("Binding identityが変わっています: %s", binding.Identity())
	}

	index, err := BuildSourceIndex(root, result)
	if err != nil {
		t.Fatalf("Source Indexを構築できません: %v", err)
	}
	bindingSubject := sourceindex.Subject{Kind: "header-detail-contract-query-binding", ID: binding.Identity()}
	refs := index.ReferencesFor(bindingSubject)
	if len(refs) < 1 {
		t.Fatalf("Binding sourceへのReferenceがありません: %#v", index.References)
	}
	for _, ref := range refs {
		if ref.ArtifactPath != "applications/order-entry/realizations/header-detail/order-lines.tei" {
			t.Fatalf("Binding Diagnosticを別sourceへ投影しています: %#v", refs)
		}
	}

	diagnosticSubject := subject.Subject{Kind: bindingSubject.Kind, ID: bindingSubject.ID}
	projected := ProjectDiagnostics([]validation.Diagnostic{
		{
			Code:     "TEST_HEADER_DETAIL_BINDING",
			Severity: validation.Error,
			Message:  "test",
			Subject:  &diagnosticSubject,
		},
	}, index)
	if len(projected) != 1 || len(projected[0].Sources) == 0 || projected[0].Sources[0].ArtifactPath != "applications/order-entry/realizations/header-detail/order-lines.tei" {
		t.Fatalf("Validation DiagnosticからBinding authoring sourceへ戻れません: %#v", projected)
	}
}

func writeHeaderDetailBindingWorkspaceFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

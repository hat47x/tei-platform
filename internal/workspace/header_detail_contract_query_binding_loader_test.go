package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadHeaderDetailContractQueryBindingsKeepsSourceIdentityExplicit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "applications", "folder-name-is-not-authority", "realizations", "header-detail", "order-lines.tei")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	src := `header-detail-contract-query-binding {
    application order_entry
    pattern order_entry.order_lines
    datasource sales
    database sales_core
    contract sales_core.order_line
    query find_by_order
    input sales.order.order_id to order_id
}`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	result := Result{Sources: map[string]SourceLocation{}}
	if err := loadHeaderDetailContractQueryBindings(root, &result); err != nil {
		t.Fatalf("Binding sourceを読み込めません: %v", err)
	}
	if len(result.World.HeaderDetailContractQueryBindings) != 1 {
		t.Fatalf("Binding数が期待値と異なります: %#v", result.World.HeaderDetailContractQueryBindings)
	}
	binding := result.World.HeaderDetailContractQueryBindings[0]
	if binding.Application != "order_entry" || binding.Pattern != "order_entry.order_lines" {
		t.Fatalf("ディレクトリ名からApplication / Pattern identityを推測しています: %#v", binding)
	}
	if binding.Target.Database != "sales_core" || binding.Target.Contract != "sales_core.order_line" || binding.Target.Query != "find_by_order" {
		t.Fatalf("Contract Query targetが変わっています: %#v", binding.Target)
	}

	key := "header-detail-contract-query-binding:" + binding.Identity()
	location, ok := result.Sources[key]
	if !ok || location.File != "applications/folder-name-is-not-authority/realizations/header-detail/order-lines.tei" {
		t.Fatalf("Binding authoring sourceを追跡できません: key=%s location=%#v", key, location)
	}
	if len(result.SourceContributions) != 1 || result.SourceContributions[0].Subject.Kind != "header-detail-contract-query-binding" || result.SourceContributions[0].Subject.ID != binding.Identity() {
		t.Fatalf("BindingのSourceContributionが失われています: %#v", result.SourceContributions)
	}
}

func TestLoadHeaderDetailContractQueryBindingsRejectsDuplicateIdentity(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "applications", "order-entry", "realizations", "header-detail")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	first := `header-detail-contract-query-binding {
    application order_entry
    pattern order_entry.order_lines
    datasource sales
    database sales_core
    contract sales_core.order_line
    query find_by_order
    input sales.order.order_id to order_id
}`
	second := `header-detail-contract-query-binding {
    application order_entry
    pattern order_entry.order_lines
    datasource archive
    database archive_sales
    contract archive_sales.order_line
    query find_by_order
    input sales.order.order_id to order_id
}`
	if err := os.WriteFile(filepath.Join(dir, "a.tei"), []byte(first), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.tei"), []byte(second), 0o644); err != nil {
		t.Fatal(err)
	}

	result := Result{Sources: map[string]SourceLocation{}}
	err := loadHeaderDetailContractQueryBindings(root, &result)
	if err == nil || !strings.Contains(err.Error(), "既に") || !strings.Contains(err.Error(), "order_entry/order_entry.order_lines") {
		t.Fatalf("同一Application Patternへの複数realization sourceを拒否していません: %v", err)
	}
}

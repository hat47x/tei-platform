package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadContractQueryExecutorBindingsKeepsSourceIdentityExplicit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "environments", "development", "contract-query-executors", "customer-find-by-id.tei")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	src := `contract-query-executor-binding {
    environment production
    instance master_data.prod
    database master_data
    contract master_data.customer
    query find_by_id
    executor database.query.oracle
}`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	result := Result{Sources: map[string]SourceLocation{}}
	if err := loadContractQueryExecutorBindings(root, &result); err != nil {
		t.Fatalf("Binding sourceを読み込めません: %v", err)
	}
	if len(result.World.ContractQueryExecutorBindings) != 1 {
		t.Fatalf("Binding数が期待値と異なります: %#v", result.World.ContractQueryExecutorBindings)
	}
	binding := result.World.ContractQueryExecutorBindings[0]
	if binding.Environment != "production" || binding.Instance != "master_data.prod" {
		t.Fatalf("ディレクトリ名developmentからEnvironment identityを推測しています: %#v", binding)
	}
	key := "contract-query-executor-binding:" + binding.Identity()
	location, ok := result.Sources[key]
	if !ok || location.File != "environments/development/contract-query-executors/customer-find-by-id.tei" {
		t.Fatalf("Binding authoring sourceを追跡できません: key=%s location=%#v", key, location)
	}
	if len(result.SourceContributions) != 1 || result.SourceContributions[0].Subject.ID != binding.Identity() {
		t.Fatalf("BindingのSourceContributionが失われています: %#v", result.SourceContributions)
	}
}

func TestLoadContractQueryExecutorBindingsRejectsDuplicateIdentity(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "environments", "development", "contract-query-executors")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	first := `contract-query-executor-binding {
    environment development
    instance master_data.dev
    database master_data
    contract master_data.customer
    query find_by_id
    executor database.query.oracle
}`
	second := `contract-query-executor-binding {
    environment development
    instance master_data.dev
    database master_data
    contract master_data.customer
    query find_by_id
    executor database.query.oracle.alternate
}`
	if err := os.WriteFile(filepath.Join(dir, "a.tei"), []byte(first), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.tei"), []byte(second), 0o644); err != nil {
		t.Fatal(err)
	}

	result := Result{Sources: map[string]SourceLocation{}}
	err := loadContractQueryExecutorBindings(root, &result)
	if err == nil || !strings.Contains(err.Error(), "既に") || !strings.Contains(err.Error(), "development/master_data.dev/master_data/master_data.customer/find_by_id") {
		t.Fatalf("同一Binding identityへの複数executor宣言を拒否していません: %v", err)
	}
}

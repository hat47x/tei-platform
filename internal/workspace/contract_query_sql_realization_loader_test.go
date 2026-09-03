package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

func TestLoadContractQuerySQLRealizationSeparatesAuthoringAndNativeSQLSources(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "databases", "master-data", "query-realizations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	declPath := filepath.Join(dir, "customer-find-by-id.tei")
	sqlPath := filepath.Join(dir, "customer-find-by-id.oracle.sql")
	decl := `contract-query-sql-realization {
    database legacy_master
    contract legacy_master.customer
    query find_by_id
    sql oracle file "customer-find-by-id.oracle.sql"
    input id -> customer_id
    result {
        column CUSTOMER_ID -> customer.customer.customer_id
        column NAME -> customer.customer.name
    }
}`
	if err := os.WriteFile(declPath, []byte(decl), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sqlPath, []byte("select CUSTOMER_ID, NAME from LEGACY_CUSTOMER where CUSTOMER_ID = :customer_id\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := Result{Sources: map[string]SourceLocation{}}
	if err := loadContractQuerySQLRealizations(root, &result); err != nil {
		t.Fatalf("SQL realization sourceを読み込めません: %v", err)
	}
	if len(result.World.ContractQuerySQLRealizations) != 1 {
		t.Fatalf("SQL realization数が期待値と異なります: %#v", result.World.ContractQuerySQLRealizations)
	}
	r := result.World.ContractQuerySQLRealizations[0]
	if r.Target.Database != "legacy_master" {
		t.Fatalf("databases/master-dataという配置からDatabase identityを推測しています: %#v", r.Target)
	}
	if !strings.Contains(r.Source, "LEGACY_CUSTOMER") {
		t.Fatalf("外部native SQL sourceを読み込めていません: %q", r.Source)
	}
	identity := r.Identity()
	location, ok := result.Sources["contract-query-sql-realization:"+identity]
	if !ok || location.File != "databases/master-data/query-realizations/customer-find-by-id.tei" {
		t.Fatalf("SQL realization authoring sourceを追跡できません: %#v", location)
	}
	var authoring, implementation bool
	for _, contribution := range result.SourceContributions {
		if contribution.Subject.Kind != "contract-query-sql-realization" || contribution.Subject.ID != identity {
			continue
		}
		if contribution.Role == sourceindex.RoleAuthoring && contribution.File == "databases/master-data/query-realizations/customer-find-by-id.tei" {
			authoring = true
		}
		if contribution.Role == sourceindex.RoleImplementation && contribution.File == "databases/master-data/query-realizations/customer-find-by-id.oracle.sql" {
			implementation = true
		}
	}
	if !authoring || !implementation {
		t.Fatalf("authoring / implementation sourceの役割分離が失われています: %#v", result.SourceContributions)
	}
}

func TestLoadContractQuerySQLRealizationRejectsDuplicateIdentity(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "databases", "master-data", "query-realizations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	first := `contract-query-sql-realization {
    database master_data
    contract master_data.customer
    query find_by_id
    sql oracle { select CUSTOMER_ID from CUSTOMER_MASTER where CUSTOMER_ID = :customer_id }
    input id -> customer_id
    result { column CUSTOMER_ID -> customer.customer.customer_id }
}`
	second := `contract-query-sql-realization {
    database master_data
    contract master_data.customer
    query find_by_id
    sql oracle { select CUSTOMER_ID from OTHER_CUSTOMER where CUSTOMER_ID = :id }
    input id -> id
    result { column CUSTOMER_ID -> customer.customer.customer_id }
}`
	if err := os.WriteFile(filepath.Join(dir, "a.tei"), []byte(first), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.tei"), []byte(second), 0o644); err != nil {
		t.Fatal(err)
	}

	result := Result{Sources: map[string]SourceLocation{}}
	err := loadContractQuerySQLRealizations(root, &result)
	if err == nil || !strings.Contains(err.Error(), "は既に") || !strings.Contains(err.Error(), "master_data/master_data.customer/find_by_id/sql/oracle") {
		t.Fatalf("同じtarget + providerへの複数SQL realizationを曖昧に共存させています: %v", err)
	}
}

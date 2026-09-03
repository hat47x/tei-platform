package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestWorkspaceLoadIncludesContractQuerySQLRealizationAndSourceIndex(t *testing.T) {
	root := t.TempDir()
	mustWriteContractQuerySQLFixture(t, filepath.Join(root, "tei.workspace.yaml"), "workspace: fixture\nsources:\n  - databases\n")
	mustWriteContractQuerySQLFixture(t, filepath.Join(root, "databases", "master-data", "database.tei"), `database master_data {
    kind relational
    ownership external-observed
}
`)
	const realizationPath = "databases/master-data/query-realizations/find-by-id.tei"
	mustWriteContractQuerySQLFixture(t, filepath.Join(root, filepath.FromSlash(realizationPath)), `contract-query-sql-realization {
    database master_data
    contract master_data.customer
    query find_by_id
    sql oracle {
        select observed_customer_id from observed_customer_view where observed_customer_id = :lookup_id
    }
    input id -> lookup_id
    result {
        column observed_customer_id -> customer.customer.customer_id
    }
}
`)

	result, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("通常のWorkspace LoadでContract Query SQL realizationを読み込めません: %v", err)
	}
	if len(result.World.ContractQuerySQLRealizations) != 1 {
		t.Fatalf("Contract Query SQL realizationの読込み件数が期待値と異なります: %#v", result.World.ContractQuerySQLRealizations)
	}
	realizationDef := result.World.ContractQuerySQLRealizations[0]
	if realizationDef.Identity() != "master_data/master_data.customer/find_by_id/sql/oracle" {
		t.Fatalf("SQL realization identityが変わっています: %s", realizationDef.Identity())
	}
	if result.Sources["contract-query-sql-realization:"+realizationDef.Identity()].File != realizationPath {
		t.Fatalf("SQL realizationのauthoring sourceを追跡できません: %#v", result.Sources)
	}

	index, err := workspace.BuildSourceIndex(root, result)
	if err != nil {
		t.Fatalf("SQL realizationを含むSource Indexを構築できません: %v", err)
	}
	refs := index.ReferencesFor(sourceindex.Subject{Kind: "contract-query-sql-realization", ID: realizationDef.Identity()})
	var authoringRef *sourceindex.Reference
	for i := range refs {
		if refs[i].Role == sourceindex.RoleAuthoring {
			authoringRef = &refs[i]
			break
		}
	}
	if authoringRef == nil {
		t.Fatalf("SQL realizationのauthoring provenanceがSource Indexにありません: %#v", refs)
	}
	if authoringRef.ArtifactPath != realizationPath {
		t.Fatalf("SQL realizationのauthoring source pathが変わっています: %#v", *authoringRef)
	}
}

func TestWorkspaceLoadDoesNotCreateContractQuerySQLRealizationWithoutAuthoringSource(t *testing.T) {
	root := t.TempDir()
	mustWriteContractQuerySQLFixture(t, filepath.Join(root, "tei.workspace.yaml"), "workspace: fixture\nsources:\n  - databases\n")
	mustWriteContractQuerySQLFixture(t, filepath.Join(root, "databases", "master-data", "database.tei"), `database master_data {
    kind relational
    ownership external-observed
}
`)

	result, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("SQL realization sourceがないexternal-observed Databaseを読み込めません: %v", err)
	}
	if len(result.World.ContractQuerySQLRealizations) != 0 {
		t.Fatalf("物理sourceがないのにSQL realizationを自動生成しています: %#v", result.World.ContractQuerySQLRealizations)
	}
}

func mustWriteContractQuerySQLFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("fixture directoryを作成できません: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("fixture file %sを書き込めません: %v", path, err)
	}
}

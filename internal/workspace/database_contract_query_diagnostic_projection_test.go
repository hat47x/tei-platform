package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestDatabaseContractQuerySignatureDiagnosticProjectsToContractSource(t *testing.T) {
	root := t.TempDir()
	writeContractQueryDiagnosticFixture(t, filepath.Join(root, "tei.workspace.yaml"), "workspace: fixture\nsources:\n  - databases\n")
	writeContractQueryDiagnosticFixture(t, filepath.Join(root, "databases", "master-data", "database.tei"), `database master_data {
    kind relational
    ownership external-observed
    uses semantic customer
}
`)
	const contractPath = "databases/master-data/contracts/customer.tei"
	const contractSource = `contract master_data.customer v2.4 {
    read customer.customer
    query find_by_id {
        input id type customer.missing_id
        result customer.customer zero-or-one
    }
}
`
	writeContractQueryDiagnosticFixture(t, filepath.Join(root, filepath.FromSlash(contractPath)), contractSource)

	result, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("Contract Queryを含むWorkspaceを読み込めません: %v", err)
	}
	index, err := workspace.BuildSourceIndex(root, result)
	if err != nil {
		t.Fatalf("Contract Queryを含むSource Indexを構築できません: %v", err)
	}
	projected := workspace.ProjectDiagnostics(validation.Validate(result.World), index)

	for _, diagnostic := range projected {
		if diagnostic.Diagnostic.Code != "UNKNOWN_DATABASE_CONTRACT_QUERY_INPUT_TYPE" {
			continue
		}
		if diagnostic.Diagnostic.Subject == nil || diagnostic.Diagnostic.Subject.Kind != "database-contract-query" || diagnostic.Diagnostic.Subject.ID != "master_data.customer/find_by_id" {
			t.Fatalf("Database Contract Query Diagnosticのsubjectが変わっています: %#v", diagnostic.Diagnostic)
		}
		if len(diagnostic.Sources) == 0 {
			t.Fatalf("Database Contract Query Diagnosticからauthoring sourceを辿れません: %#v", diagnostic)
		}
		for _, source := range diagnostic.Sources {
			if source.ArtifactPath != contractPath || source.Role != sourceindex.RoleDefinition {
				continue
			}
			content := []byte(contractSource)
			if source.Range.Start.Offset < 0 || source.Range.End.Offset > len(content) || source.Range.Start.Offset >= source.Range.End.Offset {
				t.Fatalf("Contract Queryのsource rangeが不正です: %+v", source.Range)
			}
			if got := string(content[source.Range.Start.Offset:source.Range.End.Offset]); got != "find_by_id" {
				t.Fatalf("Contract Query Diagnosticのanchor=%q want=%q range=%+v", got, "find_by_id", source.Range)
			}
			return
		}
		t.Fatalf("Database Contract Query DiagnosticがContract authoringへ投影されていません: %#v", diagnostic.Sources)
	}

	t.Fatalf("未知Semantic型を持つDatabase Contract Query signatureのDiagnosticがありません: %#v", projected)
}

func writeContractQueryDiagnosticFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("fixture directoryを作成できません: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("fixture file %sを書き込めません: %v", path, err)
	}
}

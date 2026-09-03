package workspace_test

import (
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestReferenceWorkspaceLoadsContractQueryExecutorBindingFromAuthoringSource(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	if len(result.World.ContractQueryExecutorBindings) != 1 {
		t.Fatalf("Contract Query executor Binding数が期待値と異なります: %#v", result.World.ContractQueryExecutorBindings)
	}
	binding := result.World.ContractQueryExecutorBindings[0]
	if binding.Identity() != "development/master_data.dev/master_data/master_data.customer/find_by_id" {
		t.Fatalf("Binding identityが期待値と異なります: %s", binding.Identity())
	}
	if binding.Executor != "database.query.oracle" {
		t.Fatalf("第一Referenceが明示したexecutor Capabilityを保持できていません: %#v", binding)
	}

	const sourcePath = "environments/development/contract-query-executors/customer-find-by-id.tei"
	sourceKey := "contract-query-executor-binding:" + binding.Identity()
	location, ok := result.Sources[sourceKey]
	if !ok || location.File != sourcePath {
		t.Fatalf("Binding authoring sourceを追跡できません: key=%s location=%#v", sourceKey, location)
	}

	idx, err := workspace.BuildSourceIndex(ws, result)
	if err != nil {
		t.Fatalf("Source Indexの構築に失敗しました: %v", err)
	}
	refs := idx.ReferencesFor(sourceindex.Subject{Kind: "contract-query-executor-binding", ID: binding.Identity()})
	if len(refs) != 1 {
		t.Fatalf("Contract Query executor Bindingのsource references=%#v", refs)
	}
	if refs[0].ArtifactPath != sourcePath || refs[0].Role != sourceindex.RoleAuthoring {
		t.Fatalf("Binding authoring sourceの来歴が期待値と異なります: %#v", refs[0])
	}
}

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

	if len(result.World.ContractQueryExecutorBindings) != 2 {
		t.Fatalf("Contract Query executor Binding数が期待値と異なります: %#v", result.World.ContractQueryExecutorBindings)
	}
	const oracleIdentity = "development/master_data.dev/master_data/master_data.customer/find_by_id"
	const postgresIdentity = "development/sales_core.dev/sales_core/sales_core.order_line/find_by_order"
	byIdentity := map[string]int{}
	for i, candidate := range result.World.ContractQueryExecutorBindings {
		byIdentity[candidate.Identity()] = i
	}
	oracleIndex, ok := byIdentity[oracleIdentity]
	if !ok {
		t.Fatalf("Oracle Bindingがありません: %#v", result.World.ContractQueryExecutorBindings)
	}
	binding := result.World.ContractQueryExecutorBindings[oracleIndex]
	postgresIndex, ok := byIdentity[postgresIdentity]
	if !ok {
		t.Fatalf("PostgreSQL Bindingがありません: %#v", result.World.ContractQueryExecutorBindings)
	}
	postgresBinding := result.World.ContractQueryExecutorBindings[postgresIndex]
	if postgresBinding.Executor != "database.query.postgres" {
		t.Fatalf("PostgreSQL Bindingが明示したexecutor Capabilityを保持できていません: %#v", postgresBinding)
	}
	const postgresSourcePath = "environments/development/contract-query-executors/order-line-find-by-order.tei"
	postgresLocation, ok := result.Sources["contract-query-executor-binding:"+postgresBinding.Identity()]
	if !ok || postgresLocation.File != postgresSourcePath {
		t.Fatalf("PostgreSQL Binding authoring sourceを追跡できません: location=%#v", postgresLocation)
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
	var authoring []sourceindex.Reference
	for _, ref := range refs {
		if ref.ArtifactPath != sourcePath {
			t.Fatalf("Binding sourceの来歴が別のartifactを指しています: %#v", ref)
		}
		if ref.Role == sourceindex.RoleAuthoring {
			authoring = append(authoring, ref)
		}
	}
	if len(authoring) != 1 {
		t.Fatalf("Contract Query executor Bindingのauthoring source references=%#v (all=%#v)", authoring, refs)
	}
}

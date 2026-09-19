package workspace

import (
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/realization"
	"tei/internal/sourceindex"
	"tei/internal/validation"
)

func TestReferenceWorkspaceDefinesHeaderDetailContractQueryWithoutParentIdentityLeakage(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source pathを取得できません")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	root := filepath.Join(repositoryRoot, "reference", "workspace")
	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("Reference Workspaceを読み込めません: %v", err)
	}

	var binding *realization.HeaderDetailContractQueryBinding
	for i := range loaded.World.HeaderDetailContractQueryBindings {
		candidate := &loaded.World.HeaderDetailContractQueryBindings[i]
		if candidate.Identity() == "order_entry/order_entry.order_lines" {
			binding = candidate
			break
		}
	}
	if binding == nil {
		t.Fatalf("order_linesのheader-detail Contract Query bindingがありません: %#v", loaded.World.HeaderDetailContractQueryBindings)
	}
	if binding.Datasource != "sales" || binding.Target.Database != "sales_core" || binding.Target.Contract != "sales_core.order_line" || binding.Target.Query != "find_by_order" {
		t.Fatalf("ReferenceのContract Query targetが変わっています: %#v", binding)
	}
	if len(binding.Inputs) != 1 || binding.Inputs[0].SourceAttribute != "sales.order.order_id" || binding.Inputs[0].Input != "order_id" {
		t.Fatalf("Order source identityからQuery inputへの明示bindingが変わっています: %#v", binding.Inputs)
	}

	if diagnostics := validation.ValidateHeaderDetailContractQueryBindings(loaded.World); len(diagnostics) != 0 {
		t.Fatalf("Reference header-detail Contract Query bindingがCanonical Validationを満たしません: %#v", diagnostics)
	}

	var relationFound bool
	var orderLineFound bool
	for _, namespace := range loaded.World.Semantics {
		for _, data := range namespace.Data {
			if data.ID == "sales.order" {
				for _, relation := range data.Relations {
					if relation.ID == "sales.order.lines" {
						relationFound = true
						if len(relation.Keys) != 0 {
							t.Fatalf("header-detail traversalのためにsales.order.linesへRelationKeyを追加しています: %#v", relation.Keys)
						}
					}
				}
			}
			if data.ID == "sales.order_line" {
				orderLineFound = true
				for _, attribute := range data.Attributes {
					if attribute.ID == "sales.order_line.order_id" {
						t.Fatalf("Driver traversalのためだけにOrderLineへ親Order identityを追加しています: %#v", attribute)
					}
				}
			}
		}
	}
	if !relationFound || !orderLineFound {
		t.Fatalf("Reference SemanticのOrder / OrderLine関係を確認できません: relation=%v orderLine=%v", relationFound, orderLineFound)
	}

	for _, mapping := range loaded.World.Persistence {
		if mapping.SemanticData == "sales.order_line" {
			requireExplicitDatabaseAuthoring(t, loaded, "persistence:"+string(mapping.ID), "Contract Queryとは別のOrderLine Persistence Mapping")
		}
	}

	target := realization.ContractQueryTarget{
		Database: "sales_core",
		Contract: "sales_core.order_line",
		Query:    "find_by_order",
	}
	for _, sqlRealization := range loaded.World.ContractQuerySQLRealizations {
		if sqlRealization.Target == target {
			requireExplicitDatabaseAuthoring(t, loaded, "contract-query-sql-realization:"+target.Identity()+"/sql/"+string(sqlRealization.Provider), "OrderLine SQL realization")
		}
	}

	index, err := BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("Reference Source Indexを構築できません: %v", err)
	}
	refs := index.ReferencesFor(sourceindex.Subject{
		Kind: "header-detail-contract-query-binding",
		ID:   binding.Identity(),
	})
	if len(refs) == 0 {
		t.Fatal("Reference header-detail Contract Query bindingのauthoring sourceを追跡できません")
	}
	for _, ref := range refs {
		if ref.ArtifactPath != "applications/order-entry/realizations/header-detail/order-lines.tei" {
			t.Fatalf("BindingのSource Indexが別artifactを指しています: %#v", refs)
		}
	}
}

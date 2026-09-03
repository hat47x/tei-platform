package driver_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/workspace"
)

func TestPrepareHeaderDetailContractQueryUsesExplicitSourceAttributeBinding(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	mapped := orderHeaderMappedResult()

	prepared, err := driver.NewResolver(world).PrepareHeaderDetailContractQuery(
		"order_entry",
		"order_entry.order_lines",
		mapped,
		0,
	)
	if err != nil {
		t.Fatalf("source attributeの明示bindingからheader-detail parameterを準備できません: %v", err)
	}
	if prepared.Resolution.Binding.Identity() != "order_entry/order_entry.order_lines" {
		t.Fatalf("Header-detail binding identityが変わっています: %#v", prepared.Resolution.Binding)
	}
	if prepared.Resolution.Database.ID != "sales_core" || prepared.Resolution.Contract.ID != "sales_core.order_line" || prepared.Resolution.Query.Name != "find_by_order" {
		t.Fatalf("Contract Query resolutionが変わっています: %#v", prepared.Resolution)
	}
	if len(prepared.Resolution.Relation.Relation.Keys) != 0 {
		t.Fatalf("header-detail parameter準備にRelationKeyを暗黙追加しています: %#v", prepared.Resolution.Relation.Relation.Keys)
	}
	if len(prepared.Parameters) != 1 || prepared.Parameters["order_id"] != int64(1001) {
		t.Fatalf("sales.order.order_idのruntime値をorder_idへ渡せていません: %#v", prepared.Parameters)
	}
}

func TestPrepareHeaderDetailContractQueryUsesSemanticAttributeInsteadOfColumnOrder(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	mapped := queryresult.MappedResult{
		Columns: []queryresult.Column{
			{Raw: "status", Attribute: "sales.order.status"},
			{Raw: "order_id", Attribute: "sales.order.order_id"},
		},
		Rows: [][]any{{"entered", int64(2002)}},
	}

	prepared, err := driver.NewResolver(world).PrepareHeaderDetailContractQuery(
		"order_entry",
		"order_entry.order_lines",
		mapped,
		0,
	)
	if err != nil {
		t.Fatalf("mapped column順序を変更するとheader-detail parameterを準備できません: %v", err)
	}
	if prepared.Parameters["order_id"] != int64(2002) {
		t.Fatalf("列順からruntime値を選んでいます: %#v", prepared.Parameters)
	}
}

func TestPrepareHeaderDetailContractQueryDoesNotRequireRelationKey(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	for nsIndex := range world.Semantics {
		for dataIndex := range world.Semantics[nsIndex].Data {
			if world.Semantics[nsIndex].Data[dataIndex].ID != "sales.order" {
				continue
			}
			for relationIndex := range world.Semantics[nsIndex].Data[dataIndex].Relations {
				if world.Semantics[nsIndex].Data[dataIndex].Relations[relationIndex].ID == "sales.order.lines" {
					world.Semantics[nsIndex].Data[dataIndex].Relations[relationIndex].Keys = nil
				}
			}
		}
	}

	if _, err := driver.NewResolver(world).PrepareHeaderDetailContractQuery(
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
	); err != nil {
		t.Fatalf("keylessなmany Relationをruntime準備で拒否しています: %v", err)
	}
}

func TestPrepareHeaderDetailContractQueryRejectsMissingSourceAttributeValue(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	mapped := queryresult.MappedResult{
		Columns: []queryresult.Column{{Raw: "status", Attribute: "sales.order.status"}},
		Rows:    [][]any{{"entered"}},
	}

	_, err := driver.NewResolver(world).PrepareHeaderDetailContractQuery(
		"order_entry",
		"order_entry.order_lines",
		mapped,
		0,
	)
	if err == nil || !strings.Contains(err.Error(), "source attribute sales.order.order_idがmapped resultにありません") {
		t.Fatalf("Bindingに必要なsource attribute値の欠如を明示的に拒否していません: %v", err)
	}
}

func TestPrepareHeaderDetailContractQueryRejectsSourceAttributeOutsideRelationSource(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	world.HeaderDetailContractQueryBindings[0].Inputs[0].SourceAttribute = "sales.order_line.order_line_id"

	_, err := driver.NewResolver(world).PrepareHeaderDetailContractQuery(
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
	)
	if err == nil || !strings.Contains(err.Error(), "source Data sales.orderに属していません") {
		t.Fatalf("Relation source外のSemantic attributeをruntime binding元として許可しています: %v", err)
	}
}

func TestPrepareHeaderDetailContractQueryRejectsSemanticTypeMismatch(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	for dbIndex := range world.Databases {
		if world.Databases[dbIndex].ID != "sales_core" {
			continue
		}
		for contractIndex := range world.Databases[dbIndex].Contracts {
			if world.Databases[dbIndex].Contracts[contractIndex].ID != "sales_core.order_line" {
				continue
			}
			world.Databases[dbIndex].Contracts[contractIndex].Queries[0].Inputs[0].Type = "sales.order_line_id"
		}
	}

	_, err := driver.NewResolver(world).PrepareHeaderDetailContractQuery(
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
	)
	if err == nil || !strings.Contains(err.Error(), "へ渡せません") {
		t.Fatalf("source attributeとQuery inputのSemantic型不一致を許可しています: %v", err)
	}
}

func TestPrepareHeaderDetailContractQueryRejectsZeroInputTraversal(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	world.HeaderDetailContractQueryBindings[0].Inputs = nil
	for dbIndex := range world.Databases {
		if world.Databases[dbIndex].ID != "sales_core" {
			continue
		}
		for contractIndex := range world.Databases[dbIndex].Contracts {
			if world.Databases[dbIndex].Contracts[contractIndex].ID == "sales_core.order_line" {
				world.Databases[dbIndex].Contracts[contractIndex].Queries[0].Inputs = nil
			}
		}
	}

	_, err := driver.NewResolver(world).PrepareHeaderDetailContractQuery(
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
	)
	if err == nil || !strings.Contains(err.Error(), "source attributeからQuery inputへの対応がありません") {
		t.Fatalf("親Orderと無関係な0-input collection Queryをheader-detail traversalとして許可しています: %v", err)
	}
}

func TestResolveHeaderDetailContractQueryRejectsDuplicatePatternBinding(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	world.HeaderDetailContractQueryBindings = append(
		world.HeaderDetailContractQueryBindings,
		realization.HeaderDetailContractQueryBinding{
			Application: "order_entry",
			Pattern:     "order_entry.order_lines",
			Datasource:  "sales",
			Target: realization.ContractQueryTarget{
				Database: "sales_core",
				Contract: "sales_core.order_line",
				Query:    "find_by_order",
			},
			Inputs: []realization.HeaderDetailContractQueryInputBinding{{SourceAttribute: "sales.order.order_id", Input: "order_id"}},
		},
	)

	_, err := driver.NewResolver(world).ResolveHeaderDetailContractQuery("order_entry", "order_entry.order_lines")
	if err == nil || !strings.Contains(err.Error(), "bindingが複数あります") {
		t.Fatalf("同じApplication Patternへの複数runtime realizationを暗黙選択しています: %v", err)
	}
}

func TestPrepareHeaderDetailContractQueryRejectsResultContractMismatch(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	for dbIndex := range world.Databases {
		if world.Databases[dbIndex].ID != "sales_core" {
			continue
		}
		for contractIndex := range world.Databases[dbIndex].Contracts {
			if world.Databases[dbIndex].Contracts[contractIndex].ID == "sales_core.order_line" {
				world.Databases[dbIndex].Contracts[contractIndex].Queries[0].ResultCardinality = database.QueryResultZeroOrOne
			}
		}
	}

	_, err := driver.NewResolver(world).PrepareHeaderDetailContractQuery(
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
	)
	if err == nil || !strings.Contains(err.Error(), "manyを返す必要があります") {
		t.Fatalf("header-detail collection Queryのcardinality不一致を許可しています: %v", err)
	}
}

func loadHeaderDetailReferenceWorld(t *testing.T) ir.World {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source pathを取得できません")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	loaded, err := workspace.Load(filepath.Join(repositoryRoot, "reference", "workspace"))
	if err != nil {
		t.Fatalf("Reference Workspaceを読み込めません: %v", err)
	}
	return loaded.World
}

func orderHeaderMappedResult() queryresult.MappedResult {
	return queryresult.MappedResult{
		Columns: []queryresult.Column{
			{Raw: "order_id", Attribute: "sales.order.order_id"},
			{Raw: "status", Attribute: "sales.order.status"},
		},
		Rows: [][]any{{int64(1001), "entered"}},
	}
}

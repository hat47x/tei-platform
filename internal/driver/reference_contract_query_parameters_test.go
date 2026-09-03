package driver_test

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestPrepareReferenceContractQueryUsesExplicitRelationAndInputBinding(t *testing.T) {
	world := referenceParameterWorld()
	mapped := queryresult.MappedResult{
		Columns: []queryresult.Column{
			{Raw: "order_id", Attribute: "sales.order.order_id"},
			{Raw: "customer_id", Attribute: "sales.order.customer_id"},
		},
		Rows: [][]any{{int64(1001), int64(42)}},
	}

	prepared, err := driver.NewResolver(world).PrepareReferenceContractQuery(
		"order_entry",
		"order_entry.customer_reference",
		mapped,
		0,
	)
	if err != nil {
		t.Fatalf("明示したRelation keyとinput bindingからparameterを組み立てられません: %v", err)
	}
	if prepared.Resolution.Contract.ID != "master_data.customer" || prepared.Resolution.Query.Name != "find_by_id" {
		t.Fatalf("Contract Query resolutionが変わっています: %#v", prepared.Resolution)
	}
	if len(prepared.Parameters) != 1 || prepared.Parameters["id"] != int64(42) {
		t.Fatalf("customer_idのruntime値をidへそのまま渡せていません: %#v", prepared.Parameters)
	}
}

func TestPrepareReferenceContractQueryUsesSemanticAttributeInsteadOfColumnOrder(t *testing.T) {
	world := referenceParameterWorld()
	mapped := queryresult.MappedResult{
		Columns: []queryresult.Column{
			{Raw: "customer_id", Attribute: "sales.order.customer_id"},
			{Raw: "order_id", Attribute: "sales.order.order_id"},
		},
		Rows: [][]any{{int64(84), int64(2002)}},
	}

	prepared, err := driver.NewResolver(world).PrepareReferenceContractQuery(
		"order_entry",
		"order_entry.customer_reference",
		mapped,
		0,
	)
	if err != nil {
		t.Fatalf("mapped column順序を変更するとparameterを組み立てられません: %v", err)
	}
	if prepared.Parameters["id"] != int64(84) {
		t.Fatalf("列順から値を選んでいます: %#v", prepared.Parameters)
	}
}

func TestPrepareReferenceContractQuerySupportsCompositeExplicitKeys(t *testing.T) {
	world := referenceParameterWorld()
	world.Semantics[0].Data[0].Attributes = append(world.Semantics[0].Data[0].Attributes,
		semantic.Attribute{ID: "sales.order.tenant_id", Symbol: "tenantId", TypeRef: "customer.tenant_id"},
	)
	world.Semantics[0].Data[0].Relations[0].Keys = append(world.Semantics[0].Data[0].Relations[0].Keys,
		semantic.RelationKey{SourceAttribute: "sales.order.tenant_id", TargetAttribute: "customer.customer.tenant_id"},
	)
	world.Semantics[1].Data[0].Attributes = append(world.Semantics[1].Data[0].Attributes,
		semantic.Attribute{ID: "customer.customer.tenant_id", Symbol: "tenantId", TypeRef: "customer.tenant_id"},
	)
	world.Applications[0].Patterns[0].ContractQuery.Inputs = append(world.Applications[0].Patterns[0].ContractQuery.Inputs,
		application.ContractQueryInputBinding{TargetAttribute: "customer.customer.tenant_id", Input: "tenant"},
	)
	world.Databases[0].Contracts[0].Queries[0].Inputs = append(world.Databases[0].Contracts[0].Queries[0].Inputs,
		database.QueryInput{Name: "tenant", Type: "customer.tenant_id"},
	)

	mapped := queryresult.MappedResult{
		Columns: []queryresult.Column{
			{Raw: "tenant_id", Attribute: "sales.order.tenant_id"},
			{Raw: "customer_id", Attribute: "sales.order.customer_id"},
		},
		Rows: [][]any{{"north", int64(42)}},
	}

	prepared, err := driver.NewResolver(world).PrepareReferenceContractQuery(
		"order_entry",
		"order_entry.customer_reference",
		mapped,
		0,
	)
	if err != nil {
		t.Fatalf("複合Relation keyを明示bindingからparameterへ変換できません: %v", err)
	}
	if len(prepared.Parameters) != 2 || prepared.Parameters["id"] != int64(42) || prepared.Parameters["tenant"] != "north" {
		t.Fatalf("複合keyのparameterが期待値と異なります: %#v", prepared.Parameters)
	}
}

func TestPrepareReferenceContractQueryRejectsMissingSourceKeyValue(t *testing.T) {
	world := referenceParameterWorld()
	mapped := queryresult.MappedResult{
		Columns: []queryresult.Column{{Raw: "order_id", Attribute: "sales.order.order_id"}},
		Rows:    [][]any{{int64(1001)}},
	}

	_, err := driver.NewResolver(world).PrepareReferenceContractQuery(
		"order_entry",
		"order_entry.customer_reference",
		mapped,
		0,
	)
	if err == nil || !strings.Contains(err.Error(), "source key sales.order.customer_idがmapped resultにありません") {
		t.Fatalf("必要なsource key値の欠如を明示的に拒否していません: %v", err)
	}
}

func TestPrepareReferenceContractQueryRejectsBindingOutsideRelationTargetKeys(t *testing.T) {
	world := referenceParameterWorld()
	world.Semantics[1].Data[0].Attributes = append(world.Semantics[1].Data[0].Attributes,
		semantic.Attribute{ID: "customer.customer.name", Symbol: "name", TypeRef: "std.text"},
	)
	world.Applications[0].Patterns[0].ContractQuery.Inputs[0].TargetAttribute = "customer.customer.name"

	_, err := driver.NewResolver(world).PrepareReferenceContractQuery(
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
	)
	if err == nil || !strings.Contains(err.Error(), "target keyではありません") {
		t.Fatalf("Relation target key以外からQuery inputへ値を渡すbindingを許可しています: %v", err)
	}
}

func TestPrepareReferenceContractQueryRejectsSemanticTypeMismatch(t *testing.T) {
	world := referenceParameterWorld()
	world.Databases[0].Contracts[0].Queries[0].Inputs[0].Type = "customer.other_id"

	_, err := driver.NewResolver(world).PrepareReferenceContractQuery(
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
	)
	if err == nil || !strings.Contains(err.Error(), "へ渡せません") {
		t.Fatalf("Relation target attributeとQuery inputのSemantic型不一致を許可しています: %v", err)
	}
}

func TestPrepareReferenceContractQueryRejectsAmbiguousRelationTargetKey(t *testing.T) {
	world := referenceParameterWorld()
	world.Semantics[0].Data[0].Attributes = append(world.Semantics[0].Data[0].Attributes,
		semantic.Attribute{ID: "sales.order.alternate_customer_id", Symbol: "alternateCustomerId", TypeRef: "customer.customer_id"},
	)
	world.Semantics[0].Data[0].Relations[0].Keys = append(world.Semantics[0].Data[0].Relations[0].Keys,
		semantic.RelationKey{SourceAttribute: "sales.order.alternate_customer_id", TargetAttribute: "customer.customer.customer_id"},
	)

	_, err := driver.NewResolver(world).PrepareReferenceContractQuery(
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
	)
	if err == nil || !strings.Contains(err.Error(), "一意に辿れません") {
		t.Fatalf("同じtarget keyに複数source keyがある状態をruntimeで補完しています: %v", err)
	}
}

func TestPrepareReferenceContractQueryRejectsResultContractMismatch(t *testing.T) {
	world := referenceParameterWorld()
	world.Databases[0].Contracts[0].Queries[0].Result = "customer.other"

	_, err := driver.NewResolver(world).PrepareReferenceContractQuery(
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
	)
	if err == nil || !strings.Contains(err.Error(), "customer.customerをtarget") {
		t.Fatalf("Relation targetと異なるQuery result Dataで呼び出し準備を進めています: %v", err)
	}
}

func TestPrepareReferenceContractQueryRejectsInvalidMappedRowWidth(t *testing.T) {
	world := referenceParameterWorld()
	mapped := queryresult.MappedResult{
		Columns: []queryresult.Column{{Raw: "customer_id", Attribute: "sales.order.customer_id"}},
		Rows:    [][]any{{int64(42), "unexpected"}},
	}

	_, err := driver.NewResolver(world).PrepareReferenceContractQuery(
		"order_entry",
		"order_entry.customer_reference",
		mapped,
		0,
	)
	if err == nil || !strings.Contains(err.Error(), "列数が一致しません") {
		t.Fatalf("mapped row幅の不一致を無視しています: %v", err)
	}
}

func TestPrepareReferenceContractQueryRejectsUnboundQueryInput(t *testing.T) {
	world := referenceParameterWorld()
	world.Databases[0].Contracts[0].Queries[0].Inputs = append(
		world.Databases[0].Contracts[0].Queries[0].Inputs,
		database.QueryInput{Name: "tenant", Type: "customer.customer_id"},
	)

	_, err := driver.NewResolver(world).PrepareReferenceContractQuery(
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
	)
	if err == nil || !strings.Contains(err.Error(), "input tenantへ値を渡すbindingがありません") {
		t.Fatalf("値の供給元がないQuery inputを許可しています: %v", err)
	}
}

func TestPrepareReferenceContractQueryRejectsOutOfRangeRow(t *testing.T) {
	world := referenceParameterWorld()

	_, err := driver.NewResolver(world).PrepareReferenceContractQuery(
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		1,
	)
	if err == nil || !strings.Contains(err.Error(), "mapped row 1がありません") {
		t.Fatalf("存在しないrow indexを許可しています: %v", err)
	}
}

func referenceParameterWorld() ir.World {
	world := referenceContractQueryWorld()

	world.Semantics[0].Data[0].Attributes = []semantic.Attribute{
		{ID: "sales.order.order_id", Symbol: "orderId", TypeRef: "sales.order_id"},
		{ID: "sales.order.customer_id", Symbol: "customerId", TypeRef: "customer.customer_id"},
	}
	world.Semantics[0].Data[0].Relations[0].Keys = []semantic.RelationKey{
		{SourceAttribute: "sales.order.customer_id", TargetAttribute: "customer.customer.customer_id"},
	}
	world.Semantics[1].Data[0].Attributes = []semantic.Attribute{
		{ID: "customer.customer.customer_id", Symbol: "customerId", TypeRef: "customer.customer_id"},
	}
	world.Databases[0].Contracts[0].Queries[0] = database.QueryContract{
		Name: "find_by_id",
		Inputs: []database.QueryInput{
			{Name: "id", Type: "customer.customer_id"},
		},
		Result:            "customer.customer",
		ResultCardinality: database.QueryResultZeroOrOne,
	}

	return world
}

func customerMappedResult() queryresult.MappedResult {
	return queryresult.MappedResult{
		Columns: []queryresult.Column{
			{Raw: "order_id", Attribute: "sales.order.order_id"},
			{Raw: "customer_id", Attribute: "sales.order.customer_id"},
		},
		Rows: [][]any{{int64(1001), int64(42)}},
	}
}

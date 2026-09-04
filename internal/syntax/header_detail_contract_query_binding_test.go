package syntax

import "testing"

func TestParseHeaderDetailContractQueryBinding(t *testing.T) {
	src := `header-detail-contract-query-binding {
    application order_entry
    pattern order_entry.order_lines
    datasource sales
    database sales_core
    contract sales_core.order_line
    query find_by_order
    input sales.order.order_id to order_id
}`

	binding, err := NewParser(src).ParseHeaderDetailContractQueryBinding()
	if err != nil {
		t.Fatalf("header-detail Contract Query bindingを解析できません: %v", err)
	}
	if binding.Application != "order_entry" || binding.Pattern != "order_entry.order_lines" {
		t.Fatalf("Application / Pattern identityが変わっています: %#v", binding)
	}
	if binding.Datasource != "sales" {
		t.Fatalf("Datasourceが変わっています: %#v", binding)
	}
	if binding.Target.Database != "sales_core" || binding.Target.Contract != "sales_core.order_line" || binding.Target.Query != "find_by_order" {
		t.Fatalf("Contract Query targetが変わっています: %#v", binding.Target)
	}
	if len(binding.Inputs) != 1 || binding.Inputs[0].SourceAttribute != "sales.order.order_id" || binding.Inputs[0].Input != "order_id" {
		t.Fatalf("source attributeからQuery inputへのbindingが変わっています: %#v", binding.Inputs)
	}
}

func TestParseHeaderDetailContractQueryBindingKeepsCompositeInputOrder(t *testing.T) {
	src := `header-detail-contract-query-binding {
    application order_entry
    pattern order_entry.order_lines
    datasource sales
    database sales_core
    contract sales_core.order_line
    query find_by_order
    input sales.order.tenant_id to tenant_id
    input sales.order.order_id to order_id
}`

	binding, err := NewParser(src).ParseHeaderDetailContractQueryBinding()
	if err != nil {
		t.Fatalf("複数input bindingを解析できません: %v", err)
	}
	if len(binding.Inputs) != 2 ||
		binding.Inputs[0].SourceAttribute != "sales.order.tenant_id" || binding.Inputs[0].Input != "tenant_id" ||
		binding.Inputs[1].SourceAttribute != "sales.order.order_id" || binding.Inputs[1].Input != "order_id" {
		t.Fatalf("input bindingのauthoring順序が失われています: %#v", binding.Inputs)
	}
}

func TestParseHeaderDetailContractQueryBindingAllowsValidationToReportMissingInputs(t *testing.T) {
	src := `header-detail-contract-query-binding {
    application order_entry
    pattern order_entry.order_lines
    datasource sales
    database sales_core
    contract sales_core.order_line
    query find_by_order
}`

	binding, err := NewParser(src).ParseHeaderDetailContractQueryBinding()
	if err != nil {
		t.Fatalf("input欠落は横断Validationで診断できるようparserで読み込む必要があります: %v", err)
	}
	if len(binding.Inputs) != 0 {
		t.Fatalf("存在しないinput bindingを補完しています: %#v", binding.Inputs)
	}
}

func TestParseHeaderDetailContractQueryBindingRejectsDuplicateSingletonField(t *testing.T) {
	src := `header-detail-contract-query-binding {
    application order_entry
    application other_app
    pattern order_entry.order_lines
    datasource sales
    database sales_core
    contract sales_core.order_line
    query find_by_order
}`

	if _, err := NewParser(src).ParseHeaderDetailContractQueryBinding(); err == nil {
		t.Fatal("Applicationを二重定義したBinding sourceを受理しています")
	}
}

func TestParseHeaderDetailContractQueryBindingRequiresExplicitIdentityFields(t *testing.T) {
	src := `header-detail-contract-query-binding {
    application order_entry
    pattern order_entry.order_lines
    datasource sales
    database sales_core
    query find_by_order
}`

	if _, err := NewParser(src).ParseHeaderDetailContractQueryBinding(); err == nil {
		t.Fatal("Contractをfile pathや他の情報から補完しています")
	}
}

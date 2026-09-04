package realization

import "testing"

func headerDetailContractQueryBindingFixture() HeaderDetailContractQueryBinding {
	return HeaderDetailContractQueryBinding{
		Application: "order_entry",
		Pattern:     "order_entry.order_lines",
		Datasource:  "sales",
		Target: ContractQueryTarget{
			Database: "sales_core",
			Contract: "sales_core.order_line",
			Query:    "find_by_order",
		},
		Inputs: []HeaderDetailContractQueryInputBinding{
			{SourceAttribute: "sales.order.order_id", Input: "order_id"},
		},
	}
}

func TestHeaderDetailContractQueryBindingIdentityUsesApplicationAndPatternOnly(t *testing.T) {
	binding := headerDetailContractQueryBindingFixture()
	if got := binding.Identity(); got != "order_entry/order_entry.order_lines" {
		t.Fatalf("header-detail Contract Query binding identity=%q", got)
	}

	changed := binding
	changed.Datasource = "archive"
	changed.Target = ContractQueryTarget{
		Database: "archive_sales",
		Contract: "archive_sales.order_line",
		Query:    "list_by_order",
	}
	if changed.Identity() != binding.Identity() {
		t.Fatalf("実現先を変更するとApplication Patternのbinding identityまで変わっています: before=%q after=%q", binding.Identity(), changed.Identity())
	}
}

func TestHeaderDetailContractQueryBindingRequiresExplicitIdentityAndTarget(t *testing.T) {
	valid := headerDetailContractQueryBindingFixture()
	if !valid.Valid() {
		t.Fatalf("完全なheader-detail Contract Query bindingを無効として扱っています: %#v", valid)
	}

	tests := []struct {
		name   string
		mutate func(*HeaderDetailContractQueryBinding)
	}{
		{name: "Application", mutate: func(binding *HeaderDetailContractQueryBinding) { binding.Application = "" }},
		{name: "Pattern", mutate: func(binding *HeaderDetailContractQueryBinding) { binding.Pattern = "" }},
		{name: "Datasource", mutate: func(binding *HeaderDetailContractQueryBinding) { binding.Datasource = "" }},
		{name: "Database", mutate: func(binding *HeaderDetailContractQueryBinding) { binding.Target.Database = "" }},
		{name: "Contract", mutate: func(binding *HeaderDetailContractQueryBinding) { binding.Target.Contract = "" }},
		{name: "Query", mutate: func(binding *HeaderDetailContractQueryBinding) { binding.Target.Query = "" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binding := valid
			tt.mutate(&binding)
			if binding.Valid() {
				t.Fatalf("%sが未指定のbindingを有効として扱っています: %#v", tt.name, binding)
			}
		})
	}
}

func TestHeaderDetailContractQueryInputBindingKeepsSourceAttributeDirection(t *testing.T) {
	binding := headerDetailContractQueryBindingFixture()
	if len(binding.Inputs) != 1 {
		t.Fatalf("input binding数=%d", len(binding.Inputs))
	}
	input := binding.Inputs[0]
	if input.SourceAttribute != "sales.order.order_id" || input.Input != "order_id" {
		t.Fatalf("source Semantic attributeからQuery inputへの明示bindingが変わっています: %#v", input)
	}
}

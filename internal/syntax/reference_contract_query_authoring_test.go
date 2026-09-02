package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/syntax"
)

func TestParseReferencePatternKeepsExplicitContractQueryInvocation(t *testing.T) {
	src := `pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
    contract-query {
        contract master_data.customer
        query find_by_id
        input customer.customer.tenant_id -> tenant
        input customer.customer.customer_id -> id
    }
}`

	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatalf("reference PatternのContract Query呼び出しを読み取れません: %v", err)
	}
	if got.Kind != application.PatternReference {
		t.Fatalf("Pattern Kindが変わっています: %s", got.Kind)
	}
	if got.ContractQuery == nil {
		t.Fatal("Contract Query呼び出しがCanonical Patternへ保持されていません")
	}
	if got.ContractQuery.Contract != "master_data.customer" || got.ContractQuery.Query != "find_by_id" {
		t.Fatalf("Contract / Query identityを保持できていません: %#v", got.ContractQuery)
	}
	if len(got.ContractQuery.Inputs) != 2 {
		t.Fatalf("input binding数が変わっています: %d", len(got.ContractQuery.Inputs))
	}
	if got.ContractQuery.Inputs[0].TargetAttribute != "customer.customer.tenant_id" || got.ContractQuery.Inputs[0].Input != "tenant" {
		t.Fatalf("先頭のinput binding順序を保持できていません: %#v", got.ContractQuery.Inputs)
	}
	if got.ContractQuery.Inputs[1].TargetAttribute != "customer.customer.customer_id" || got.ContractQuery.Inputs[1].Input != "id" {
		t.Fatalf("後続のinput binding順序を保持できていません: %#v", got.ContractQuery.Inputs)
	}
}

func TestParseReferencePatternRejectsDuplicateContractQueryBlock(t *testing.T) {
	src := `pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
    contract-query {
        contract master_data.customer
        query find_by_id
    }
    contract-query {
        contract master_data.customer
        query find_by_id
    }
}`

	_, err := syntax.NewParser(src).ParseApplicationPattern()
	if err == nil {
		t.Fatal("同じPatternにcontract-queryを複数定義できてしまいました")
	}
	if !strings.Contains(err.Error(), "contract-queryは一つのPatternで一度だけ定義できます") {
		t.Fatalf("重複定義の理由を追えないerrorです: %v", err)
	}
}

func TestParseReferencePatternRejectsIncompleteContractQueryBlock(t *testing.T) {
	for name, src := range map[string]string{
		"contract": `pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
    contract-query {
        query find_by_id
    }
}`,
		"query": `pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
    contract-query {
        contract master_data.customer
    }
}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := syntax.NewParser(src).ParseApplicationPattern()
			if err == nil {
				t.Fatalf("contract-queryから%sを省略できてしまいました", name)
			}
			if !strings.Contains(err.Error(), "contract-queryには"+name+"の指定が必要です") {
				t.Fatalf("不足している%sを判別できないerrorです: %v", name, err)
			}
		})
	}
}

func TestParseNonReferencePatternRejectsContractQueryInvocation(t *testing.T) {
	src := `pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
    contract-query {
        contract master_data.customer
        query find_by_id
    }
}`

	_, err := syntax.NewParser(src).ParseApplicationPattern()
	if err == nil {
		t.Fatal("reference以外のPatternでcontract-queryを利用できてしまいました")
	}
	if !strings.Contains(err.Error(), "does not accept contract-query") {
		t.Fatalf("Pattern Kindの境界を判別できないerrorです: %v", err)
	}
}

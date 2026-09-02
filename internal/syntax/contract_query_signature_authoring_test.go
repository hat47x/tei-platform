package syntax_test

import (
	"testing"

	"tei/internal/database"
	"tei/internal/syntax"
)

func TestParseContractQuerySignatureKeepsTypedInputAndResult(t *testing.T) {
	src := `contract master_data.customer v2.4 {
    read customer.customer
    query find_by_id {
        input id type customer.customer_id
        result customer.customer zero-or-one
    }
}`

	contract, err := syntax.NewParser(src).ParseContract()
	if err != nil {
		t.Fatalf("typed Contract Queryを読み取れません: %v", err)
	}
	if len(contract.Queries) != 1 {
		t.Fatalf("Contract Query数が期待値と異なります: %#v", contract.Queries)
	}

	queryContract := contract.Queries[0]
	if queryContract.Name != "find_by_id" {
		t.Fatalf("Query名を変更しています: %#v", queryContract)
	}
	if len(queryContract.Inputs) != 1 || queryContract.Inputs[0].Name != "id" || queryContract.Inputs[0].Type != "customer.customer_id" {
		t.Fatalf("typed inputを保持できていません: %#v", queryContract.Inputs)
	}
	if queryContract.Result != "customer.customer" || queryContract.ResultCardinality != database.QueryResultZeroOrOne {
		t.Fatalf("result契約を保持できていません: %#v", queryContract)
	}
}

func TestParseContractQuerySignatureKeepsInputOrder(t *testing.T) {
	src := `contract sample.lookup v1.0 {
    query find {
        input tenant type sample.tenant_id
        input item type sample.item_id
        result sample.item one
    }
}`

	contract, err := syntax.NewParser(src).ParseContract()
	if err != nil {
		t.Fatalf("複数inputを持つContract Queryを読み取れません: %v", err)
	}
	inputs := contract.Queries[0].Inputs
	if len(inputs) != 2 || inputs[0].Name != "tenant" || inputs[1].Name != "item" {
		t.Fatalf("inputのauthor順を保持できていません: %#v", inputs)
	}
}

func TestParseContractQuerySignatureKeepsLegacyNameOnlyQuery(t *testing.T) {
	src := `contract master_data.customer v2.4 {
    read customer.customer
    query find_by_id
}`

	contract, err := syntax.NewParser(src).ParseContract()
	if err != nil {
		t.Fatalf("従来のContract Query構文を読み取れません: %v", err)
	}
	if len(contract.Queries) != 1 {
		t.Fatalf("Contract Query数が期待値と異なります: %#v", contract.Queries)
	}
	queryContract := contract.Queries[0]
	if queryContract.Name != "find_by_id" || len(queryContract.Inputs) != 0 || queryContract.Result != "" || queryContract.ResultCardinality != "" {
		t.Fatalf("legacy Queryへ暗黙のsignatureを追加しています: %#v", queryContract)
	}
}

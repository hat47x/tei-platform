package syntax_test

import (
	"strings"
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

func TestParseContractQuerySignatureAllowsInputlessTypedQuery(t *testing.T) {
	src := `contract master_data.customer v2.4 {
    query list_all {
        result customer.customer many
    }
}`

	contract, err := syntax.NewParser(src).ParseContract()
	if err != nil {
		t.Fatalf("inputを持たないtyped Contract Queryを読み取れません: %v", err)
	}
	queryContract := contract.Queries[0]
	if len(queryContract.Inputs) != 0 || queryContract.ResultCardinality != database.QueryResultMany {
		t.Fatalf("inputなしQueryのsignatureが変わっています: %#v", queryContract)
	}
}

func TestParseContractQuerySignatureRejectsEmptyBlock(t *testing.T) {
	src := `contract master_data.customer v2.4 {
    query find_by_id {
    }
}`

	_, err := syntax.NewParser(src).ParseContract()
	if err == nil {
		t.Fatal("明示した空のQuery blockをlegacy Queryとして受理しています")
	}
	if !strings.Contains(err.Error(), "Query signatureを定義する場合はresultが必要です") {
		t.Fatalf("空のQuery signatureを拒否した理由を追えません: %v", err)
	}
}

func TestParseContractQuerySignatureRejectsDuplicateResult(t *testing.T) {
	src := `contract master_data.customer v2.4 {
    query find_by_id {
        result customer.customer zero-or-one
        result customer.customer one
    }
}`

	_, err := syntax.NewParser(src).ParseContract()
	if err == nil {
		t.Fatal("一つのQuery signatureへresultを複数定義できています")
	}
	if !strings.Contains(err.Error(), "resultを一度だけ定義できます") {
		t.Fatalf("result重複を拒否した理由を追えません: %v", err)
	}
}

func TestParseContractQuerySignatureRejectsUnknownCardinality(t *testing.T) {
	src := `contract master_data.customer v2.4 {
    query find_by_id {
        result customer.customer optional
    }
}`

	_, err := syntax.NewParser(src).ParseContract()
	if err == nil {
		t.Fatal("未定義のQuery result cardinalityを受理しています")
	}
	if !strings.Contains(err.Error(), "Query result cardinality \"optional\"は定義されていません") {
		t.Fatalf("未定義cardinalityを拒否した理由を追えません: %v", err)
	}
}

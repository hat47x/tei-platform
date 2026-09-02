package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/database"
	"tei/internal/syntax"
)

func TestParseContractAuthoringKeepsLegacyQuery(t *testing.T) {
	src := `contract master_data.customer v2.4 {
    read customer.customer
    query find_by_id
}`

	got, err := syntax.NewParser(src).ParseContractAuthoring()
	if err != nil {
		t.Fatalf("既存のContract Query構文を読み取れません: %v", err)
	}
	if got.ID != "master_data.customer" || got.Version.Major != 2 || got.Version.Minor != 4 {
		t.Fatalf("Contract identity / versionが変わっています: %#v", got)
	}
	if len(got.Reads) != 1 || got.Reads[0] != "customer.customer" {
		t.Fatalf("既存のread契約を保持できていません: %#v", got.Reads)
	}
	if len(got.Queries) != 1 {
		t.Fatalf("既存のQuery数が変わっています: %d", len(got.Queries))
	}
	query := got.Queries[0]
	if query.Name != "find_by_id" || len(query.Inputs) != 0 || query.Result != "" || query.ResultCardinality != "" {
		t.Fatalf("名前だけのlegacy Queryへsignatureを補っています: %#v", query)
	}
}

func TestParseContractAuthoringKeepsTypedQuerySignature(t *testing.T) {
	src := `contract master_data.customer v2.4 {
    read customer.customer
    query find_by_id {
        input tenant type customer.tenant_id
        input id type customer.customer_id
        result customer.customer zero-or-one
    }
}`

	got, err := syntax.NewParser(src).ParseContractAuthoring()
	if err != nil {
		t.Fatalf("typed Query signatureを読み取れません: %v", err)
	}
	if len(got.Queries) != 1 {
		t.Fatalf("Query数が変わっています: %d", len(got.Queries))
	}
	query := got.Queries[0]
	if query.Name != "find_by_id" {
		t.Fatalf("Query名を保持できていません: %#v", query)
	}
	if len(query.Inputs) != 2 {
		t.Fatalf("Query input数が変わっています: %d", len(query.Inputs))
	}
	if query.Inputs[0].Name != "tenant" || query.Inputs[0].Type != "customer.tenant_id" {
		t.Fatalf("先頭inputのauthor順序またはSemantic型を保持できていません: %#v", query.Inputs)
	}
	if query.Inputs[1].Name != "id" || query.Inputs[1].Type != "customer.customer_id" {
		t.Fatalf("後続inputのauthor順序またはSemantic型を保持できていません: %#v", query.Inputs)
	}
	if query.Result != "customer.customer" || query.ResultCardinality != database.QueryResultZeroOrOne {
		t.Fatalf("Query result契約を保持できていません: %#v", query)
	}
}

func TestParseContractAuthoringAllowsInputlessTypedQuery(t *testing.T) {
	src := `contract master_data.customer v2.4 {
    read customer.customer
    query list_all {
        result customer.customer many
    }
}`

	got, err := syntax.NewParser(src).ParseContractAuthoring()
	if err != nil {
		t.Fatalf("inputを持たないtyped Queryを読み取れません: %v", err)
	}
	query := got.Queries[0]
	if len(query.Inputs) != 0 || query.ResultCardinality != database.QueryResultMany {
		t.Fatalf("inputなしQueryのsignatureが変わっています: %#v", query)
	}
}

func TestParseContractAuthoringRejectsEmptySignatureBlock(t *testing.T) {
	src := `contract master_data.customer v2.4 {
    read customer.customer
    query find_by_id {
    }
}`

	_, err := syntax.NewParser(src).ParseContractAuthoring()
	if err == nil {
		t.Fatal("空のQuery blockがlegacy Queryと同じ意味に落ちています")
	}
	if !strings.Contains(err.Error(), "Query signatureを定義する場合はresultが必要です") {
		t.Fatalf("空signatureを拒否した理由を追えないerrorです: %v", err)
	}
}

func TestParseContractAuthoringRejectsDuplicateResult(t *testing.T) {
	src := `contract master_data.customer v2.4 {
    read customer.customer
    query find_by_id {
        result customer.customer zero-or-one
        result customer.customer one
    }
}`

	_, err := syntax.NewParser(src).ParseContractAuthoring()
	if err == nil {
		t.Fatal("一つのQuery signatureへresultを複数定義できてしまいました")
	}
	if !strings.Contains(err.Error(), "resultを一度だけ定義できます") {
		t.Fatalf("result重複を判別できないerrorです: %v", err)
	}
}

func TestParseContractAuthoringRejectsUnknownResultCardinality(t *testing.T) {
	src := `contract master_data.customer v2.4 {
    read customer.customer
    query find_by_id {
        result customer.customer optional
    }
}`

	_, err := syntax.NewParser(src).ParseContractAuthoring()
	if err == nil {
		t.Fatal("未定義のQuery result cardinalityを受理しています")
	}
	if !strings.Contains(err.Error(), "Query result cardinality \"optional\"は定義されていません") {
		t.Fatalf("未定義cardinalityを判別できないerrorです: %v", err)
	}
}

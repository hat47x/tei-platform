package driver

import (
	"strings"
	"testing"

	"tei/internal/driver/queryresult"
	"tei/internal/environment"
	"tei/internal/query"
	"tei/internal/realization"
)

func contractQuerySQLResultFixture() realization.ContractQuerySQLRealization {
	return realization.ContractQuerySQLRealization{
		Target: realization.ContractQueryTarget{
			Database: "master_data",
			Contract: "master_data.customer",
			Query:    "find_by_id",
		},
		Provider: environment.ProviderOracle,
		Source:   "select ...",
		ResultColumns: []query.ResultColumnMapping{
			{Column: "CUSTOMER_ID", Attribute: "customer.customer.customer_id"},
			{Column: "NAME", Attribute: "customer.customer.name"},
		},
	}
}

func TestMapContractQuerySQLResultUsesExplicitColumnMapping(t *testing.T) {
	r := contractQuerySQLResultFixture()
	raw := queryresult.RawResult{
		Columns: []string{"NAME", "CUSTOMER_ID"},
		Rows:    [][]any{{"Alice", int64(42)}},
	}

	result, err := MapContractQuerySQLResult(r, raw)
	if err != nil {
		t.Fatalf("SQL raw resultをSemantic rowへ変換できません: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("Semantic row数が変わっています: %#v", result.Rows)
	}
	row := result.Rows[0].Attributes
	if row["customer.customer.customer_id"] != int64(42) || row["customer.customer.name"] != "Alice" {
		t.Fatalf("raw列順から意味を推測しています: %#v", row)
	}
}

func TestMapContractQuerySQLResultRejectsExtraRawColumn(t *testing.T) {
	r := contractQuerySQLResultFixture()
	raw := queryresult.RawResult{
		Columns: []string{"CUSTOMER_ID", "NAME", "STATUS"},
		Rows:    [][]any{{int64(42), "Alice", "active"}},
	}

	_, err := MapContractQuerySQLResult(r, raw)
	if err == nil || !strings.Contains(err.Error(), "raw result列 \"STATUS\"にSemantic attributeのmappingがありません") {
		t.Fatalf("未mappingの物理列を黙って捨てています: %v", err)
	}
}

func TestMapContractQuerySQLResultRejectsMissingRawColumn(t *testing.T) {
	r := contractQuerySQLResultFixture()
	raw := queryresult.RawResult{
		Columns: []string{"CUSTOMER_ID"},
		Rows:    [][]any{{int64(42)}},
	}

	_, err := MapContractQuerySQLResult(r, raw)
	if err == nil || !strings.Contains(err.Error(), "raw列 \"NAME\"が実行結果にありません") {
		t.Fatalf("欠けた物理列をSemantic属性から補完しています: %v", err)
	}
}

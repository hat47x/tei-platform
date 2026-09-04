package queryresult

import (
	"strings"
	"testing"

	"tei/internal/query"
)

func TestMapColumnsAppliesExplicitMappingWithoutColumnOrderInference(t *testing.T) {
	mappings := []query.ResultColumnMapping{
		{Column: "CUSTOMER_ID", Attribute: "customer.customer.customer_id"},
		{Column: "NAME", Attribute: "customer.customer.name"},
	}
	raw := RawResult{
		Columns: []string{"NAME", "CUSTOMER_ID"},
		Rows:    [][]any{{"Alice", int64(42)}},
	}

	mapped, err := MapColumns("Contract Query master_data/master_data.customer/find_by_id", mappings, raw)
	if err != nil {
		t.Fatalf("明示mappingをraw resultへ適用できません: %v", err)
	}
	if len(mapped.Columns) != 2 || mapped.Columns[0].Attribute != "customer.customer.customer_id" || mapped.Columns[1].Attribute != "customer.customer.name" {
		t.Fatalf("mapping順序またはSemantic attributeが変わっています: %#v", mapped.Columns)
	}
	if mapped.Rows[0][0] != int64(42) || mapped.Rows[0][1] != "Alice" {
		t.Fatalf("raw列順から値を推測して並べています: %#v", mapped.Rows)
	}
}

func TestMapColumnsRejectsUnmappedRawColumn(t *testing.T) {
	mappings := []query.ResultColumnMapping{
		{Column: "CUSTOMER_ID", Attribute: "customer.customer.customer_id"},
	}
	raw := RawResult{
		Columns: []string{"CUSTOMER_ID", "NAME"},
		Rows:    [][]any{{int64(42), "Alice"}},
	}

	_, err := MapColumns("Contract Query master_data/master_data.customer/find_by_id", mappings, raw)
	if err == nil || !strings.Contains(err.Error(), "raw result列 \"NAME\"にSemantic attributeのmappingがありません") {
		t.Fatalf("未mappingのraw列を黙って捨てています: %v", err)
	}
}

func TestMapColumnsRejectsMissingRawColumn(t *testing.T) {
	mappings := []query.ResultColumnMapping{
		{Column: "CUSTOMER_ID", Attribute: "customer.customer.customer_id"},
		{Column: "NAME", Attribute: "customer.customer.name"},
	}
	raw := RawResult{
		Columns: []string{"CUSTOMER_ID"},
		Rows:    [][]any{{int64(42)}},
	}

	_, err := MapColumns("Contract Query master_data/master_data.customer/find_by_id", mappings, raw)
	if err == nil || !strings.Contains(err.Error(), "result mappingが要求するraw列 \"NAME\"が実行結果にありません") {
		t.Fatalf("必要なraw列の欠如を推測で補っています: %v", err)
	}
}

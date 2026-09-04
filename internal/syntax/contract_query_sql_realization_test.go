package syntax

import (
	"strings"
	"testing"
)

func TestParseContractQuerySQLRealizationKeepsPhysicalMappingExplicit(t *testing.T) {
	src := `contract-query-sql-realization {
    database master_data
    contract master_data.customer
    query find_by_id
    sql oracle file "find-by-id.oracle.sql"
    input id -> customer_id
    result {
        column CUSTOMER_ID -> customer.customer.customer_id
        column NAME -> customer.customer.name
    }
}`

	authoring, err := NewParser(src).ParseContractQuerySQLRealizationAuthoring()
	if err != nil {
		t.Fatalf("Contract Query SQL realizationを読み込めません: %v", err)
	}
	if authoring.Value.Target.Identity() != "master_data/master_data.customer/find_by_id" {
		t.Fatalf("target identityが変わっています: %#v", authoring.Value.Target)
	}
	if authoring.Value.Provider != "oracle" || authoring.NativeSQLPath != "find-by-id.oracle.sql" {
		t.Fatalf("provider / native SQL pathが変わっています: %#v", authoring)
	}
	if len(authoring.Value.Inputs) != 1 || authoring.Value.Inputs[0].Input != "id" || authoring.Value.Inputs[0].Parameter != "customer_id" {
		t.Fatalf("Contract input→物理parameter mappingが変わっています: %#v", authoring.Value.Inputs)
	}
	if len(authoring.Value.ResultColumns) != 2 || authoring.Value.ResultColumns[0].Column != "CUSTOMER_ID" || authoring.Value.ResultColumns[0].Attribute != "customer.customer.customer_id" {
		t.Fatalf("raw result→Semantic attribute mappingが変わっています: %#v", authoring.Value.ResultColumns)
	}
}

func TestParseContractQuerySQLRealizationAllowsInlineNativeSQLWithoutInferringMappings(t *testing.T) {
	src := `contract-query-sql-realization {
    database master_data
    contract master_data.customer
    query find_by_id
    sql oracle {
        select CUSTOMER_ID from CUSTOMER_MASTER where CUSTOMER_ID = :customer_id
    }
    input id -> customer_id
    result {
        column CUSTOMER_ID -> customer.customer.customer_id
    }
}`

	authoring, err := NewParser(src).ParseContractQuerySQLRealizationAuthoring()
	if err != nil {
		t.Fatalf("inline SQL realizationを読み込めません: %v", err)
	}
	if !strings.Contains(authoring.Value.Source, "CUSTOMER_MASTER") || authoring.NativeSQLPath != "" {
		t.Fatalf("inline SQL sourceの扱いが期待値と異なります: %#v", authoring)
	}
}

func TestParseContractQuerySQLRealizationRejectsDuplicateInputMapping(t *testing.T) {
	src := `contract-query-sql-realization {
    database master_data
    contract master_data.customer
    query find_by_id
    sql oracle { select 1 from dual }
    input id -> customer_id
    input id -> alternate_customer_id
    result { column CUSTOMER_ID -> customer.customer.customer_id }
}`

	_, err := NewParser(src).ParseContractQuerySQLRealizationAuthoring()
	if err == nil || !strings.Contains(err.Error(), "input idのparameter mappingが重複") {
		t.Fatalf("同じContract inputへの複数parameter mappingを受け入れています: %v", err)
	}
}

func TestParseContractQuerySQLRealizationRejectsDuplicatePhysicalParameter(t *testing.T) {
	src := `contract-query-sql-realization {
    database master_data
    contract master_data.customer
    query find_by_id
    sql oracle { select 1 from dual }
    input id -> customer_id
    input tenant -> customer_id
    result { column CUSTOMER_ID -> customer.customer.customer_id }
}`

	_, err := NewParser(src).ParseContractQuerySQLRealizationAuthoring()
	if err == nil || !strings.Contains(err.Error(), "physical parameter \"customer_id\"が重複") {
		t.Fatalf("複数Contract inputを同じ物理parameterへ対応付けています: %v", err)
	}
}

func TestParseContractQuerySQLRealizationDoesNotAcceptProviderAsTargetIdentity(t *testing.T) {
	src := `contract-query-sql-realization {
    database master_data
    contract master_data.customer
    sql oracle { select 1 from dual }
    input id -> customer_id
    result { column CUSTOMER_ID -> customer.customer.customer_id }
}`

	_, err := NewParser(src).ParseContractQuerySQLRealizationAuthoring()
	if err == nil || !strings.Contains(err.Error(), "requires database, contract, and query") {
		t.Fatalf("providerやfile contextからQuery identityを補完しています: %v", err)
	}
}

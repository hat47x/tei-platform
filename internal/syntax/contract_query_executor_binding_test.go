package syntax

import (
	"strings"
	"testing"
)

func TestParseContractQueryExecutorBindingKeepsAllIdentityExplicit(t *testing.T) {
	src := `contract-query-executor-binding {
    environment development
    instance master_data.dev
    database master_data
    contract master_data.customer
    query find_by_id
    executor database.query.oracle
}`

	binding, err := NewParser(src).ParseContractQueryExecutorBinding()
	if err != nil {
		t.Fatalf("Contract Query executor Bindingを読み込めません: %v", err)
	}
	if binding.Environment != "development" || binding.Instance != "master_data.dev" {
		t.Fatalf("Environment / instance identityが変わっています: %#v", binding)
	}
	if binding.Target.Database != "master_data" || binding.Target.Contract != "master_data.customer" || binding.Target.Query != "find_by_id" {
		t.Fatalf("Database / Contract / Query target identityが変わっています: %#v", binding.Target)
	}
	if binding.Executor != "database.query.oracle" {
		t.Fatalf("executor Capability identityが変わっています: %#v", binding)
	}
}

func TestParseContractQueryExecutorBindingDoesNotAcceptMissingExplicitIdentity(t *testing.T) {
	src := `contract-query-executor-binding {
    environment development
    instance master_data.dev
    database master_data
    contract master_data.customer
    executor database.query.oracle
}`

	_, err := NewParser(src).ParseContractQueryExecutorBinding()
	if err == nil || !strings.Contains(err.Error(), "requires database, contract, and query") {
		t.Fatalf("Query名の欠如をファイル名やproviderから補完しています: %v", err)
	}
}

func TestParseContractQueryExecutorBindingRejectsDuplicateField(t *testing.T) {
	src := `contract-query-executor-binding {
    environment development
    environment production
    instance master_data.dev
    database master_data
    contract master_data.customer
    query find_by_id
    executor database.query.oracle
}`

	_, err := NewParser(src).ParseContractQueryExecutorBinding()
	if err == nil || !strings.Contains(err.Error(), "environment is declared more than once") {
		t.Fatalf("同じ意味を複数回宣言したBindingを受け入れています: %v", err)
	}
}

func TestParseContractQueryExecutorBindingRejectsUnknownField(t *testing.T) {
	src := `contract-query-executor-binding {
    environment development
    instance master_data.dev
    database master_data
    contract master_data.customer
    query find_by_id
    provider oracle
    executor database.query.oracle
}`

	_, err := NewParser(src).ParseContractQueryExecutorBinding()
	if err == nil || !strings.Contains(err.Error(), "expected environment, instance, database, contract, query, or executor") {
		t.Fatalf("Binding sourceへprovider選択を混ぜています: %v", err)
	}
}

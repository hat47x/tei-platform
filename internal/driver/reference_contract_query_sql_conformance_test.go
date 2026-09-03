package driver_test

import (
	"context"
	"testing"

	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/environment"
	"tei/internal/query"
	"tei/internal/realization"
)

type referenceSQLRunner struct {
	source     string
	parameters map[string]any
	result     sqlquery.RawExecution
}

func (r *referenceSQLRunner) ExecuteSQL(
	_ context.Context,
	source string,
	parameters map[string]any,
) (sqlquery.RawExecution, error) {
	r.source = source
	r.parameters = make(map[string]any, len(parameters))
	for key, value := range parameters {
		r.parameters[key] = value
	}
	return r.result, nil
}

func referenceSQLRealization() realization.ContractQuerySQLRealization {
	return realization.ContractQuerySQLRealization{
		Target: realization.ContractQueryTarget{
			Database: "master_data",
			Contract: "master_data.customer",
			Query:    "find_by_id",
		},
		Provider: environment.ProviderOracle,
		Source:   "select legacy_customer_id from observed_customer_view where legacy_customer_id = :legacy_id",
		Inputs: []realization.ContractQueryInputParameterMapping{
			{Input: "id", Parameter: "legacy_id"},
		},
		ResultColumns: []query.ResultColumnMapping{
			{Column: "legacy_customer_id", Attribute: "customer.customer.customer_id"},
		},
	}
}

func TestReferenceContractQueryEnvironmentCompositionReachesSQLRunnerWithoutInference(t *testing.T) {
	world := contractQueryEnvironmentWorld()
	world.ContractQuerySQLRealizations = []realization.ContractQuerySQLRealization{referenceSQLRealization()}
	runner := &referenceSQLRunner{
		result: sqlquery.RawExecution{
			Result: queryresult.RawResult{
				Columns: []string{"legacy_customer_id"},
				Rows:    [][]any{{int64(42)}},
			},
			Payload: "synthetic-reference-evidence",
		},
	}
	executor := sqlquery.Executor{
		Provider:     environment.ProviderOracle,
		Realizations: world.ContractQuerySQLRealizations,
		Runner:       runner,
	}
	registry := contractQueryExecutorRegistry("master_data.dev", executor)

	result, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		registry,
	)
	if err != nil {
		t.Fatalf("Environment compositionからSQL Runnerまでreference Contract Queryを実行できません: %v", err)
	}
	if runner.parameters["legacy_id"] != int64(42) {
		t.Fatalf("Relation key → Contract input → 物理parameterの明示経路が切れています: %#v", runner.parameters)
	}
	if _, exists := runner.parameters["id"]; exists {
		t.Fatalf("Contract input名を物理parameterへ暗黙流用しています: %#v", runner.parameters)
	}
	if runner.source != world.ContractQuerySQLRealizations[0].Source {
		t.Fatalf("明示realization以外のSQL sourceを実行しています: %q", runner.source)
	}
	if len(result.Execution.Result.Rows) != 1 || result.Execution.Result.Rows[0].Attributes["customer.customer.customer_id"] != int64(42) {
		t.Fatalf("raw result → Semantic rowの明示mappingが切れています: %#v", result.Execution.Result.Rows)
	}
	if result.Execution.Result.Payload != "synthetic-reference-evidence" {
		t.Fatalf("provider固有Evidenceが失われています: %#v", result.Execution.Result.Payload)
	}
}

func TestReferenceContractQueryEnvironmentCompositionKeepsZeroRowsAsNormalNotFound(t *testing.T) {
	world := contractQueryEnvironmentWorld()
	world.ContractQuerySQLRealizations = []realization.ContractQuerySQLRealization{referenceSQLRealization()}
	runner := &referenceSQLRunner{
		result: sqlquery.RawExecution{
			Result: queryresult.RawResult{
				Columns: []string{"legacy_customer_id"},
				Rows:    nil,
			},
		},
	}
	executor := sqlquery.Executor{
		Provider:     environment.ProviderOracle,
		Realizations: world.ContractQuerySQLRealizations,
		Runner:       runner,
	}
	registry := contractQueryExecutorRegistry("master_data.dev", executor)

	result, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		registry,
	)
	if err != nil {
		t.Fatalf("zero-or-one Contract Queryの0件をExecution Errorへ変換しています: %v", err)
	}
	if len(result.Execution.Result.Rows) != 0 {
		t.Fatalf("正常なnot-foundが別の結果へ変わっています: %#v", result.Execution.Result.Rows)
	}
}

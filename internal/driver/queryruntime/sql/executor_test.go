package sqlquery

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/database"
	"tei/internal/driver/queryresult"
	"tei/internal/environment"
	"tei/internal/query"
	"tei/internal/realization"
)

type captureRunner struct {
	source     string
	parameters map[string]any
	result     RawExecution
	err        error
}

func (r *captureRunner) ExecuteSQL(_ context.Context, source string, parameters map[string]any) (RawExecution, error) {
	r.source = source
	r.parameters = make(map[string]any, len(parameters))
	for key, value := range parameters {
		r.parameters[key] = value
	}
	if r.err != nil {
		return RawExecution{}, r.err
	}
	return r.result, nil
}

func sqlExecutorFixture(runner Runner) Executor {
	return Executor{
		Provider: environment.ProviderOracle,
		Realizations: []realization.ContractQuerySQLRealization{
			{
				Target: realization.ContractQueryTarget{
					Database: "master_data",
					Contract: "master_data.customer",
					Query:    "find_by_id",
				},
				Provider: environment.ProviderOracle,
				Source:   "select customer_id, customer_name from legacy_customer where customer_id = :customer_id",
				Inputs: []realization.ContractQueryInputParameterMapping{
					{Input: "id", Parameter: "customer_id"},
				},
				ResultColumns: []query.ResultColumnMapping{
					{Column: "customer_id", Attribute: "customer.customer.customer_id"},
					{Column: "customer_name", Attribute: "customer.customer.name"},
				},
			},
		},
		Runner: runner,
	}
}

func sqlExecutorContract() database.QueryContract {
	return database.QueryContract{
		Name: "find_by_id",
		Inputs: []database.QueryInput{
			{Name: "id", Type: "customer.customer_id"},
		},
		Result:            "customer.customer",
		ResultCardinality: database.QueryResultZeroOrOne,
	}
}

func TestExecutorRunsExplicitRealizationAndMapsSemanticResult(t *testing.T) {
	runner := &captureRunner{result: RawExecution{
		Result: queryresult.RawResult{
			Columns: []string{"customer_name", "customer_id"},
			Rows:    [][]any{{"Alice", int64(42)}},
		},
		Payload: "oracle-evidence",
	}}
	executor := sqlExecutorFixture(runner)
	target := executor.Realizations[0].Target

	result, err := executor.ExecuteContractQuery(context.Background(), target, sqlExecutorContract(), map[string]any{"id": int64(42)})
	if err != nil {
		t.Fatalf("明示SQL realizationを実行できません: %v", err)
	}
	if runner.parameters["customer_id"] != int64(42) {
		t.Fatalf("Contract inputを明示した物理parameterへ渡していません: %#v", runner.parameters)
	}
	if _, exists := runner.parameters["id"]; exists {
		t.Fatalf("Contract input名を物理parameter名として暗黙利用しています: %#v", runner.parameters)
	}
	if len(result.Rows) != 1 || result.Rows[0].Attributes["customer.customer.customer_id"] != int64(42) || result.Rows[0].Attributes["customer.customer.name"] != "Alice" {
		t.Fatalf("raw resultを明示mappingからSemantic rowへ変換できていません: %#v", result.Rows)
	}
	if result.Payload != "oracle-evidence" {
		t.Fatalf("provider固有Evidenceを結果へ引き継げていません: %#v", result.Payload)
	}
}

func TestExecutorDoesNotFallbackToDifferentProviderRealization(t *testing.T) {
	runner := &captureRunner{}
	executor := sqlExecutorFixture(runner)
	executor.Provider = environment.ProviderPostgres
	target := executor.Realizations[0].Target

	_, err := executor.ExecuteContractQuery(context.Background(), target, sqlExecutorContract(), map[string]any{"id": int64(42)})
	if err == nil || !strings.Contains(err.Error(), "provider postgresにSQL realizationがありません") {
		t.Fatalf("別providerのrealizationへfallbackしています: %v", err)
	}
}

func TestExecutorRejectsDuplicateTargetProviderRealization(t *testing.T) {
	runner := &captureRunner{}
	executor := sqlExecutorFixture(runner)
	executor.Realizations = append(executor.Realizations, executor.Realizations[0])
	target := executor.Realizations[0].Target

	_, err := executor.ExecuteContractQuery(context.Background(), target, sqlExecutorContract(), map[string]any{"id": int64(42)})
	if err == nil || !strings.Contains(err.Error(), "SQL realizationが複数あります") {
		t.Fatalf("同じtarget + providerのrealization競合を受け入れています: %v", err)
	}
}

func TestExecutorKeepsRunnerFailureDistinctFromEmptyResult(t *testing.T) {
	runner := &captureRunner{err: errors.New("connection lost")}
	executor := sqlExecutorFixture(runner)
	target := executor.Realizations[0].Target

	_, err := executor.ExecuteContractQuery(context.Background(), target, sqlExecutorContract(), map[string]any{"id": int64(42)})
	if err == nil || !strings.Contains(err.Error(), "物理SQL実行に失敗") || !strings.Contains(err.Error(), "connection lost") {
		t.Fatalf("SQL実行errorを正常な0件へ変換しています: %v", err)
	}
}

func TestExecutorRejectsUnmappedRawColumn(t *testing.T) {
	runner := &captureRunner{result: RawExecution{Result: queryresult.RawResult{
		Columns: []string{"customer_id", "customer_name", "status"},
		Rows:    [][]any{{int64(42), "Alice", "active"}},
	}}}
	executor := sqlExecutorFixture(runner)
	target := executor.Realizations[0].Target

	_, err := executor.ExecuteContractQuery(context.Background(), target, sqlExecutorContract(), map[string]any{"id": int64(42)})
	if err == nil || !strings.Contains(err.Error(), "raw result列 \"status\"にSemantic attributeのmappingがありません") {
		t.Fatalf("未mappingの物理列を黙って捨てています: %v", err)
	}
}

func TestExecutorRejectsGenericProvider(t *testing.T) {
	runner := &captureRunner{}
	executor := sqlExecutorFixture(runner)
	executor.Provider = environment.ProviderGeneric
	target := executor.Realizations[0].Target

	_, err := executor.ExecuteContractQuery(context.Background(), target, sqlExecutorContract(), map[string]any{"id": int64(42)})
	if err == nil || !strings.Contains(err.Error(), "provider \"generic\"は実行に使用できません") {
		t.Fatalf("具体providerを持たないSQL executorを実行しています: %v", err)
	}
}

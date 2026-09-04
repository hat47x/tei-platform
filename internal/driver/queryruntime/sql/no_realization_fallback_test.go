package sqlquery

import (
	"context"
	"strings"
	"testing"

	"tei/internal/realization"
)

func TestExecutorDoesNotFallbackAcrossContractQueryTargets(t *testing.T) {
	runner := &captureRunner{}
	executor := sqlExecutorFixture(runner)
	target := realization.ContractQueryTarget{
		Database: "master_data",
		Contract: "master_data.customer",
		Query:    "find_by_code",
	}

	_, err := executor.ExecuteContractQuery(
		context.Background(),
		target,
		sqlExecutorContract(),
		map[string]any{"id": int64(42)},
	)
	if err == nil || !strings.Contains(err.Error(), "master_data/master_data.customer/find_by_code") || !strings.Contains(err.Error(), "SQL realizationがありません") {
		t.Fatalf("同じproviderの別Contract Query realizationへfallbackしています: %v", err)
	}
	if runner.source != "" || len(runner.parameters) != 0 {
		t.Fatalf("realizationを明示解決する前にSQL Runnerを呼び出しています: source=%q parameters=%#v", runner.source, runner.parameters)
	}
}

func TestExecutorDoesNotFallbackAcrossDatabaseTargets(t *testing.T) {
	runner := &captureRunner{}
	executor := sqlExecutorFixture(runner)
	target := executor.Realizations[0].Target
	target.Database = "shadow_master_data"

	_, err := executor.ExecuteContractQuery(
		context.Background(),
		target,
		sqlExecutorContract(),
		map[string]any{"id": int64(42)},
	)
	if err == nil || !strings.Contains(err.Error(), "shadow_master_data/master_data.customer/find_by_id") || !strings.Contains(err.Error(), "SQL realizationがありません") {
		t.Fatalf("同じContract / Query名を持つ別Databaseのrealizationへfallbackしています: %v", err)
	}
	if runner.source != "" || len(runner.parameters) != 0 {
		t.Fatalf("Database targetを明示解決する前にSQL Runnerを呼び出しています: source=%q parameters=%#v", runner.source, runner.parameters)
	}
}

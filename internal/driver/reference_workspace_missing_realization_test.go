package driver_test

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/driver"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/environment"
	"tei/internal/workspace"
)

func TestReferenceWorkspaceStopsBeforePhysicalReadWithoutObservedSQLRealization(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source pathを取得できません")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	loaded, err := workspace.Load(filepath.Join(repositoryRoot, "reference", "workspace"))
	if err != nil {
		t.Fatalf("Reference Workspaceを読み込めません: %v", err)
	}
	for _, sqlRealization := range loaded.World.ContractQuerySQLRealizations {
		if sqlRealization.Target.Identity() == "master_data/master_data.customer/find_by_id" {
			t.Fatalf("観測済みsourceがないmaster_data customer QueryにSQL realizationが生成されています: %#v", sqlRealization)
		}
	}

	runner := &referenceSQLRunner{}
	executor := sqlquery.Executor{
		Provider:     environment.ProviderOracle,
		Realizations: loaded.World.ContractQuerySQLRealizations,
		Runner:       runner,
	}

	_, err = driver.NewResolver(loaded.World).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		contractQueryExecutorRegistry("master_data.dev", executor),
	)
	if err == nil || !strings.Contains(err.Error(), "master_data/master_data.customer/find_by_id") || !strings.Contains(err.Error(), "SQL realizationがありません") {
		t.Fatalf("未観測の物理Queryを明示的な未解決として停止していません: %v", err)
	}
	if runner.source != "" || len(runner.parameters) != 0 {
		t.Fatalf("SQL realizationがないのに物理Runnerを呼び出しています: source=%q parameters=%#v", runner.source, runner.parameters)
	}
}

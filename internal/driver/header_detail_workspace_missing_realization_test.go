package driver_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/driver"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/environment"
)

type headerDetailMissingRealizationRunner struct {
	source     string
	parameters map[string]any
}

func (r *headerDetailMissingRealizationRunner) ExecuteSQL(
	_ context.Context,
	source string,
	parameters map[string]any,
) (sqlquery.RawExecution, error) {
	r.source = source
	r.parameters = parameters
	return sqlquery.RawExecution{}, nil
}

func TestHeaderDetailReferenceWorkspaceStopsBeforePhysicalReadWithoutSQLRealization(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	const targetIdentity = "sales_core/sales_core.order_line/find_by_order"
	// Referenceは現在このrealizationを明示authoringしている。realizationがない状態は、対象を取り除いたWorldで作る。
	kept := world.ContractQuerySQLRealizations[:0:0]
	for _, sqlRealization := range world.ContractQuerySQLRealizations {
		if sqlRealization.Target.Identity() == targetIdentity && sqlRealization.Provider == environment.ProviderPostgres {
			continue
		}
		kept = append(kept, sqlRealization)
	}
	world.ContractQuerySQLRealizations = kept

	runner := &headerDetailMissingRealizationRunner{}
	executor := sqlquery.Executor{
		Provider:     environment.ProviderPostgres,
		Realizations: world.ContractQuerySQLRealizations,
		Runner:       runner,
	}

	_, err := driver.NewResolver(world).ExecuteHeaderDetailContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
		headerDetailExecutorRegistry(executor),
	)
	if err == nil || !strings.Contains(err.Error(), targetIdentity) || !strings.Contains(err.Error(), "SQL realizationがありません") {
		t.Fatalf("未確定のOrderLine物理Queryを明示的な未解決として停止していません: %v", err)
	}
	if runner.source != "" || len(runner.parameters) != 0 {
		t.Fatalf("SQL realizationがないのにPostgreSQL Runnerを呼び出しています: source=%q parameters=%#v", runner.source, runner.parameters)
	}
}

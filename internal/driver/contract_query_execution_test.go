package driver_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
)

type recordingContractQueryExecutor struct {
	databaseID core.CanonicalID
	contractID core.CanonicalID
	query      database.QueryContract
	parameters map[string]any
	result     driver.ContractQueryResult
	err        error
}

func (e *recordingContractQueryExecutor) ExecuteContractQuery(
	_ context.Context,
	databaseID core.CanonicalID,
	contractID core.CanonicalID,
	query database.QueryContract,
	parameters map[string]any,
) (driver.ContractQueryResult, error) {
	e.databaseID = databaseID
	e.contractID = contractID
	e.query = query
	e.parameters = parameters
	return e.result, e.err
}

func TestExecuteReferenceContractQueryPassesExplicitIdentityAndParameters(t *testing.T) {
	world := referenceParameterWorld()
	executor := &recordingContractQueryExecutor{
		result: driver.ContractQueryResult{
			Rows: []driver.ContractQueryRow{
				{Attributes: map[core.CanonicalID]any{
					"customer.customer.customer_id": int64(42),
				}},
			},
		},
	}

	execution, err := driver.NewResolver(world).ExecuteReferenceContractQuery(
		context.Background(),
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		executor,
	)
	if err != nil {
		t.Fatalf("reference Contract Queryを実行できません: %v", err)
	}
	if executor.databaseID != "master_data" || executor.contractID != "master_data.customer" || executor.query.Name != "find_by_id" {
		t.Fatalf("Database / Contract / Query identityが変わっています: database=%s contract=%s query=%s", executor.databaseID, executor.contractID, executor.query.Name)
	}
	if len(executor.parameters) != 1 || executor.parameters["id"] != int64(42) {
		t.Fatalf("明示Relation keyから準備したparameterをExecutorへ渡せていません: %#v", executor.parameters)
	}
	if len(execution.Result.Rows) != 1 || execution.Result.Rows[0].Attributes["customer.customer.customer_id"] != int64(42) {
		t.Fatalf("Semantic result rowを保持できていません: %#v", execution.Result)
	}
}

func TestExecuteReferenceContractQueryAcceptsZeroRowsForZeroOrOne(t *testing.T) {
	world := referenceParameterWorld()
	executor := &recordingContractQueryExecutor{result: driver.ContractQueryResult{Rows: nil}}

	execution, err := driver.NewResolver(world).ExecuteReferenceContractQuery(
		context.Background(),
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		executor,
	)
	if err != nil {
		t.Fatalf("zero-or-one Queryの0件をExecution Errorへ変換しています: %v", err)
	}
	if len(execution.Result.Rows) != 0 {
		t.Fatalf("0件結果が変わっています: %#v", execution.Result.Rows)
	}
}

func TestExecuteReferenceContractQueryRejectsMultipleRowsForZeroOrOne(t *testing.T) {
	world := referenceParameterWorld()
	executor := &recordingContractQueryExecutor{
		result: driver.ContractQueryResult{Rows: []driver.ContractQueryRow{{}, {}}},
	}

	_, err := driver.NewResolver(world).ExecuteReferenceContractQuery(
		context.Background(),
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		executor,
	)
	if err == nil || !strings.Contains(err.Error(), "zero-or-one") || !strings.Contains(err.Error(), "2件") {
		t.Fatalf("zero-or-one Queryの複数件resultをContract違反として拒否していません: %v", err)
	}
}

func TestExecuteReferenceContractQueryDoesNotConvertExecutorErrorToNotFound(t *testing.T) {
	world := referenceParameterWorld()
	executor := &recordingContractQueryExecutor{err: errors.New("database unavailable")}

	_, err := driver.NewResolver(world).ExecuteReferenceContractQuery(
		context.Background(),
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		executor,
	)
	if err == nil || !strings.Contains(err.Error(), "database unavailable") || !strings.Contains(err.Error(), "実行できませんでした") {
		t.Fatalf("Executor障害を正常な0件と区別できていません: %v", err)
	}
}

func TestExecuteReferenceContractQueryRejectsZeroRowsForOne(t *testing.T) {
	world := referenceParameterWorld()
	world.Databases[0].Contracts[0].Queries[0].ResultCardinality = database.QueryResultOne
	executor := &recordingContractQueryExecutor{result: driver.ContractQueryResult{Rows: nil}}

	_, err := driver.NewResolver(world).ExecuteReferenceContractQuery(
		context.Background(),
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		executor,
	)
	if err == nil || !strings.Contains(err.Error(), "oneを要求") || !strings.Contains(err.Error(), "0件") {
		t.Fatalf("one Queryの0件resultをContract違反として拒否していません: %v", err)
	}
}

func TestExecuteReferenceContractQueryAllowsMultipleRowsForMany(t *testing.T) {
	world := referenceParameterWorld()
	world.Databases[0].Contracts[0].Queries[0].ResultCardinality = database.QueryResultMany
	executor := &recordingContractQueryExecutor{
		result: driver.ContractQueryResult{Rows: []driver.ContractQueryRow{{}, {}}},
	}

	execution, err := driver.NewResolver(world).ExecuteReferenceContractQuery(
		context.Background(),
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		executor,
	)
	if err != nil {
		t.Fatalf("many Queryの複数件resultを拒否しています: %v", err)
	}
	if len(execution.Result.Rows) != 2 {
		t.Fatalf("many Queryのrow数が変わっています: %#v", execution.Result.Rows)
	}
}

func TestExecuteReferenceContractQueryRejectsAttributeOutsideResultData(t *testing.T) {
	world := referenceParameterWorld()
	executor := &recordingContractQueryExecutor{
		result: driver.ContractQueryResult{
			Rows: []driver.ContractQueryRow{
				{Attributes: map[core.CanonicalID]any{
					"sales.order.order_id": int64(1001),
				}},
			},
		},
	}

	_, err := driver.NewResolver(world).ExecuteReferenceContractQuery(
		context.Background(),
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		executor,
	)
	if err == nil || !strings.Contains(err.Error(), "customer.customerへ属さない") || !strings.Contains(err.Error(), "sales.order.order_id") {
		t.Fatalf("result Data外のSemantic attributeを受け入れています: %v", err)
	}
}

func TestExecuteReferenceContractQueryRejectsNilExecutor(t *testing.T) {
	world := referenceParameterWorld()

	_, err := driver.NewResolver(world).ExecuteReferenceContractQuery(
		context.Background(),
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "Executorが指定されていません") {
		t.Fatalf("Executor未指定を明示的に拒否していません: %v", err)
	}
}

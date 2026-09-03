package driver_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
)

func TestExecuteHeaderDetailContractQueryPassesExplicitTargetAndParameters(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	executor := &recordingContractQueryExecutor{
		result: driver.ContractQueryResult{
			Rows: []driver.ContractQueryRow{
				{Attributes: map[core.CanonicalID]any{
					"sales.order_line.order_line_id": int64(11),
				}},
				{Attributes: map[core.CanonicalID]any{
					"sales.order_line.order_line_id": int64(12),
				}},
			},
		},
	}

	execution, err := driver.NewResolver(world).ExecuteHeaderDetailContractQuery(
		context.Background(),
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
		executor,
	)
	if err != nil {
		t.Fatalf("header-detail Contract Queryを実行できません: %v", err)
	}
	if executor.target.Identity() != "sales_core/sales_core.order_line/find_by_order" {
		t.Fatalf("Bindingで選んだContract Query targetが変わっています: %s", executor.target.Identity())
	}
	if executor.query.Name != "find_by_order" {
		t.Fatalf("Contract Query signatureが変わっています: %#v", executor.query)
	}
	if len(executor.parameters) != 1 || executor.parameters["order_id"] != int64(1001) {
		t.Fatalf("source Order identityから準備したparameterをExecutorへ渡せていません: %#v", executor.parameters)
	}
	if len(execution.Result.Rows) != 2 {
		t.Fatalf("many resultをcollectionとして保持できていません: %#v", execution.Result.Rows)
	}
}

func TestExecuteHeaderDetailContractQueryAcceptsZeroRowsAsEmptyCollection(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	executor := &recordingContractQueryExecutor{result: driver.ContractQueryResult{Rows: nil}}

	execution, err := driver.NewResolver(world).ExecuteHeaderDetailContractQuery(
		context.Background(),
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
		executor,
	)
	if err != nil {
		t.Fatalf("many Queryの0件をExecution Errorへ変換しています: %v", err)
	}
	if len(execution.Result.Rows) != 0 {
		t.Fatalf("正常な空collectionが変わっています: %#v", execution.Result.Rows)
	}
}

func TestExecuteHeaderDetailContractQueryDoesNotConvertExecutorErrorToEmptyCollection(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	executor := &recordingContractQueryExecutor{err: errors.New("database unavailable")}

	_, err := driver.NewResolver(world).ExecuteHeaderDetailContractQuery(
		context.Background(),
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
		executor,
	)
	if err == nil || !strings.Contains(err.Error(), "database unavailable") || !strings.Contains(err.Error(), "実行できませんでした") {
		t.Fatalf("Executor障害を正常な空collectionと区別できていません: %v", err)
	}
}

func TestExecuteHeaderDetailContractQueryRejectsAttributeOutsideOrderLineResult(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	executor := &recordingContractQueryExecutor{
		result: driver.ContractQueryResult{
			Rows: []driver.ContractQueryRow{
				{Attributes: map[core.CanonicalID]any{
					"sales.order.order_id": int64(1001),
				}},
			},
		},
	}

	_, err := driver.NewResolver(world).ExecuteHeaderDetailContractQuery(
		context.Background(),
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
		executor,
	)
	if err == nil || !strings.Contains(err.Error(), "sales.order_lineへ属さない") || !strings.Contains(err.Error(), "sales.order.order_id") {
		t.Fatalf("OrderLine result Data外のSemantic attributeを受け入れています: %v", err)
	}
}

func TestExecuteHeaderDetailContractQueryRejectsNilExecutor(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)

	_, err := driver.NewResolver(world).ExecuteHeaderDetailContractQuery(
		context.Background(),
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "Executorが指定されていません") {
		t.Fatalf("Executor未指定を明示的に拒否していません: %v", err)
	}
}

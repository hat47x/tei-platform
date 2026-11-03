package driver_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"

	_ "modernc.org/sqlite"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/operations"
	"tei/internal/realization"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
)

// This is a real-provider round trip on a second provider (#1127 decision:
// verify on an alternative provider). It drives the standard-crud derived update
// through the driver into a SQLite compare-and-swap adapter and reads the row
// through an independent connection.
func TestStandardCRUDUpdateAgainstSQLite(t *testing.T) {
	databaseFile := filepath.Join(t.TempDir(), "standard_crud.db")
	writer, err := sql.Open("sqlite", databaseFile)
	if err != nil {
		t.Fatal("failed to open the SQLite writer pool")
	}
	t.Cleanup(func() { writer.Close() })
	reader, err := sql.Open("sqlite", databaseFile)
	if err != nil {
		t.Fatal("failed to open the independent reader pool")
	}
	t.Cleanup(func() { reader.Close() })

	ctx := context.Background()
	if _, err := writer.ExecContext(ctx, `CREATE TABLE order_status (
		order_id TEXT PRIMARY KEY,
		status TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := writer.ExecContext(ctx, `INSERT INTO order_status (order_id, status) VALUES ('O-1', 'new')`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	independentStatus := func() string {
		t.Helper()
		var status string
		if err := reader.QueryRowContext(ctx, `SELECT status FROM order_status WHERE order_id = 'O-1'`).Scan(&status); err != nil {
			t.Fatalf("independent read: %v", err)
		}
		return status
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	workspaceRoot := filepath.Join(root, "reference", "standard-crud", "workspace")
	loaded, err := workspace.Load(workspaceRoot)
	if err != nil {
		t.Fatalf("load standard-crud: %v", err)
	}
	surface, err := operations.SurfaceWorkspace(workspaceRoot, "ledger")
	if err != nil {
		t.Fatalf("derive surface: %v", err)
	}
	var update operations.Operation
	found := false
	for _, operation := range surface.Operations {
		if operation.ID == "ledger.update.sales.order" {
			update = operation
			found = true
			break
		}
	}
	if !found || update.Concurrency != "expected-state" || update.Binding == "" {
		t.Fatalf("derived update is not bound to expected-state execution: %+v", update)
	}

	resolver := driver.NewResolver(loaded.World)
	adapter := bindingAdapterResolverFunc(func(_ context.Context, _ realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
		return capabilityAdapterFunc(func(ctx context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
			values, _ := input.(map[string]any)
			orderID, _ := values["orderId"].(string)
			expected, _ := values["expectedStatus"].(string)
			return compareAndSwapStatusSQLite(ctx, writer, string(capability), orderID, expected)
		}), nil
	})
	invoke := func(expected string) capabilityinvoke.BusinessResult {
		t.Helper()
		execution, err := resolver.InvokeCapability(
			context.Background(), "ledger", core.CanonicalID(update.Binding), adapter,
			map[string]any{"orderId": "O-1", "expectedStatus": expected},
		)
		if err != nil {
			t.Fatalf("expected-state update must not be an execution error: %v", err)
		}
		return execution.Result
	}

	if result := invoke("cancelled"); result.Mode != capabilityinvoke.ResultOutcome || result.Outcome != "conflict" {
		t.Fatalf("stale expected state was not a business conflict: %+v", result)
	}
	if status := independentStatus(); status != "new" {
		t.Fatalf("rejected update changed the row: %q", status)
	}

	if result := invoke("new"); result.Mode != capabilityinvoke.ResultOutcome || result.Outcome != "cancelled" {
		t.Fatalf("matching expected state was not applied: %+v", result)
	}
	if status := independentStatus(); status != "cancelled" {
		t.Fatalf("independent read did not observe the applied update: %q", status)
	}

	if result := invoke("new"); result.Outcome != "conflict" {
		t.Fatalf("replaying a consumed expected state must conflict: %+v", result)
	}
	if status := independentStatus(); status != "cancelled" {
		t.Fatalf("conflicting replay changed the row: %q", status)
	}
}

func compareAndSwapStatusSQLite(ctx context.Context, db *sql.DB, capability, orderID, expected string) (capabilityinvoke.BusinessResult, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx,
		`UPDATE order_status SET status = 'cancelled' WHERE order_id = ? AND status = ?`,
		orderID, expected)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	if affected == 1 {
		if err := tx.Commit(); err != nil {
			return capabilityinvoke.BusinessResult{}, err
		}
		return capabilityinvoke.BusinessResult{
			Capability: core.CanonicalID(capability),
			Mode:       capabilityinvoke.ResultOutcome,
			Outcome:    "cancelled",
			Payload:    map[string]any{"orderId": orderID, "status": "cancelled"},
		}, nil
	}
	return capabilityinvoke.BusinessResult{
		Capability: core.CanonicalID(capability),
		Mode:       capabilityinvoke.ResultOutcome,
		Outcome:    "conflict",
		Payload:    map[string]any{"orderId": orderID, "expectedStatus": expected},
	}, nil
}

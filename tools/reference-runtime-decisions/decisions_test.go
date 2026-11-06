// Package decisions checks the execution-time decisions against a real database.
// The tests skip unless TEI_TEST_POSTGRES_DSN is set; when it is, a temporary
// table records whether a refused decision left the data untouched.
package decisions

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"

	"tei/internal/core"
	"tei/internal/execution"
	"tei/internal/operations"
	"tei/internal/runlog"
	"tei/internal/semantic"
)

type memoryJournal struct {
	records []runlog.Record
}

func (j *memoryJournal) Append(record runlog.Record) error {
	j.records = append(j.records, record)
	return nil
}

func (j *memoryJournal) Last(key string) (runlog.Record, bool, error) {
	for i := len(j.records) - 1; i >= 0; i-- {
		if j.records[i].RunKey == key {
			return j.records[i], true, nil
		}
	}
	return runlog.Record{}, false, nil
}

// probe opens a connection and prepares a one-row table.
func probe(t *testing.T) (*pgx.Conn, func()) {
	t.Helper()
	dsn := os.Getenv("TEI_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEI_TEST_POSTGRES_DSN is not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if _, err := conn.Exec(ctx, "CREATE TEMPORARY TABLE tei_decisions_probe (id integer primary key, status text not null)"); err != nil {
		_ = conn.Close(ctx)
		t.Fatalf("create the probe table: %v", err)
	}
	if _, err := conn.Exec(ctx, "INSERT INTO tei_decisions_probe (id, status) VALUES (1, 'new')"); err != nil {
		_ = conn.Close(ctx)
		t.Fatalf("seed the probe row: %v", err)
	}
	return conn, func() { _ = conn.Close(ctx) }
}

func status(t *testing.T, conn *pgx.Conn) string {
	t.Helper()
	var value string
	if err := conn.QueryRow(context.Background(), "SELECT status FROM tei_decisions_probe WHERE id = 1").Scan(&value); err != nil {
		t.Fatalf("read the probe row: %v", err)
	}
	return value
}

func probeSurface() operations.OperationSurface {
	return operations.OperationSurface{Operations: []operations.Operation{{
		ID:            "app.create.sales.order",
		Kind:          "create",
		Target:        "sales.order",
		Binding:       "sales.close",
		Period:        "s.month",
		PeriodRejects: []string{"create"},
	}}}
}

func probePlan() semantic.RunPlan {
	return semantic.RunPlan{
		ID:         "sales.close_run",
		Operations: []core.CanonicalID{"sales.close"},
		Autonomy:   "unattended",
	}
}

// runPlan executes the plan and lets the step perform the update, so a refused
// decision leaves the row untouched in the database.
func runPlan(t *testing.T, conn *pgx.Conn, states map[string]operations.PeriodState) (decision runlog.Decision, results []execution.StepResult) {
	t.Helper()
	journal := &memoryJournal{}
	step := func(ctx context.Context, operation core.CanonicalID) execution.StepResult {
		if _, err := conn.Exec(ctx, "UPDATE tei_decisions_probe SET status = $1 WHERE id = 1", "closed"); err != nil {
			return execution.StepResult{Operation: string(operation), Status: "unresolved", Detail: err.Error()}
		}
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	preconditions := execution.CombinePreconditions(
		execution.PeriodPreconditions(probeSurface(), states),
	)
	decision, results, err := execution.RunPlan(context.Background(), probePlan(), []string{"2026-10"}, journal, step, preconditions)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return decision, results
}

func TestAClosedPeriodLeavesTheRowUntouched(t *testing.T) {
	conn, closeConn := probe(t)
	defer closeConn()

	decision, results := runPlan(t, conn, map[string]operations.PeriodState{"s.month": operations.PeriodClosed})
	if decision.Action != runlog.Start {
		t.Fatalf("the run must be identified: %+v", decision)
	}
	if len(results) != 1 || results[0].Status != "blocked" {
		t.Fatalf("the closed period must block the step: %+v", results)
	}
	if got := status(t, conn); got != "new" {
		t.Fatalf("a blocked run must not change the row: %q", got)
	}
}

func TestAnOpenPeriodUpdatesTheRow(t *testing.T) {
	conn, closeConn := probe(t)
	defer closeConn()

	_, results := runPlan(t, conn, map[string]operations.PeriodState{"s.month": operations.PeriodOpen})
	if len(results) != 1 || results[0].Status != "resolved" {
		t.Fatalf("the open period must let the step run: %+v", results)
	}
	if got := status(t, conn); got != "closed" {
		t.Fatalf("the row must change: %q", got)
	}
}

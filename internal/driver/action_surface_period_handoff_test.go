package driver_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/operations"
	"tei/internal/periodstate"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

// TestActionSurfacePeriodStateChangesAtDispatch uses production Surface,
// GuardCapabilityWith and a real JSON File source. The extra period meaning is
// deliberately test-only: this is NOT a new TEI DSL or accepted business Rule.
// It distinguishes old offer-time evidence from the current dispatch decision.
func TestActionSurfacePeriodStateChangesAtDispatch(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine fixture location")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	loaded, err := workspace.Load(filepath.Join(root, "reference", "standard-crud", "workspace"))
	if err != nil {
		t.Fatalf("load public fixture: %v", err)
	}

	foundNamespace := false
	for i := range loaded.World.Semantics {
		if loaded.World.Semantics[i].ID != "sales" {
			continue
		}
		foundNamespace = true
		loaded.World.Semantics[i].Periods = append(
			loaded.World.Semantics[i].Periods,
			semantic.Period{
				ID:            "sales.month",
				Data:          "sales.order",
				BusinessDate:  "sales.order.order_id",
				States:        []string{"open", "closing", "closed"},
				ClosedRejects: []string{"create", "update"},
			},
		)
	}
	if !foundNamespace {
		t.Fatal("sales namespace missing from negative fixture")
	}

	surface := operations.DeriveOperationSurface(loaded.World, "ledger")
	const intendedOperation = "ledger.update.sales.order"
	const binding = "sales.order.cancel"
	foundOperation := false
	for _, op := range surface.Operations {
		if op.ID != intendedOperation {
			continue
		}
		foundOperation = true
		if op.Binding != binding || op.Period != "sales.month" {
			t.Fatalf("wrong derived operation period/binding: %+v", op)
		}
	}
	if !foundOperation {
		t.Fatalf("derived operation %s missing", intendedOperation)
	}

	ctx := context.Background()
	statePath := filepath.Join(t.TempDir(), "period.json")
	source := periodstate.File{Path: statePath}
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(statePath, []byte(body), 0o600); err != nil {
			t.Fatalf("write local period state: %v", err)
		}
	}
	check := func(wantRefusal bool, wantErr bool) {
		t.Helper()
		rejection, refused, err := operations.GuardCapabilityWith(ctx, source, surface, "ledger", binding)
		if (err != nil) != wantErr {
			t.Fatalf("unexpected guard error: err=%v wantErr=%t", err, wantErr)
		}
		if wantErr {
			return
		}
		if refused != wantRefusal {
			t.Fatalf("refused=%t want=%t rejection=%+v", refused, wantRefusal, rejection)
		}
		if refused {
			if rejection.Operation != intendedOperation || rejection.Period != "sales.month" {
				t.Fatalf("wrong refusal target: %+v", rejection)
			}
		}
	}

	// An earlier offer may be valid. It cannot authorize later use.
	write(`{"sales.month":"open"}`)
	check(false, false)

	// The actual state is re-read at each guard invocation.
	write(`{"sales.month":"closing"}`)
	check(true, false)
	write(`{"sales.month":"closed"}`)
	check(true, false)

	// Missing current state is NOT evidence of 'open'; this particular
	// reference helper leaves the operation unguarded on unknown.
	write(`{}`)
	check(false, false)

	// A state for a different period does not establish current eligibility
	// of this period. In the current reference helper, omission is unguarded.
	write(`{"sales.week":"closed"}`)
	check(false, false)

	// Distinguish an omitted key in a readable source from a source that
	// disappears after the offer: an unreadable source must raise an error.
	if err := os.Remove(statePath); err != nil {
		t.Fatalf("remove current state file: %v", err)
	}
	check(false, true)

	// Invalid Source is an error, not a successful business refusal.
	write(`{"sales.month":"invalid"}`)
	check(false, true)

	// Source can become eligible again, but this cannot revive a revoked
	// execution Grant or certify that an already-started Commit is absent.
	write(`{"sales.month":"open"}`)
	check(false, false)
}

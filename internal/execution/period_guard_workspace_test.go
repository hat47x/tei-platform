package execution_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/execution"
	"tei/internal/operations"
	"tei/internal/workspace"
)

// This test goes from workspace text to a guard decision: the period is
// declared in the workspace, the surface derives the period and its rejects, and
// the guard refuses the bound capability while the period is closed.
func TestWorkspacePeriodRefusesTheBoundCapability(t *testing.T) {
	root := t.TempDir()
	write := func(path, content string) {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", "workspace: w\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n  - databases\n")
	write("semantic/s/x.tei", `namespace s

data Order {
    attribute order_date type std.date
    attribute status type std.code
}

capability s.order.register {
    input order s.order
    outcome created s.order
    effect create s.order
}

period s.month {
    data s.order
    business-date s.order.order_date
    states open closing closed
    closed-rejects create update
}
`)
	write("applications/app/application.tei", `application app {
    uses semantic s

    database main {
        target main_db
        access read-write
    }
}
`)
	write("databases/main-db/database.tei", "database main_db {\n    kind relational\n    ownership managed-here\n    uses semantic s\n}\n")
	write("databases/main-db/persistence/order.tei", `persist main_db.order {
    database main_db
    semantic s.order
    table public.orders

    column s.order.order_date -> order_date storage date required
    column s.order.status -> status storage varchar required

    primary-key s.order.order_date
}
`)

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	surface := operations.DeriveOperationSurface(loaded.World, "app")
	rejection, refused, err := operations.GuardCapabilityWith(
		context.Background(),
		fakeSource{states: map[string]operations.PeriodState{"s.month": operations.PeriodClosed}},
		surface,
		"app",
		core.MustCanonicalID("s.order.register"),
	)
	if err != nil {
		t.Fatalf("guard: %v", err)
	}
	if !refused {
		t.Fatalf("the closed period must refuse the capability: %+v / %+v", surface.Operations, surface.Gaps)
	}
	if rejection.Period != "s.month" || rejection.State != operations.PeriodClosed {
		t.Fatalf("unexpected rejection: %+v", rejection)
	}

	// The execution composition must not reach the driver either.
	invoker := &fakeInvoker{}
	_, refusal, err := execution.Invoke(context.Background(), invoker, execution.PeriodGuard{
		Surface: surface,
		Source:  fakeSource{states: map[string]operations.PeriodState{"s.month": operations.PeriodClosed}},
	}, "app", "s.order.register", nil, nil)
	if err != nil || refusal == nil {
		t.Fatalf("the composition must refuse: refusal=%+v err=%v", refusal, err)
	}
	if invoker.called {
		t.Fatal("a refused capability must not reach the driver")
	}
}

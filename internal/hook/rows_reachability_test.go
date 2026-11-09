package hook

import (
	"context"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

// recordRowScopeWorld declares a record pattern whose hook says scope row. The
// validator does not accept this shape (a record pattern looks at the record
// unit), so this world can only come from a hand-built fixture; the point is to
// pin down what RunRows does when it is asked for a row-scoped hook.
func recordRowScopeWorld() ir.World {
	return ir.World{Applications: []application.Application{{
		ID: "order_entry",
		Patterns: []application.Pattern{{
			ID:   "order_entry.order_record",
			Kind: application.PatternRecord,
			Data: "sales.order",
			Interaction: &application.Interaction{Hooks: []application.Hook{{
				Phase: "before-update",
				Scope: "row",
				Reads: []core.CanonicalID{"sales.order.amount"},
			}}},
		}},
	}}}
}

// browseRowScopeWorld declares a browse pattern whose hook says scope row. The
// validator accepts this shape, because a browse result is a row set.
func browseRowScopeWorld() ir.World {
	return ir.World{Applications: []application.Application{{
		ID: "order_entry",
		Patterns: []application.Pattern{{
			ID:   "order_entry.order_list",
			Kind: application.PatternBrowse,
			Data: "sales.order",
			Interaction: &application.Interaction{Hooks: []application.Hook{{
				Phase: "before-read",
				Scope: "row",
				Reads: []core.CanonicalID{"sales.order.amount"},
			}}},
		}},
	}}}
}

// rowCaller records the values each row handed to the hook, so a test can check
// that the row set actually reached the implementation.
type rowCaller struct {
	seen []map[core.CanonicalID]string
}

func (c *rowCaller) Run(_ context.Context, invocation Invocation) error {
	c.seen = append(c.seen, invocation.Reads)
	return nil
}

// TestRunRowsReachesABrowsePattern checks the gap this test exists for.
//
// The validator lets a browse pattern declare a row-scoped hook, because a
// browse result is a row set, and RunRows is the method that applies one hook
// per row. Before this, RunRows resolved only record patterns, so the shape the
// validator accepts could not be run by the method the row scope points at: a
// browse hook declaring `scope row` validated cleanly and then could never run.
func TestRunRowsReachesABrowsePattern(t *testing.T) {
	implementation := &rowCaller{}
	runner := Runner{World: browseRowScopeWorld(), Implementation: implementation}
	err := runner.RunRows(context.Background(), Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_list",
		Phase:       "before-read",
	}, []map[core.CanonicalID]string{
		{"sales.order.amount": "10", "sales.order.unrelated": "x"},
		{"sales.order.amount": "20"},
	})
	if err != nil {
		t.Fatalf("a browse pattern was refused by RunRows: %v", err)
	}
	if len(implementation.seen) != 2 {
		t.Fatalf("the hook ran %d times, want 2", len(implementation.seen))
	}
	if implementation.seen[0]["sales.order.amount"] != "10" || implementation.seen[1]["sales.order.amount"] != "20" {
		t.Fatalf("rows=%v", implementation.seen)
	}
	// The row is narrowed to the declared reads, so an unrelated attribute of
	// the row never reaches the hook.
	if _, leaked := implementation.seen[0]["sales.order.unrelated"]; leaked {
		t.Fatalf("an undeclared attribute reached the hook: %v", implementation.seen[0])
	}
}

// TestRunRowsReachesARecordPatternOnlyWhenTheScopeIsRow checks that the record
// shape RunRows accepts is exactly the one the validator refuses.
func TestRunRowsReachesARecordPatternOnlyWhenTheScopeIsRow(t *testing.T) {
	runner := Runner{World: recordRowScopeWorld(), Implementation: ReferenceRule{}}
	err := runner.RunRows(context.Background(), Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-update",
	}, []map[core.CanonicalID]string{{"sales.order.amount": "10"}})
	if err != nil {
		t.Fatalf("a record pattern with scope row was refused: %v", err)
	}
}

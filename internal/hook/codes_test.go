package hook_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
)

// Each refusal in this package names the reason a hook did not run. A caller
// branches on those names, so every code needs an input that produces it
// (#4032).
func TestHookCodesAreProduced(t *testing.T) {
	failingImplementation := funcCodeImplementation{err: errors.New("the fixture refused")}

	t.Run("caller unavailable", func(t *testing.T) {
		guarded := hook.Guarded{Rule: hook.ReferenceRule{ID: "fixture", Required: []core.CanonicalID{"sales.order.quantity"}}}
		err := guarded.Run(context.Background(), hook.Invocation{Reads: map[core.CanonicalID]string{"sales.order.quantity": "1"}})
		assertHookCode(t, err, "HOOK_CALLER_UNAVAILABLE")
	})

	t.Run("capability not configured", func(t *testing.T) {
		guarded := hook.Guarded{Rule: hook.ReferenceRule{ID: "fixture", Required: []core.CanonicalID{"sales.order.quantity"}}}
		err := guarded.RunWithCaller(context.Background(),
			hook.Invocation{Reads: map[core.CanonicalID]string{"sales.order.quantity": "1"}}, &inputCaller{})
		assertHookCode(t, err, "HOOK_CAPABILITY_NOT_CONFIGURED")
	})

	t.Run("pattern not found", func(t *testing.T) {
		runner := hook.Runner{World: rowScopeWorld("row"), Implementation: &rowImplementation{}}
		err := runner.RunRows(context.Background(), hook.Invocation{
			Application: "order_entry",
			Pattern:     "order_entry.missing",
			Phase:       "before-update",
		}, rowsOf("1"))
		assertHookCode(t, err, "HOOK_PATTERN_NOT_FOUND")
	})

	t.Run("pattern not interactive", func(t *testing.T) {
		world := ir.World{Applications: []application.Application{{
			ID:       "order_entry",
			Patterns: []application.Pattern{{ID: "order_entry.order_record", Kind: application.PatternRecord, Data: "sales.order"}},
		}}}
		runner := hook.Runner{World: world, Implementation: &rowImplementation{}}
		err := runner.RunRows(context.Background(), hook.Invocation{
			Application: "order_entry",
			Pattern:     "order_entry.order_record",
			Phase:       "before-update",
		}, rowsOf("1"))
		assertHookCode(t, err, "HOOK_PATTERN_NOT_INTERACTIVE")
	})

	t.Run("phase unsupported", func(t *testing.T) {
		world := ir.World{Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
				Interaction: &application.Interaction{Hooks: []application.Hook{
					{Phase: "during-update", Reads: []core.CanonicalID{"sales.order.amount"}},
				}},
			}},
		}}}
		runner := hook.Runner{World: world, Implementation: &rowImplementation{}}
		_, err := runner.RunSequence(context.Background(), hook.Invocation{
			Application: "order_entry",
			Pattern:     "order_entry.order_record",
			Phase:       "before-update",
		})
		assertHookCode(t, err, "HOOK_PHASE_UNSUPPORTED")
	})

	t.Run("row failed", func(t *testing.T) {
		runner := hook.Runner{World: rowScopeWorld("row"), Implementation: failingImplementation}
		err := runner.RunRows(context.Background(), hook.Invocation{
			Application: "order_entry",
			Pattern:     "order_entry.order_record",
			Phase:       "before-update",
		}, rowsOf("1"))
		assertHookCode(t, err, "HOOK_ROW_FAILED")
	})

	t.Run("scope not row", func(t *testing.T) {
		runner := hook.Runner{World: rowScopeWorld("record"), Implementation: &rowImplementation{}}
		err := runner.RunRows(context.Background(), hook.Invocation{
			Application: "order_entry",
			Pattern:     "order_entry.order_record",
			Phase:       "before-update",
		}, rowsOf("1"))
		assertHookCode(t, err, "HOOK_SCOPE_NOT_ROW")
	})
}

// funcCodeImplementation always fails with a plain error, which is neither a
// business refusal nor a configured outcome.
type funcCodeImplementation struct {
	err error
}

func (i funcCodeImplementation) Run(_ context.Context, _ hook.Invocation) error {
	return i.err
}

func assertHookCode(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s, got no error", want)
	}
	var hookErr *hook.Error
	if !errors.As(err, &hookErr) {
		t.Fatalf("expected a hook error with code %s, got %vh", want, err)
	}
	if hookErr.Code != want {
		t.Fatalf("expected code %s, got %s (%v)", want, hookErr.Code, err)
	}
}

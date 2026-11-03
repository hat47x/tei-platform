package hook

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

func hookWorld(failure string) ir.World {
	return ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
				Interaction: &application.Interaction{
					Hooks: []application.Hook{{
						Phase:    "before-update",
						Reads:    []core.CanonicalID{"sales.order.status"},
						Uses:     []core.CanonicalID{"sales.submit"},
						Produces: []core.CanonicalID{"sales.order.customer_id"},
						Failure:  failure,
					}},
				},
			}},
		}},
	}
}

type fakeImplementation struct {
	err error
}

func (f fakeImplementation) Run(context.Context, Invocation) error { return f.err }

func invocation() Invocation {
	return Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-update",
		Reads:       map[core.CanonicalID]string{"sales.order.status": "open"},
	}
}

func hookErrorCode(t *testing.T, err error) string {
	t.Helper()
	var hookError *Error
	if !errors.As(err, &hookError) {
		t.Fatalf("error is not hook.Error: %T %v", err, err)
	}
	return hookError.Code
}

func TestRunInvokesDeclaredHook(t *testing.T) {
	runner := Runner{World: hookWorld("reject"), Implementation: fakeImplementation{}}
	if err := runner.Run(context.Background(), invocation()); err != nil {
		t.Fatal(err)
	}
}

func TestRunRejectsMissingHook(t *testing.T) {
	runner := Runner{World: hookWorld("reject"), Implementation: fakeImplementation{}}
	missing := invocation()
	missing.Phase = "after-update"
	if code := hookErrorCode(t, runner.Run(context.Background(), missing)); code != "HOOK_NOT_DECLARED" {
		t.Fatalf("code=%s", code)
	}
}

func TestRunRequiresImplementation(t *testing.T) {
	runner := Runner{World: hookWorld("reject")}
	if code := hookErrorCode(t, runner.Run(context.Background(), invocation())); code != "HOOK_IMPLEMENTATION_UNAVAILABLE" {
		t.Fatalf("code=%s", code)
	}
}

func TestRunRejectsOnFailure(t *testing.T) {
	runner := Runner{World: hookWorld("reject"), Implementation: fakeImplementation{err: errors.New("invalid status")}}
	if code := hookErrorCode(t, runner.Run(context.Background(), invocation())); code != "HOOK_REJECTED" {
		t.Fatalf("code=%s", code)
	}
}

func TestRunWarnsOnFailure(t *testing.T) {
	runner := Runner{World: hookWorld("warn"), Implementation: fakeImplementation{err: errors.New("invalid status")}}
	err := runner.Run(context.Background(), invocation())
	var warning *Warning
	if !errors.As(err, &warning) {
		t.Fatalf("error is not hook.Warning: %T %v", err, err)
	}
}

func TestRunRejectsNonRecordPattern(t *testing.T) {
	world := hookWorld("reject")
	world.Applications[0].Patterns[0].Kind = application.PatternBrowse
	runner := Runner{World: world, Implementation: fakeImplementation{}}
	if code := hookErrorCode(t, runner.Run(context.Background(), invocation())); code != "HOOK_PATTERN_NOT_RECORD" {
		t.Fatalf("code=%s", code)
	}
}

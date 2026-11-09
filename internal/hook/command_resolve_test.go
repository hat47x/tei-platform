package hook_test

import (
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
)

// resolveWorld declares a record pattern and the command patterns that the
// lookup resolves, so the test exercises the actual application model.
func resolveWorld(commands ...application.Pattern) ir.World {
	patterns := []application.Pattern{{
		ID:   "order_entry.order_record",
		Kind: application.PatternRecord,
		Data: "sales.order",
	}}
	patterns = append(patterns, commands...)
	return ir.World{
		Applications: []application.Application{{
			ID:       "order_entry",
			Patterns: patterns,
		}},
	}
}

func commandPattern(id string, capability core.CanonicalID) application.Pattern {
	return application.Pattern{ID: core.CanonicalID(id), Kind: application.PatternCommand, Capability: capability}
}

// TestCommandPatternForFindsTheDeclaringPattern checks that one command pattern
// resolving the capability is returned.
func TestCommandPatternForFindsTheDeclaringPattern(t *testing.T) {
	world := resolveWorld(commandPattern("order_entry.submit_order", "sales.submit_order"))
	pattern, err := hook.CommandPatternFor(world, "order_entry", "sales.submit_order")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if pattern != "order_entry.submit_order" {
		t.Fatalf("pattern=%s", pattern)
	}
}

// TestCommandPatternForRefusesWithoutADeclaration checks that a capability no
// command pattern declares is refused instead of guessed.
func TestCommandPatternForRefusesWithoutADeclaration(t *testing.T) {
	world := resolveWorld(commandPattern("order_entry.submit_order", "sales.submit_order"))
	_, err := hook.CommandPatternFor(world, "order_entry", "sales.delete_order")
	var hookError *hook.Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_PATTERN_NOT_DECLARED" {
		t.Fatalf("err=%v", err)
	}
}

// TestCommandPatternForRefusesAmbiguity checks that two command patterns
// declaring the same capability are refused, so the caller cannot pick one by
// declaration order.
func TestCommandPatternForRefusesAmbiguity(t *testing.T) {
	world := resolveWorld(
		commandPattern("order_entry.submit_order", "sales.submit_order"),
		commandPattern("order_entry.submit_order_again", "sales.submit_order"),
	)
	_, err := hook.CommandPatternFor(world, "order_entry", "sales.submit_order")
	var hookError *hook.Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_PATTERN_AMBIGUOUS" {
		t.Fatalf("err=%v", err)
	}
}

// TestCommandPatternForIgnoresNonCommandPatterns checks that a record pattern
// naming the capability is not treated as a command pattern.
func TestCommandPatternForIgnoresNonCommandPatterns(t *testing.T) {
	record := application.Pattern{ID: "order_entry.order_record", Kind: application.PatternRecord, Capability: "sales.submit_order"}
	world := resolveWorld(record)
	_, err := hook.CommandPatternFor(world, "order_entry", "sales.submit_order")
	var hookError *hook.Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_PATTERN_NOT_DECLARED" {
		t.Fatalf("err=%v", err)
	}
}

// TestCommandPatternForNeedsAnApplication checks that a lookup without an
// application is reported, not resolved against every application.
func TestCommandPatternForNeedsAnApplication(t *testing.T) {
	world := resolveWorld(commandPattern("order_entry.submit_order", "sales.submit_order"))
	_, err := hook.CommandPatternFor(world, "", "sales.submit_order")
	var hookError *hook.Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_APPLICATION_REQUIRED" {
		t.Fatalf("err=%v", err)
	}
}

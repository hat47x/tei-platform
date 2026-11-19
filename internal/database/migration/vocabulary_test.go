package migration

import (
	"slices"
	"testing"
)

// Tool is the migration runner a database release may name. A reader selects a
// runner by this value, so the set is part of the contract (#4032).
func TestMigrationToolsAreCovered(t *testing.T) {
	tools := []Tool{ToolNative, ToolAtlas, ToolFlyway, ToolLiquibase, ToolGolangMigrate}
	want := []string{"native", "atlas", "flyway", "liquibase", "golang-migrate"}
	if len(tools) != len(want) {
		t.Fatalf("a tool was dropped: %d values", len(tools))
	}
	seen := map[string]bool{}
	for i, tool := range tools {
		if string(tool) != want[i] {
			t.Fatalf("tools[%d]=%q want %q", i, tool, want[i])
		}
		if seen[string(tool)] {
			t.Fatalf("tool %q is declared twice", tool)
		}
		seen[string(tool)] = true
	}
}

// RecoveryStrategy and StepKind are validated by a closed list, so a value that
// no test names could be added to the list without anything showing that it is
// accepted or refused. Each value is listed here (#4032).
func TestMigrationRecoveryAndStepKindsAreCovered(t *testing.T) {
	t.Run("recovery strategies", func(t *testing.T) {
		strategies := []RecoveryStrategy{RecoveryForwardFix, RecoveryExplicitRollback, RecoveryNotReversible}
		want := []string{"forward-fix", "explicit-rollback", "not-reversible"}
		if len(strategies) != len(want) {
			t.Fatalf("a strategy was dropped: %d values", len(strategies))
		}
		for i, strategy := range strategies {
			if string(strategy) != want[i] {
				t.Fatalf("strategies[%d]=%q want %q", i, strategy, want[i])
			}
			if !strategy.Valid() {
				t.Fatalf("%q is declared but not accepted by Valid", strategy)
			}
		}
		if !slices.Equal(RecoveryStrategies, strategies) {
			t.Fatalf("RecoveryStrategies=%v, want %v", RecoveryStrategies, strategies)
		}
		for _, bad := range []RecoveryStrategy{"", "ForwardFix", "rollback", "shadow-write"} {
			if bad.Valid() {
				t.Fatalf("an undeclared recovery strategy %q was accepted", bad)
			}
		}
	})

	t.Run("step kinds", func(t *testing.T) {
		kinds := []StepKind{StepSchema, StepData, StepVerify}
		want := []string{"schema", "data", "verify"}
		if len(kinds) != len(want) {
			t.Fatalf("a step kind was dropped: %d values", len(kinds))
		}
		for i, kind := range kinds {
			if string(kind) != want[i] {
				t.Fatalf("kinds[%d]=%q want %q", i, kind, want[i])
			}
			if !kind.Valid() {
				t.Fatalf("%q is declared but not accepted by Valid", kind)
			}
		}
		if !slices.Equal(StepKinds, kinds) {
			t.Fatalf("StepKinds=%v, want %v", StepKinds, kinds)
		}
		for _, bad := range []StepKind{"", "Schema", "migrate", "rollback"} {
			if bad.Valid() {
				t.Fatalf("an undeclared step kind %q was accepted", bad)
			}
		}
	})
}

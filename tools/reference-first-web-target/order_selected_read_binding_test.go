//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"strings"
	"testing"
)

// A test double with a nonzero-sized identity. Its marker stands for a
// distinct provider, not a claim that the test accessed PostgreSQL.
type selectedReadBindingTestRunner struct {
	provider string
}

func (r *selectedReadBindingTestRunner) Run(context.Context, readScenarioInput) (readScenarioResult, error) {
	return readScenarioResult{ScenarioID: r.provider}, nil
}

func TestCommandAlwaysUsesTheReadSelectedForTheWeb(t *testing.T) {
	// Even when the Host does not inject a Browse executor, a separately
	// selected header-detail provider must be used by both HTTP and Submit.
	webRead := &selectedReadBindingTestRunner{provider: "selected-postgres"}
	oldFixture := &selectedReadBindingTestRunner{provider: "synthetic-fixture"}
	command := &referenceCommandScenarioRunner{read: oldFixture}
	if err := bindCommandToSelectedRead(command, webRead, nil); err != nil {
		t.Fatal(err)
	}
	if command.read != readScenarioRunner(webRead) {
		t.Fatal("Submit still uses a different read source from the Web")
	}
	got, err := command.read.Run(context.Background(), readScenarioInput{})
	if err != nil || got.ScenarioID != "selected-postgres" {
		t.Fatalf("Command uses %q, error=%v; wanted selected reader", got.ScenarioID, err)
	}
}

func TestCommandReadWrapperReceivesTheSelectedWebRead(t *testing.T) {
	webRead := &selectedReadBindingTestRunner{provider: "selected-postgres"}
	wrapped := &selectedReadBindingTestRunner{provider: "test-adapter"}
	command := &referenceCommandScenarioRunner{}
	var seen readScenarioRunner
	err := bindCommandToSelectedRead(command, webRead, func(input readScenarioRunner) readScenarioRunner {
		seen = input
		return wrapped
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen != readScenarioRunner(webRead) {
		t.Fatal("wrapper received the constructor's synthetic reader, not the selected Web reader")
	}
	if command.read != readScenarioRunner(wrapped) {
		t.Fatal("Command did not use the wrapper of the selected Web reader")
	}
}

func TestCommandReadBindingRefusesMissingOrDiscardedSelection(t *testing.T) {
	oldFixture := &selectedReadBindingTestRunner{provider: "synthetic-fixture"}
	webRead := &selectedReadBindingTestRunner{provider: "selected-postgres"}
	command := &referenceCommandScenarioRunner{read: oldFixture}
	cases := []struct {
		name     string
		command  *referenceCommandScenarioRunner
		selected readScenarioRunner
		wrap     func(readScenarioRunner) readScenarioRunner
		want     string
	}{
		{name: "no selected reader", command: command, want: "required"},
		{name: "no command", selected: webRead, want: "required"},
		{name: "wrapper discards selected reader", command: command, selected: webRead, wrap: func(readScenarioRunner) readScenarioRunner { return nil }, want: "removed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := bindCommandToSelectedRead(tc.command, tc.selected, tc.wrap)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected refusal containing %q, got %v", tc.want, err)
			}
			if command.read != readScenarioRunner(oldFixture) {
				t.Fatal("failed binding replaced the previous reader")
			}
		})
	}
}

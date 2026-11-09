package main

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/hook"
)

// refusedReport builds a hook report whose refusal named the given outcome.
func refusedReport(outcome string) cliHookReport {
	return cliHookReport{
		PhaseReport: &hook.PhaseReport{
			Refused: "before-update",
			Outcome: outcome,
		},
		Pattern: "order_entry.order_record",
		Note:    cliHookNote,
	}
}

// TestHookReportNamesTheCommandOutcomeItCorrespondsTo checks that a hook
// refusal naming a Command-side outcome is reported with the Command name, so a
// caller can branch on it without knowing the hook vocabulary.
func TestHookReportNamesTheCommandOutcomeItCorrespondsTo(t *testing.T) {
	report := refusedReport(hook.OutcomeValueRejected)
	if name, ok := hook.CommandOutcomeFor(report.Outcome); !ok {
		t.Fatalf("%s has no Command counterpart", report.Outcome)
	} else {
		report.CommandOutcome = name
	}
	var out bytes.Buffer
	if _, err := writeHookReportText(&out, report); err != nil {
		t.Fatalf("err=%v", err)
	}
	got := out.String()
	if !strings.Contains(got, "refused") {
		t.Fatalf("the refusal was not printed:\n%s", got)
	}
	if !strings.Contains(got, "command outcome value-rejected") {
		t.Fatalf("the correspondence was not printed:\n%s", got)
	}
}

// TestHookReportKeepsAnOutcomeWithNoCommandCounterpartDistinct checks that
// `not-satisfied` is not presented as a Command refusal reason: the hook's
// judgement is not the Command's.
func TestHookReportKeepsAnOutcomeWithNoCommandCounterpartDistinct(t *testing.T) {
	report := refusedReport(hook.OutcomeUnmet)
	if name, ok := hook.CommandOutcomeFor(report.Outcome); ok {
		report.CommandOutcome = name
	}
	var out bytes.Buffer
	if _, err := writeHookReportText(&out, report); err != nil {
		t.Fatalf("err=%v", err)
	}
	got := out.String()
	if strings.Contains(got, "command outcome") {
		t.Fatalf("an outcome with no Command counterpart was mapped:\n%s", got)
	}
	if !strings.Contains(got, hook.OutcomeUnmet) {
		t.Fatalf("the hook's own outcome was dropped:\n%s", got)
	}
}

package main

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/hook"
)

// TestWriteHookReportTextNamesProposals checks that a value a hook proposed is
// printed beside the phase, and labelled as a candidate rather than as an
// applied value.
func TestWriteHookReportTextNamesProposals(t *testing.T) {
	var out bytes.Buffer
	report := cliHookReport{
		PhaseReport: &hook.PhaseReport{
			Executed: []string{"before-update"},
			Produced: map[string]hook.Produced{
				"before-update": {"sales.order.amount": "2000"},
			},
		},
		Pattern: "order_entry.order_record",
		Note:    cliHookNote,
	}
	if code, err := writeHookReportText(&out, report); err != nil {
		t.Fatalf("code=%d err=%v", code, err)
	}
	got := out.String()
	if !strings.Contains(got, "hook before-update: executed") {
		t.Fatalf("the phase was not printed:\n%s", got)
	}
	if !strings.Contains(got, "proposes sales.order.amount = 2000 (candidate, not adopted)") {
		t.Fatalf("the proposal was not printed:\n%s", got)
	}
}

// TestWriteHookReportTextPrintsProposalsInAttributeOrder checks that two
// proposals print in a fixed order, so the same run prints the same bytes.
func TestWriteHookReportTextPrintsProposalsInAttributeOrder(t *testing.T) {
	render := func() string {
		var out bytes.Buffer
		report := cliHookReport{
			PhaseReport: &hook.PhaseReport{
				Executed: []string{"before-update"},
				Produced: map[string]hook.Produced{
					"before-update": {
						"sales.order.total":    "3000",
						"sales.order.amount":   "2000",
						"sales.order.discount": "0",
					},
				},
			},
			Pattern: "order_entry.order_record",
			Note:    cliHookNote,
		}
		if _, err := writeHookReportText(&out, report); err != nil {
			t.Fatalf("err=%v", err)
		}
		return out.String()
	}
	first := render()
	second := render()
	if first != second {
		t.Fatalf("the same report printed differently:\n%s\n---\n%s", first, second)
	}
	amount := strings.Index(first, "sales.order.amount")
	discount := strings.Index(first, "sales.order.discount")
	total := strings.Index(first, "sales.order.total")
	if !(amount < discount && discount < total) {
		t.Fatalf("the proposals are not in attribute order:\n%s", first)
	}
}

// TestWriteHookReportTextOmitsProposalsForAReadOnlyPhase checks that a phase
// which proposed nothing prints no proposal line.
func TestWriteHookReportTextOmitsProposalsForAReadOnlyPhase(t *testing.T) {
	var out bytes.Buffer
	report := cliHookReport{
		PhaseReport: &hook.PhaseReport{Executed: []string{"before-read"}},
		Pattern:     "order_entry.order_record",
		Note:        cliHookNote,
	}
	if _, err := writeHookReportText(&out, report); err != nil {
		t.Fatalf("err=%v", err)
	}
	if strings.Contains(out.String(), "proposes") {
		t.Fatalf("a read-only phase claimed a proposal:\n%s", out.String())
	}
}

// TestWriteHookReportTextNamesAnUncalledDeclaration checks that a hook's
// declared `uses` which never ran is printed, so "not evaluated" is not read as
// "the hook ran but did nothing".
func TestWriteHookReportTextNamesAnUncalledDeclaration(t *testing.T) {
	var out bytes.Buffer
	report := cliHookReport{
		PhaseReport: &hook.PhaseReport{
			NotEvaluated: []string{"before-update"},
			Reasons:      map[string]string{"before-update": hook.ReasonCallerUnavailable},
			Uncalled: map[string][]core.CanonicalID{
				"before-update": {"sales.submit_order"},
			},
		},
		Pattern: "order_entry.order_record",
		Note:    cliHookNote,
	}
	if _, err := writeHookReportText(&out, report); err != nil {
		t.Fatalf("err=%v", err)
	}
	got := out.String()
	if !strings.Contains(got, "hook before-update: not evaluated") {
		t.Fatalf("the phase was not printed as not evaluated:\n%s", got)
	}
	if !strings.Contains(got, "uncalled sales.submit_order") {
		t.Fatalf("the declared call was not named:\n%s", got)
	}
}

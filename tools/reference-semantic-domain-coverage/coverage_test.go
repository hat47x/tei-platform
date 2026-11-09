package semanticdomaincoverage

import (
	"path/filepath"
	"reflect"
	"testing"
)

func loadFixture(t *testing.T, name string) Input {
	t.Helper()
	input, err := LoadInput(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("LoadInput(%s): %v", name, err)
	}
	return input
}

func byID(report Report) map[string]ElementOutcome {
	out := make(map[string]ElementOutcome, len(report.Elements))
	for _, outcome := range report.Elements {
		out[outcome.ID] = outcome
	}
	return out
}

func TestUntypedListsElementsWithoutAcceptedDomain(t *testing.T) {
	report := Check(loadFixture(t, "input.json"))
	got := byID(report)

	wantUntyped := []string{
		"sales.customer.name",
		"sales.order.due-date",
		"sales.order.line",
		"sales.order.status",
		"sales.order.tax",
		"sales.order.total-with-tax",
	}
	for _, id := range wantUntyped {
		if got[id].Status != ElementUntyped {
			t.Errorf("%s: status = %q, want %q", id, got[id].Status, ElementUntyped)
		}
	}
}

func TestAmbiguousElementReportsBothDomains(t *testing.T) {
	report := Check(loadFixture(t, "input.json"))
	if report.Status != StatusWarning {
		t.Fatalf("status = %q, want %q", report.Status, StatusWarning)
	}
	outcome := byID(report)["sales.order.total"]
	if outcome.Status != ElementAmbiguous {
		t.Fatalf("sales.order.total: status = %q, want %q", outcome.Status, ElementAmbiguous)
	}
	want := []string{"domain.money", "domain.money-total"}
	if !reflect.DeepEqual(outcome.Domains, want) {
		t.Errorf("domains = %v, want %v", outcome.Domains, want)
	}
}

func TestProposedDomainIsNotTreatedAsDefinition(t *testing.T) {
	input := loadFixture(t, "input.json")
	report := Check(input)
	if got := byID(report)["sales.order.status"]; got.Status != ElementUntyped {
		t.Errorf("sales.order.status: status = %q, want %q because its only domain is a proposal", got.Status, ElementUntyped)
	}
	if report.AcceptedDomains != 4 {
		t.Errorf("acceptedDomains = %d, want 4", report.AcceptedDomains)
	}
	if report.ProposedDomains != 1 {
		t.Errorf("proposedDomains = %d, want 1", report.ProposedDomains)
	}
	if got := UnacceptedDomains(input); !reflect.DeepEqual(got, []string{"domain.order-status"}) {
		t.Errorf("UnacceptedDomains = %v", got)
	}
}

func TestAddingDomainOnlyAffectsItsElements(t *testing.T) {
	before := byID(Check(loadFixture(t, "input.json")))
	afterReport := Check(loadFixture(t, "input-added.json"))
	after := byID(afterReport)

	changed := map[string]bool{
		"sales.order.tax":            true,
		"sales.order.total-with-tax": true,
	}
	for id, outcome := range before {
		if changed[id] {
			if after[id].Status != ElementTyped {
				t.Errorf("%s should become typed, got %q", id, after[id].Status)
			}
			continue
		}
		if !reflect.DeepEqual(outcome, after[id]) {
			t.Errorf("%s changed although the new domain does not name it: before=%+v after=%+v", id, outcome, after[id])
		}
	}
	if len(after) != len(before) {
		t.Errorf("element count changed: before=%d after=%d", len(before), len(after))
	}
}

func TestWhatWhySeparatesImplementationFromHumanMeaning(t *testing.T) {
	report := Check(loadFixture(t, "input.json"))
	want := map[string]WhatWhy{
		"sales.order.total":          {Element: "sales.order.total", What: true, Why: true},
		"sales.order.tax":            {Element: "sales.order.tax", What: true, Why: false},
		"sales.order.total-with-tax": {Element: "sales.order.total-with-tax", What: true, Why: false},
	}
	got := make(map[string]WhatWhy, len(report.WhatWhy))
	for _, entry := range report.WhatWhy {
		got[entry.Element] = entry
	}
	for id, wantEntry := range want {
		if got[id] != wantEntry {
			t.Errorf("%s: WhatWhy = %+v, want %+v", id, got[id], wantEntry)
		}
	}
	if len(got) != len(report.Elements) {
		t.Errorf("WhatWhy entries = %d, elements = %d", len(got), len(report.Elements))
	}
}

func TestDuplicateElementIDIsRejected(t *testing.T) {
	input := Input{
		Schema: InputSchema,
		Elements: []Element{
			{ID: "a", Name: "a"},
			{ID: "a", Name: "again"},
		},
	}
	report := Check(input)
	if report.Status != StatusRejected {
		t.Fatalf("status = %q, want %q", report.Status, StatusRejected)
	}
	if len(report.Problems) == 0 {
		t.Fatal("expected a problem describing the duplicate id")
	}
}

func TestUnknownElementReferenceIsRejected(t *testing.T) {
	input := Input{
		Schema:   InputSchema,
		Elements: []Element{{ID: "a", Name: "a"}},
		Domains: []Domain{{
			ID:       "d",
			Name:     "d",
			Elements: []string{"missing"},
			Accepted: true,
		}},
	}
	report := Check(input)
	if report.Status != StatusRejected {
		t.Fatalf("status = %q, want %q", report.Status, StatusRejected)
	}
}

func TestCheckDoesNotChangeInput(t *testing.T) {
	input := loadFixture(t, "input.json")
	original := loadFixture(t, "input.json")
	_ = Check(input)
	if !reflect.DeepEqual(input, original) {
		t.Errorf("Check changed the input: before=%+v after=%+v", original, input)
	}
}

func TestCleanInputPasses(t *testing.T) {
	input := Input{
		Schema: InputSchema,
		Elements: []Element{
			{ID: "a", Name: "a", What: "input", Why: "意味"},
			{ID: "b", Name: "b", What: "derived", Why: "意味"},
		},
		Domains: []Domain{{
			ID:       "d",
			Name:     "d",
			Elements: []string{"a", "b"},
			Accepted: true,
		}},
	}
	report := Check(input)
	if report.Status != StatusPassed {
		t.Fatalf("status = %q, want %q; problems=%v", report.Status, StatusPassed, report.Problems)
	}
}

package rulecandidate

import "testing"

func sample() []Example {
	return []Example{
		{ID: "e1", Input: "ABC-001", Output: "ABC"},
		{ID: "e2", Input: "XYZ-025", Output: "XYZ"},
		{ID: "e3", Input: "DEF-100", Output: "DEF"},
	}
}

func TestAnalyzeDoesNotConfirmASingleRule(t *testing.T) {
	r := Analyze(sample())
	if !r.ReviewRequired {
		t.Fatal("ReviewRequired must always be true")
	}
	if !r.Ambiguous {
		t.Fatalf("3 examples admit several rules, but Ambiguous=%v", r.Ambiguous)
	}
	if !hasFinding(r.Findings, "multiple-hypotheses-fit") {
		t.Errorf("missing multiple-hypotheses-fit in %+v", r.Findings)
	}
	general := 0
	for _, h := range r.Hypotheses {
		if h.FitsAll && h.Generalizable {
			general++
		}
	}
	if general < 2 {
		t.Fatalf("expected at least two generalizable fitting hypotheses, got %d", general)
	}
}

func TestAnalyzeAlwaysSeparatesFormatFromMeaning(t *testing.T) {
	for _, h := range Analyze(sample()).Hypotheses {
		if !h.FormatOnly {
			t.Errorf("hypothesis %q is not marked format-only", h.Kind)
		}
	}
	if !hasFinding(Analyze(sample()).Findings, "format-only-not-business-meaning") {
		t.Errorf("format-only-not-business-meaning missing")
	}
}

func TestAnalyzeProposesBoundaryInputsThatDisagree(t *testing.T) {
	r := Analyze(sample())
	if len(r.Probes) == 0 {
		t.Fatal("no boundary inputs proposed")
	}
	disagree := false
	for _, p := range r.Probes {
		if p.Disagree {
			disagree = true
		}
	}
	if !disagree {
		t.Fatalf("no proposed boundary input separates the fitting hypotheses: %+v", r.Probes)
	}
	for _, p := range r.Probes {
		if len(p.Outputs) < 2 {
			continue
		}
		seen := map[string]bool{}
		for _, o := range p.Outputs {
			seen[o.Output] = true
		}
		if p.Disagree != (len(seen) > 1) {
			t.Errorf("probe %q Disagree=%v but outputs=%+v", p.Input, p.Disagree, p.Outputs)
		}
	}
}

func TestAnalyzeRequiresMoreExamplesWhenFew(t *testing.T) {
	r := Analyze(sample()[:2])
	if !hasFinding(r.Findings, "insufficient-examples") {
		t.Fatalf("missing insufficient-examples in %+v", r.Findings)
	}
}

func TestAnalyzeIsReproducibleAndStable(t *testing.T) {
	a := Analyze(sample())
	b := Analyze(sample())
	if a.Fingerprint != b.Fingerprint {
		t.Fatalf("fingerprint is not reproducible: %s vs %s", a.Fingerprint, b.Fingerprint)
	}
	if a.Examples[0].ID != "e1" || a.Examples[2].ID != "e3" {
		t.Fatalf("examples are not in a stable order: %+v", a.Examples)
	}
	changed := sample()
	changed[2].Output = "DEFG"
	if Analyze(changed).Fingerprint == a.Fingerprint {
		t.Fatal("a changed example did not change the fingerprint")
	}
}

func TestConfirmRejectsBusinessAndAuthorizationClaims(t *testing.T) {
	sel := Selection{Kind: PrefixBeforeFirstDelimiter, ExampleIDs: []string{"e1", "e2", "e3"}}
	for _, tc := range []struct {
		claim string
		code  string
	}{
		{"business-identity", "format-cannot-establish-business-identity"},
		{"authorization", "format-cannot-establish-authorization"},
	} {
		got, findings := Confirm(sel, sample(), tc.claim)
		if got.Selected != "" {
			t.Errorf("claim %q was accepted: %+v", tc.claim, got)
		}
		if !hasFinding(findings, tc.code) {
			t.Errorf("claim %q missing finding %q in %+v", tc.claim, tc.code, findings)
		}
	}
}

func TestConfirmAcceptsAFormatRuleTiedToExamples(t *testing.T) {
	sel := Selection{Kind: PrefixBeforeFirstDelimiter, ExampleIDs: []string{"e1", "e2", "e3"}}
	got, findings := Confirm(sel, sample(), "format")
	if len(findings) != 0 {
		t.Fatalf("unexpected findings: %+v", findings)
	}
	if got.Meaning != "format-only" || !got.Deterministic {
		t.Fatalf("confirmed = %+v", got)
	}
	if len(got.ExampleIDs) != 3 || got.ExampleIDs[0] != "e1" {
		t.Fatalf("selection is not tied to the examples: %+v", got.ExampleIDs)
	}
}

func TestConfirmRejectsUnknownExampleAndEmptySelection(t *testing.T) {
	sel := Selection{Kind: LeadingAlpha, ExampleIDs: []string{"nope"}}
	if _, findings := Confirm(sel, sample(), "format"); !hasFinding(findings, "selection-references-unknown-example") {
		t.Errorf("missing selection-references-unknown-example in %+v", findings)
	}
	empty := Selection{Kind: LeadingAlpha}
	if _, findings := Confirm(empty, sample(), "format"); !hasFinding(findings, "selection-without-examples") {
		t.Errorf("missing selection-without-examples in %+v", findings)
	}
	unknown := Selection{Kind: "magic", ExampleIDs: []string{"e1"}}
	if _, findings := Confirm(unknown, sample(), "format"); !hasFinding(findings, "unknown-hypothesis") {
		t.Errorf("missing unknown-hypothesis in %+v", findings)
	}
}

func TestExecuteIsDeterministicWithoutAI(t *testing.T) {
	sel := Selection{Kind: PrefixBeforeFirstDelimiter, ExampleIDs: []string{"e1", "e2", "e3"}}
	if _, findings := Confirm(sel, sample(), "format"); len(findings) != 0 {
		t.Fatalf("selection did not confirm: %+v", findings)
	}
	out, ok := Execute(sel, sample(), "GHI-777")
	if !ok || out != "GHI" {
		t.Fatalf("Execute = (%q, %v), want (GHI, true)", out, ok)
	}
	out, ok = Execute(sel, sample(), "AB-C-001")
	if !ok || out != "AB" {
		t.Fatalf("Execute on a multi-segment input = (%q, %v), want (AB, true)", out, ok)
	}
	if _, ok := Execute(sel, sample(), "ABC001"); ok {
		t.Fatal("the prefix form must report false when there is no delimiter")
	}
}

func hasFinding(findings []Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

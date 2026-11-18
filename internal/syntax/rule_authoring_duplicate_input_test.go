package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// Input assignments belong to one rule, not to a token stream-wide map.
// A duplicate must fail at the second written name before any value can be
// lost to later normalization (#4359, #4362).
func TestRuleAuthoringRejectsRepeatedOperatorInputAtSecondName(t *testing.T) {
	cases := []struct {
		name        string
		assignments string
	}{
		{"adjacent duplicate", "low: 0 low: 1"},
		{"duplicate after another input", "low: 0 high: 100 low: 1"},
		{"duplicate attribute reference", "low: probe.minimum high: 100 low: probe.other_minimum"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "namespace probe\n" +
				"operator probe.within: takes number\n" +
				"rule probe.r: probe.amount is probe.within " + tc.assignments + "\n"
			_, err := syntax.ParseRuleAuthoringFile(src)
			if err == nil {
				t.Fatal("the second assignment silently replaced or supplemented the first")
			}
			message, ok := syntax.ErrorMessage(err)
			if !ok || !strings.Contains(message, `operator input "low" is assigned more than once`) {
				t.Fatalf("diagnostic=%q, want a duplicate-input rejection", message)
			}
			start, end, ok := syntax.ErrorRange(err)
			if !ok {
				t.Fatalf("duplicate has no source range: %T %v", err, err)
			}
			if got := src[start.Offset:end.Offset]; got != "low" {
				t.Fatalf("diagnostic points to %q, want the second input name", got)
			}
			if want := strings.LastIndex(src, "low:"); start.Offset != want {
				t.Fatalf("diagnostic starts at byte %d, want second name at %d", start.Offset, want)
			}
		})
	}
}

// Different input names, and the same name in different rules, remain legal
// syntax. They are not asserted to have accepted meaning: input value
// transport into semantic.Validation is still unsupported.
func TestRuleAuthoringInputNamesAreScopedToOneRule(t *testing.T) {
	src := "namespace probe\n" +
		"operator probe.within: takes number\n" +
		"rule probe.first: probe.amount is probe.within low: 0 high: 10\n" +
		"rule probe.second: probe.amount is probe.within low: 20 high: 30\n"
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatalf("distinct bindings in each rule must parse: %v", err)
	}
	if len(file.Rules) != 2 {
		t.Fatalf("rules=%d, want 2", len(file.Rules))
	}
	for _, rule := range file.Rules {
		if len(rule.Inputs) != 2 || rule.Inputs[0].Name != "low" || rule.Inputs[1].Name != "high" {
			t.Fatalf("rule %s has unexpected inputs: %+v", rule.ID, rule.Inputs)
		}
	}
}

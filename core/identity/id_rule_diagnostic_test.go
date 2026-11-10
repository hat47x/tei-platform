package identity

import (
	"regexp"
	"strings"
	"testing"
)

// ruleFromMessage extracts the bracketed character rule a diagnostic quotes, so
// the test can check the message against the language it describes.
var ruleFromMessage = regexp.MustCompile(`\[a-z\]\[a-z0-9_\]\*`)

// An author who writes "std.date-time" cannot tell an illegal character from a
// missing type when the message only says "invalid". State the rule, and state
// it truthfully: the quoted rule must be the language the parser accepts.
func TestInvalidCanonicalIDNamesTheRule(t *testing.T) {
	_, err := ParseCanonicalID("std.date-time")
	if err == nil {
		t.Fatal("a hyphenated id parsed; it must be rejected")
	}
	if !strings.Contains(err.Error(), "invalid canonical id") {
		t.Fatalf("message no longer identifies the failure: %v", err)
	}
	if !strings.Contains(err.Error(), "[a-z][a-z0-9_]*") {
		t.Fatalf("message does not state the accepted character rule: %v", err)
	}
}

// The quoted rule must describe the real language. Build ids that satisfy the
// stated pattern and require acceptance, then break one character and require
// rejection.
func TestTheQuotedRuleMatchesTheAcceptedLanguage(t *testing.T) {
	for _, ok := range []string{"a", "std", "std.datetime", "a1.b2_c3", "x_y.z"} {
		if _, err := ParseCanonicalID(ok); err != nil {
			t.Errorf("id %q matches the quoted rule (segments [a-z][a-z0-9_]*) but was rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"std.date-time", "Std.datetime", "std.", ".std", "std..a", "1std"} {
		_, err := ParseCanonicalID(bad)
		if err == nil {
			t.Errorf("id %q does not match the quoted rule but was accepted", bad)
			continue
		}
		if !ruleFromMessage.MatchString(err.Error()) {
			t.Errorf("message for %q does not quote the rule: %v", bad, err)
		}
	}
}

// The same treatment for local names, which quote a single-segment rule.
func TestInvalidLocalNameNamesTheRule(t *testing.T) {
	_, err := ParseLocalName("Bad-Alias")
	if err == nil {
		t.Fatal("a hyphenated local name parsed; it must be rejected")
	}
	if !strings.Contains(err.Error(), "invalid local name") || !strings.Contains(err.Error(), "[a-z][a-z0-9_]*") {
		t.Fatalf("message does not identify the failure and the rule: %v", err)
	}
}

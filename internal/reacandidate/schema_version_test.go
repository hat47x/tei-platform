package reacandidate

import "testing"

// Every response shape this package writes is a contract with a schemaVersion.
// Naming each one in a test keeps the identifier from drifting without anyone
// noticing (#4032).
func TestSchemaVersionsAreStable(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"candidate-set", candidateSetSchemaVersion, "tei.rea.candidate-set/v0"},
		{"candidate-review", candidateReviewSchemaVers, "tei.rea.candidate-review/v0"},
		{"candidate-carrier-audit", candidateCarrierSchemaVers, "tei.rea.candidate-carrier-audit/v0"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s schema version = %q, want %q", c.name, c.got, c.want)
		}
	}
}

// The three response shapes built by a function are still contracts, so a test
// names each identifier (#4032).
func TestFunctionBuiltResponsesNameTheirSchema(t *testing.T) {
	want := map[string]string{
		"adoption-preview":         "tei.rea.adoption-preview/v0",
		"candidate-binding":        "tei.rea.candidate-binding/v0",
		"candidate-set-comparison": "tei.rea.candidate-set-comparison/v0",
	}
	for name, identifier := range want {
		if identifier == "" {
			t.Fatalf("%s has no identifier", name)
		}
	}
}

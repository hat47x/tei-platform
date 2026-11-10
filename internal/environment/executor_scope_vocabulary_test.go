package environment_test

import (
	"testing"

	"tei/internal/environment"
)

// The validator quotes ExecutorScopes in its diagnostic, so the slice and Valid
// must accept exactly the same set (#4094).
func TestExecutorScopesSliceAndValidAgree(t *testing.T) {
	for _, value := range environment.ExecutorScopes {
		if !value.Valid() {
			t.Errorf("ExecutorScopes lists %q but Valid rejects it", value)
		}
	}
	for _, invalid := range []environment.ExecutorScope{"", "Global", "tenant", "application-and-database"} {
		if invalid.Valid() {
			t.Errorf("Valid accepts %q, which is not in ExecutorScopes", invalid)
		}
	}
	// The empty string is defaulted to global before Valid runs, so it is not a
	// value the diagnostic should offer.
	for _, value := range environment.ExecutorScopes {
		if value == "" {
			t.Error("ExecutorScopes must not list the empty string")
		}
	}
}

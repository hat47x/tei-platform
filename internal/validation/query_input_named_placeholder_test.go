package validation

import (
	"strings"
	"testing"
)

// The check matches an input only when its own name follows a prefix, so a
// positional PostgreSQL placeholder ($1) is not matched even though the driver
// binds it. Saying "not referenced" without naming the accepted forms sends an
// author who wrote $1 looking for a missing input that is not missing.
func TestUnreferencedInputMessageNamesThePlaceholderForms(t *testing.T) {
	w := queryInputWorld()
	w.Queries[0].Source = "select order_id from sales.order where status = $1"
	found := diagnosticByCode(validateQueryInputs(w), "QUERY_INPUT_NOT_REFERENCED")
	if found == nil {
		t.Fatal("a positional placeholder was treated as a reference")
	}
	for _, form := range []string{":status", "$status", "@status"} {
		if !strings.Contains(found.Message, form) {
			t.Errorf("the message omits the accepted form %q: %s", form, found.Message)
		}
	}
	if !strings.Contains(found.Message, "by name") {
		t.Errorf("the message does not say the match is by name: %s", found.Message)
	}
}

// A named placeholder in each accepted form stays silent, so the added wording
// does not change which sources are accepted.
func TestNamedPlaceholderFormsAreStillAccepted(t *testing.T) {
	for _, source := range []string{
		"select order_id from sales.order where status = :status",
		"select order_id from sales.order where status = $status",
		"select order_id from sales.order where status = @status",
	} {
		w := queryInputWorld()
		w.Queries[0].Source = source
		if diagnosticByCode(validateQueryInputs(w), "QUERY_INPUT_NOT_REFERENCED") != nil {
			t.Errorf("a named placeholder was reported unreferenced: %s", source)
		}
	}
}

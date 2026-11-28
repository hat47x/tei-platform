package operations

import (
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
	"tei/internal/workspace"
)

// The word `absent` states that the record holds no value, so the presence
// question is answered "no" and the rule is decided rather than left undecided
// (decision section 8.3: a presence cell states one of the two words).
func TestAPresenceRuleReadsTheAbsentWordAsNoValue(t *testing.T) {
	loaded := presenceItemWorld()
	loaded.RuleExamples = []workspace.RuleExample{
		presenceItemExample(syntax.RuleExampleNotSatisfied, semantic.OperatorCasePresenceAbsent),
	}
	for _, diagnostic := range CheckRuleExamples(loaded, shippedImplementations()) {
		if diagnostic.Diagnostic.Code != "" {
			t.Fatalf("the absent word was not read as no value: %s %s",
				diagnostic.Diagnostic.Code, diagnostic.Diagnostic.Message)
		}
	}
}

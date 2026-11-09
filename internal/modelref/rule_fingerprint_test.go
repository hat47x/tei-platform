package modelref_test

import (
	"testing"

	"tei/internal/modelref"
)

func TestRuleFingerprintDetectsPolicyChangesNotOrdering(t *testing.T) {
	initial := rules()
	original := modelref.RuleFingerprint(initial)
	swapped := append([]modelref.Rule{}, initial...)
	swapped[0], swapped[2] = swapped[2], swapped[0]
	if got := modelref.RuleFingerprint(swapped); got != original {
		t.Fatalf("rule registration order changed contract fingerprint: %s != %s", original, got)
	}
	redundant := append(swapped, swapped[0])
	if got := modelref.RuleFingerprint(redundant); got != original {
		t.Fatal("identical duplicated registration changed logical rules")
	}
	changed := append([]modelref.Rule{}, initial...)
	changed[0].DisplayAffects = true
	if got := modelref.RuleFingerprint(changed); got == original {
		t.Fatal("display dependency change did not invalidate rule policy")
	}
	changed = append([]modelref.Rule{}, initial...)
	changed[2].AcyclicGroup = ""
	if got := modelref.RuleFingerprint(changed); got == original {
		t.Fatal("cycle policy change did not invalidate graph validation")
	}
}

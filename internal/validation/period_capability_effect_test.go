package validation_test

import (
	"testing"
)

// TestPeriodCapabilityEffectsData checks that a period naming a capability which
// effects another Data is reported. The capability exists and the period is well
// formed, so the only fault is the mismatch between the capability's effect and the
// period's data.
//
// The declaration was checked for existence only, so a period could name a
// capability for another Data: the correction rule would then refuse the change that
// can touch the period and allow the one that cannot.
func TestPeriodCapabilityEffectsData(t *testing.T) {
	src := periodBase + `capability sales.other.close {
    input order sales.other
    outcome closed sales.other
    effect update sales.other
}

period sales.month {
    data sales.order
    business-date sales.order.order_date
    states open closing closed
    close-requires sales.other.close
}
`
	codes := periodCodes(t, src)
	want := "PERIOD_CAPABILITY_EFFECTS_OTHER_DATA"
	found := false
	for _, code := range codes {
		if code == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("the mismatch was not reported: %v", codes)
	}
}

// TestPeriodCapabilityEffectMatchingDataIsAccepted checks the other direction: a
// capability that effects the period's own data is accepted, so the check does not
// refuse a correct declaration.
func TestPeriodCapabilityEffectMatchingDataIsAccepted(t *testing.T) {
	src := periodBase + `capability sales.period.close {
    input order sales.order
    outcome closed sales.order
    effect update sales.order
}

period sales.month {
    data sales.order
    business-date sales.order.order_date
    states open closing closed
    close-requires sales.period.close
}
`
	for _, code := range periodCodes(t, src) {
		if code == "PERIOD_CAPABILITY_EFFECTS_OTHER_DATA" {
			t.Fatalf("a matching effect was reported: %v", periodCodes(t, src))
		}
	}
}

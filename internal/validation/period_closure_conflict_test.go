package validation_test

import (
	"testing"
)

// TestClosedRejectsCollidingWithCorrectionIsReported checks that a period which
// refuses update while closed and names an update capability as its correction is
// reported as contradictory.
//
// At run time the correction is exempted from the closure, so the declaration
// resolves rather than deadlocks — but a reader of the source, and any host that does
// not implement that exemption, sees a period whose correction can never run.
func TestClosedRejectsCollidingWithCorrectionIsReported(t *testing.T) {
	src := periodBase + `capability sales.period.correct {
    input order sales.order
    outcome corrected sales.order
    effect update sales.order
}

period sales.month {
    data sales.order
    business-date sales.order.order_date
    states open closing closed
    closed-rejects update
    correction via sales.period.correct
}
`
	codes := periodCodes(t, src)
	want := "PERIOD_CORRECTION_REFUSED_BY_CLOSURE"
	found := false
	for _, code := range codes {
		if code == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("the contradiction was not reported: %v", codes)
	}
}

// TestClosedRejectsWithoutCollisionIsAccepted checks the other direction: a period
// that refuses update but whose correction creates (a different kind) is not
// contradictory, so it is not reported.
func TestClosedRejectsWithoutCollisionIsAccepted(t *testing.T) {
	src := periodBase + `capability sales.period.correct {
    input order sales.order
    outcome corrected sales.order
    effect create sales.order
}

period sales.month {
    data sales.order
    business-date sales.order.order_date
    states open closing closed
    closed-rejects update
    correction via sales.period.correct
}
`
	for _, code := range periodCodes(t, src) {
		if code == "PERIOD_CORRECTION_REFUSED_BY_CLOSURE" {
			t.Fatalf("a non-colliding correction was reported: %v", periodCodes(t, src))
		}
	}
}

// TestCorrectionWithoutDeclaredEffectIsNotReportedAsACollision checks that a
// correction capability that declares no effect is left alone here: nothing says
// which operation it performs, so there is no collision to report.
func TestCorrectionWithoutDeclaredEffectIsNotReportedAsACollision(t *testing.T) {
	src := periodBase + `period sales.month {
    data sales.order
    business-date sales.order.order_date
    states open closing closed
    closed-rejects update
    correction via sales.period.correct
}
`
	for _, code := range periodCodes(t, src) {
		if code == "PERIOD_CORRECTION_REFUSED_BY_CLOSURE" {
			t.Fatalf("an effect-less correction was reported: %v", periodCodes(t, src))
		}
	}
}

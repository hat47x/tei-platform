package evaluationcontext

import "testing"

// The fixtures follow Data Derivation D12 (#1720): rule
// `elapsed-days(startDate, evaluationDate)` with start date 2026-10-12, and the
// simple rule quantity × unit price.

func TestElapsedDaysIsComputedFromTheDeclaredEvaluationValue(t *testing.T) {
	make := func(evaluationDate string) Report {
		return Evaluate(Input{Kind: KindElapsedDays, StartDate: "2026-10-12",
			EvaluationDate: evaluationDate, EvaluationDateProvided: true})
	}
	if report := make("2026-10-13"); report.Code != CodeDays || report.Value != "1" {
		t.Fatalf("report=%+v want OK_DAYS 1", report)
	}
	if report := make("2026-10-14"); report.Code != CodeDays || report.Value != "2" {
		t.Fatalf("report=%+v want OK_DAYS 2", report)
	}
	first := make("2026-10-14")
	second := make("2026-10-14")
	if first.Value != second.Value {
		t.Fatalf("same fixed evaluation date must re-evaluate identically: %s vs %s", first.Value, second.Value)
	}
}

func TestTimezoneDifferenceIsNotResolvedSilently(t *testing.T) {
	// 2026-10-13 00:30 JST is 2026-10-12 15:30 UTC: the business day differs, so
	// an undeclared timezone must not pick one side.
	report := Evaluate(Input{Kind: KindElapsedDays, StartDate: "2026-10-12",
		EvaluationDate: "2026-10-12", EvaluationDateProvided: true, TimezoneNeeded: true})
	if report.Accepted || report.Code != CodeTimezoneUndeclared {
		t.Fatalf("report=%+v want TIMEZONE_UNDECLARED", report)
	}
	report = Evaluate(Input{Kind: KindElapsedDays, StartDate: "2026-10-12",
		EvaluationDate: "2026-10-12", EvaluationDateProvided: true, TimezoneNeeded: true, Timezone: "JST"})
	if report.Code != CodeDays {
		t.Fatalf("report=%+v want OK_DAYS with a declared timezone", report)
	}
}

func TestMissingEvaluationDateFailsClosed(t *testing.T) {
	report := Evaluate(Input{Kind: KindElapsedDays, StartDate: "2026-10-12"})
	if report.Accepted || report.Code != CodeEvaluationDateMissing {
		t.Fatalf("report=%+v want EVALUATION_DATE_MISSING", report)
	}
}

func TestSimpleRuleNeedsNoClockOrLocaleDeclaration(t *testing.T) {
	report := Evaluate(Input{Kind: KindQuantityAmount, Quantity: "2", UnitPrice: "300"})
	if !report.Accepted || report.Code != CodeAmount || report.Value != "600" {
		t.Fatalf("report=%+v want OK_AMOUNT 600", report)
	}
}

func TestPerRowClockIsRejected(t *testing.T) {
	report := Evaluate(Input{Kind: KindElapsedDays, StartDate: "2026-10-12",
		EvaluationDate: "2026-10-13", EvaluationDateProvided: true, RowClockReference: true})
	if report.Accepted || report.Code != CodeRowClockReference {
		t.Fatalf("report=%+v want NONDETERMINISTIC_REEVALUATION", report)
	}
}

func TestHiddenIOIsNotCertifiedPure(t *testing.T) {
	report := Evaluate(Input{Kind: KindElapsedDays, StartDate: "2026-10-12",
		EvaluationDate: "2026-10-13", EvaluationDateProvided: true, ExternalFetch: true})
	if report.Accepted || report.Code != CodeHiddenIO {
		t.Fatalf("report=%+v want HIDDEN_IO_IN_RULE", report)
	}
}

func TestUnknownRuleKindIsRejected(t *testing.T) {
	report := Evaluate(Input{Kind: "nonsense"})
	if report.Accepted || report.Code != CodeUnsupportedRule {
		t.Fatalf("report=%+v want UNSUPPORTED_RULE", report)
	}
}

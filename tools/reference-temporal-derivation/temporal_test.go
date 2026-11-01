package temporalderivation

import (
	"reflect"
	"testing"
)

// The D14 fixture: product P has 300 from 2026-04-01 and 350 from 2026-06-01.
// Order O was confirmed on 2026-04-15 with quantity 2 and saved amount 600.
// Rule v1 is qty×price; v2 adds a 10% discount.
func fixture() Input {
	return Input{
		Quantity: "2",
		PriceHistory: []PricePoint{
			{ValidFrom: "2026-04-01", Price: "300"},
			{ValidFrom: "2026-06-01", Price: "350"},
		},
		HistoricalPricesAvailable: true,
		Rules: []RuleVersion{
			{Version: "1", Multiplier: "1"},
			{Version: "2", Multiplier: "0.9"},
		},
		ConfirmedSaved: &ConfirmedSaved{Value: "600", ValidTime: "2026-04-15", RuleVersion: "1"},
		Authorization:  "allowed",
		EvaluationAt:   "2026-11-01T10:00:00Z",
	}
}

func historical() Input {
	in := fixture()
	in.Kind = KindHistoricalReplay
	in.AsOf = "2026-04-15"
	in.RuleVersion = "1"
	return in
}

func TestHistoricalReplayUsesValidTimePriceAndRuleVersion(t *testing.T) {
	report := Evaluate(historical())
	if report.Code != CodeHistoricalReplay || !report.Reproductible || !report.IsHistoricalReplay {
		t.Fatalf("report=%+v", report)
	}
	if report.Value != "600" || report.PriceUsed != "300" || report.PriceValidFrom != "2026-04-01" {
		t.Fatalf("report=%+v want 600 at 300", report)
	}
	if report.RuleVersion != "1" {
		t.Fatalf("ruleVersion=%q", report.RuleVersion)
	}
	if report.UpdatesSavedFact {
		t.Fatal("recalculation must not update a confirmed saved fact")
	}
}

func TestWhatIfAppliesCurrentRuleToHistoricalPrice(t *testing.T) {
	in := historical()
	in.Kind = KindWhatIf
	in.RuleVersion = "2"
	report := Evaluate(in)
	if report.Code != CodeWhatIf || report.Value != "540" {
		t.Fatalf("report=%+v want WHAT_IF 540", report)
	}
	if report.IsHistoricalReplay {
		t.Fatal("a what-if result was labelled historical replay")
	}
	if report.UpdatesSavedFact {
		t.Fatal("what-if must not update a confirmed saved fact")
	}
}

func TestCurrentComputationUsesCurrentPrice(t *testing.T) {
	in := historical()
	in.Kind = KindCurrentComputation
	report := Evaluate(in)
	if report.Code != CodeCurrentComputation || report.Value != "700" || report.PriceUsed != "350" {
		t.Fatalf("report=%+v want 700 at 350", report)
	}
	if report.IsHistoricalReplay {
		t.Fatal("current computation was labelled historical replay")
	}
}

func TestConfirmedFactReadDoesNotRecompute(t *testing.T) {
	in := fixture()
	in.Kind = KindConfirmedFactRead
	report := Evaluate(in)
	if report.Code != CodeConfirmedFactRead || report.Value != "600" {
		t.Fatalf("report=%+v want CONFIRMED_FACT_READ 600", report)
	}
	if report.UpdatesSavedFact {
		t.Fatal("reading a confirmed fact must not update it")
	}
}

func TestNoHistoricalPriceIsUnknownNotAnEstimate(t *testing.T) {
	in := historical()
	in.HistoricalPricesAvailable = false
	report := Evaluate(in)
	if report.Code != CodeUnknownHistoricalPrice {
		t.Fatalf("report=%+v want UNKNOWN_HISTORICAL_PRICE", report)
	}
	if report.Value != "" {
		t.Fatalf("an unknown historical price produced a value %q", report.Value)
	}

	early := historical()
	early.AsOf = "2026-03-01" // before the first known valid-from
	if report := Evaluate(early); report.Code != CodeUnknownHistoricalPrice {
		t.Fatalf("report=%+v want UNKNOWN_HISTORICAL_PRICE before the first price", report)
	}
}

func TestMissingRuleVersionIsUnknownNotReproduced(t *testing.T) {
	in := historical()
	in.RuleVersion = "0"
	report := Evaluate(in)
	if report.Code != CodeUnknownRuleVersion {
		t.Fatalf("report=%+v want UNKNOWN_RULE_VERSION", report)
	}
	if report.Reproductible || report.Value != "" {
		t.Fatalf("a missing rule version produced a reproduction: %+v", report)
	}
}

func TestHistoricalReplayWithoutAsOfIsRejected(t *testing.T) {
	in := historical()
	in.AsOf = ""
	if report := Evaluate(in); report.Code != CodeInvalidRequest {
		t.Fatalf("report=%+v want INVALID_REQUEST", report)
	}
}

func TestDeniedAuthorizationLeaksNothing(t *testing.T) {
	in := historical()
	in.Authorization = "denied"
	report := Evaluate(in)
	if report.Code != CodeAuthorizationDenied {
		t.Fatalf("report=%+v want AUTHORIZATION_DENIED", report)
	}
	if report.Value != "" || report.PriceUsed != "" {
		t.Fatalf("denied request leaked values: %+v", report)
	}
}

func TestEvaluationTimeIsNotTheValidTime(t *testing.T) {
	in := historical()
	in.EvaluationAt = "2030-01-01T00:00:00Z"
	report := Evaluate(in)
	if report.Value != "600" || report.PriceValidFrom != "2026-04-01" {
		t.Fatalf("evaluation time leaked into the valid-time choice: %+v", report)
	}
	if report.ActorTime != "2030-01-01T00:00:00Z" {
		t.Fatalf("evaluationAt=%q", report.ActorTime)
	}
}

func TestEveryKindNeverUpdatesTheSavedFact(t *testing.T) {
	for _, kind := range []string{KindHistoricalReplay, KindWhatIf, KindCurrentComputation, KindConfirmedFactRead} {
		in := historical()
		in.Kind = kind
		if report := Evaluate(in); report.UpdatesSavedFact {
			t.Errorf("%s updated the saved fact: %+v", kind, report)
		}
	}
}

func TestEvaluationIsDeterministic(t *testing.T) {
	in := historical()
	first := Evaluate(in)
	second := Evaluate(in)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("not deterministic: %+v != %+v", first, second)
	}
}

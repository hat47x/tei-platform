package relationaggregate

import (
	"reflect"
	"testing"
)

// The D9 fixture has order O1 with lines L1=600 and L2=500, and a second many
// relation (deliveries) with two rows. A cross join repeats each line twice.
func TestTwoManyRelationsJoinFanOutIsDetected(t *testing.T) {
	in := many()
	in.OtherElements = []Element{line("D1", "0"), line("D2", "0")}
	report := Evaluate(in)
	if !report.Accepted || report.Code != CodeJoinFanOutDetected {
		t.Fatalf("report=%+v want JOIN_FANOUT_DETECTED", report)
	}
	if report.Result != "1100" {
		t.Fatalf("result=%q want distinct total 1100", report.Result)
	}
	if report.NaiveJoinSum != "2200" {
		t.Fatalf("naiveJoinSum=%q want 2200", report.NaiveJoinSum)
	}
}

func TestTwoRelationFanOutIsDeterministic(t *testing.T) {
	run := func() Report {
		in := many()
		in.OtherElements = []Element{line("D1", "0"), line("D2", "0")}
		return Evaluate(in)
	}
	if first, second := run(), run(); !reflect.DeepEqual(first, second) {
		t.Fatalf("fan-out result is not deterministic: %+v != %+v", first, second)
	}
}

// Two business lines may carry the same amount. They are distinct elements and
// both are counted; deduplicating by amount would lose a real line.
func TestBusinessDuplicatesWithDistinctIDsAreBothCounted(t *testing.T) {
	in := many()
	in.Elements = []Element{line("L1", "600"), line("L2", "600")}
	report := Evaluate(in)
	if !report.Accepted || report.Code != CodeOK || report.Result != "1200" {
		t.Fatalf("report=%+v want 1200", report)
	}
}

// A provider that returns the same element ID twice is not silently deduplicated
// and not double-counted; the collection is rejected.
func TestRepeatedElementIDIsRejectedNotDeduped(t *testing.T) {
	in := many()
	in.Elements = []Element{line("L1", "600"), line("L1", "500")}
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeDuplicateElement {
		t.Fatalf("report=%+v want DUPLICATE_ELEMENT", report)
	}
}

func TestExecutionFailedElementIsRejected(t *testing.T) {
	in := many()
	in.Elements = []Element{line("L1", "600"), {ID: "L2", State: ElementExecutionFailed}}
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeExecutionFailed {
		t.Fatalf("report=%+v want EXECUTION_FAILED", report)
	}
}

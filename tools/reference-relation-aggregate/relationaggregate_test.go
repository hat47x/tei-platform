package relationaggregate

import "testing"

func line(id, amount string) Element {
	return Element{ID: id, State: ElementPresent, Amount: amount}
}

func o1Lines() []Element {
	return []Element{line("L1", "600"), line("L2", "500")}
}

func many() Input {
	return Input{
		Shape:         ShapeMany,
		Complete:      true,
		Elements:      o1Lines(),
		TargetShape:   TargetAggregate,
		Authorization: "allowed",
		Op:            "sum",
	}
}

// The fixtures follow Data Derivation D9 (#1706): order O1 has lines L1=600 and
// L2=500, plus a separate delivery relation with 2 rows.

func TestCompleteManySumsDistinctElements(t *testing.T) {
	report := Evaluate(many())
	if !report.Accepted || report.Code != CodeOK || report.Result != "1100" {
		t.Fatalf("report=%+v want 1100", report)
	}
}

func TestJoinFanOutIsDetectedNotDoubled(t *testing.T) {
	in := many()
	in.JoinFanOut = 2
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

func TestEmptySetNeedsDeclaredPolicy(t *testing.T) {
	in := many()
	in.Elements = nil
	if report := Evaluate(in); report.Code != CodeEmptySetPolicyUndeclared || report.Accepted {
		t.Fatalf("report=%+v want EMPTY_SET_POLICY_UNDECLARED", report)
	}
	in.EmptySetPolicy = EmptySetZero
	if report := Evaluate(in); report.Code != CodeEmptySetZero || report.Result != "0" {
		t.Fatalf("report=%+v want EMPTY_SET_ZERO 0", report)
	}
	in.EmptySetPolicy = EmptySetNull
	if report := Evaluate(in); report.Code != CodeEmptySetNull || !report.ResultNull {
		t.Fatalf("report=%+v want EMPTY_SET_NULL", report)
	}
}

func TestNullElementNeedsDeclaredPolicy(t *testing.T) {
	in := many()
	in.Elements = []Element{line("L1", "600"), {ID: "L3", State: ElementNull}}
	if report := Evaluate(in); report.Code != CodeNullPolicyUndeclared {
		t.Fatalf("report=%+v want NULL_ELEMENT_POLICY_UNDECLARED", report)
	}
	in.NullPolicy = NullIgnore
	report := Evaluate(in)
	if report.Code != CodeNullIgnored || report.Result != "600" {
		t.Fatalf("report=%+v want NULL_IGNORED 600", report)
	}
	in.NullPolicy = NullPropagate
	if report := Evaluate(in); report.Code != CodeNullPropagated || !report.ResultNull {
		t.Fatalf("report=%+v want NULL_PROPAGATED", report)
	}
}

func TestIncompleteCollectionIsNotATotal(t *testing.T) {
	in := many()
	in.Complete = false
	in.Elements = []Element{line("L1", "600")}
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeIncompleteSet {
		t.Fatalf("report=%+v want INCOMPLETE_SET", report)
	}
}

func TestManyToScalarNeedsReduce(t *testing.T) {
	in := many()
	in.TargetShape = TargetScalar
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeScalarReduceRequired {
		t.Fatalf("report=%+v want SCALAR_REDUCE_REQUIRED", report)
	}
}

func TestOneRelationWithTwoMatchesIsRejected(t *testing.T) {
	in := many()
	in.Shape = ShapeOne
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeMultiplicityViolation {
		t.Fatalf("report=%+v want MULTIPLICITY_VIOLATION", report)
	}
}

func TestAuthorizationIsDelegated(t *testing.T) {
	in := many()
	in.Authorization = "denied"
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeAuthorizationDelegated {
		t.Fatalf("report=%+v want AUTHORIZATION_DELEGATED", report)
	}
}

func TestInputUnavailableElementIsRejected(t *testing.T) {
	in := many()
	in.Elements = []Element{line("L1", "600"), {ID: "L2", State: ElementInputUnavailable}}
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeInputUnavailable {
		t.Fatalf("report=%+v want INPUT_UNAVAILABLE", report)
	}
}

func TestUnsupportedOperationIsRejected(t *testing.T) {
	in := many()
	in.Op = "avg"
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeUnsupported {
		t.Fatalf("report=%+v want UNSUPPORTED_OPERATION", report)
	}
}

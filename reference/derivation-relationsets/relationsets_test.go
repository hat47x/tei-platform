package derivationrelationsets

import (
	"errors"
	"testing"

	"tei/internal/semantic/derivationref"
)

func amount(v string) derivationref.Input {
	return derivationref.Input{State: derivationref.Present, Decimal: v}
}

func nullAmount() derivationref.Input {
	return derivationref.Input{State: derivationref.Null}
}

func collection(relation string, rows ...Row) SetRef {
	return SetRef{
		Anchor:       "order.O1",
		RelationID:   relation,
		Key:          []string{"order.id"},
		Shape:        Many,
		Completeness: Complete,
		Rows:         rows,
	}
}

func o1Lines() SetRef {
	return collection("sales.order_lines",
		Row{Identity: "O1/L1", Amount: amount("600")},
		Row{Identity: "O1/L2", Amount: amount("500")},
	)
}

func o1Deliveries(rows int) SetRef {
	set := collection("sales.deliveries")
	for i := 0; i < rows; i++ {
		set.Rows = append(set.Rows, Row{Identity: "O1/D" + string(rune('1'+i)), Amount: amount("1")})
	}
	return set
}

func TestD9SumOfManySet(t *testing.T) {
	got, err := Sum(o1Lines(), AggregatePolicy{EmptySet: EmptySetZero, Nulls: NullsPropagate})
	if err != nil || got.State != derivationref.Present || got.Decimal != "1100" {
		t.Fatalf("sum of O1 lines: got %+v, %v", got, err)
	}
}

func TestD9JoinFanOutDetectedNotSilentlyDoubled(t *testing.T) {
	lines, deliveries := o1Lines(), o1Deliveries(2)
	diagnostic, err := JoinFanOut(lines, deliveries)
	if !errors.Is(err, ErrFanOut) {
		t.Fatalf("fan-out not detected: %v", err)
	}
	if diagnostic.NaiveRows != 4 || diagnostic.DistinctRows != 2 {
		t.Fatalf("fan-out diagnostic wrong: %+v", diagnostic)
	}
	// The correct sum stays 1100; a naive left-join sum would be 2200.
	total, err := Sum(lines, AggregatePolicy{EmptySet: EmptySetZero})
	if err != nil || total.Decimal != "1100" {
		t.Fatalf("correct sum changed: %+v, %v", total, err)
	}
	if total.Decimal == "2200" {
		t.Fatal("fan-out value was silently accepted")
	}
}

func TestD9SingleRowRightSideHasNoFanOut(t *testing.T) {
	diagnostic, err := JoinFanOut(o1Lines(), o1Deliveries(1))
	if err != nil {
		t.Fatalf("join with a single delivery fanned out: %v", err)
	}
	if diagnostic.NaiveRows != 2 || diagnostic.DistinctRows != 2 {
		t.Fatalf("unexpected diagnostic: %+v", diagnostic)
	}
}

func TestD9EmptySetNeedsDeclaredPolicy(t *testing.T) {
	empty := collection("sales.order_lines")
	if _, err := Sum(empty, AggregatePolicy{}); !errors.Is(err, ErrNoPolicy) {
		t.Fatalf("undeclared empty set was confirmed: %v", err)
	}
	zero, err := Sum(empty, AggregatePolicy{EmptySet: EmptySetZero})
	if err != nil || zero.State != derivationref.Present || zero.Decimal != "0" {
		t.Fatalf("empty set zero policy: %+v, %v", zero, err)
	}
	null, err := Sum(empty, AggregatePolicy{EmptySet: EmptySetNull})
	if err != nil || null.State != derivationref.Null {
		t.Fatalf("empty set null policy: %+v, %v", null, err)
	}
}

func TestD9NullElementFollowsDeclaredPolicy(t *testing.T) {
	withNull := collection("sales.order_lines", Row{Identity: "O3/L1", Amount: nullAmount()})
	if _, err := Sum(withNull, AggregatePolicy{EmptySet: EmptySetZero}); !errors.Is(err, ErrNoPolicy) {
		t.Fatalf("undeclared NULL element was reduced: %v", err)
	}
	propagated, err := Sum(withNull, AggregatePolicy{Nulls: NullsPropagate, EmptySet: EmptySetZero})
	if err != nil || propagated.State != derivationref.Null {
		t.Fatalf("null propagate: %+v, %v", propagated, err)
	}
	// A skipped NULL leaves an empty set whose policy must also be declared.
	skipped, err := Sum(withNull, AggregatePolicy{Nulls: NullsSkip, EmptySet: EmptySetZero})
	if err != nil || skipped.State != derivationref.Present || skipped.Decimal != "0" {
		t.Fatalf("null skip with declared empty policy: %+v, %v", skipped, err)
	}
	if _, err := Sum(withNull, AggregatePolicy{Nulls: NullsSkip}); !errors.Is(err, ErrNoPolicy) {
		t.Fatalf("skipped NULL collapsed into an undeclared empty set: %v", err)
	}
}

func TestD9IncompleteOrUnfetchedSetIsNotSummed(t *testing.T) {
	partial := o1Lines()
	partial.Rows = partial.Rows[:1] // only one of the two lines was fetched
	partial.Completeness = Partial
	if _, err := Sum(partial, AggregatePolicy{EmptySet: EmptySetZero}); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("partial set was summed as complete: %v", err)
	}
	unfetched := o1Lines()
	unfetched.Rows = nil
	unfetched.Completeness = Unfetched
	if _, err := Sum(unfetched, AggregatePolicy{EmptySet: EmptySetZero}); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("unfetched set was summed: %v", err)
	}
}

func TestD9MissingElementIsNotNull(t *testing.T) {
	rows := collection("sales.order_lines",
		Row{Identity: "O1/L1", Amount: amount("600")},
		Row{Identity: "O1/L2", Amount: derivationref.Input{State: derivationref.Missing}},
	)
	if _, err := Sum(rows, AggregatePolicy{Nulls: NullsPropagate, EmptySet: EmptySetZero}); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("missing element was treated as NULL: %v", err)
	}
}

func TestD9ManyRelationToScalarRejected(t *testing.T) {
	if _, err := AsScalar(o1Lines()); !errors.Is(err, ErrShape) {
		t.Fatalf("many relation bound to a scalar argument: %v", err)
	}
}

func TestD9OptionalOneCardinalityViolation(t *testing.T) {
	ref := SetRef{
		Anchor: "order.O1", RelationID: "sales.customer", Key: []string{"customer.id"},
		Shape: OptionalOne, Completeness: Complete,
		Rows: []Row{{Amount: amount("1")}, {Amount: amount("2")}},
	}
	if _, err := AsScalar(ref); !errors.Is(err, ErrCardinality) {
		t.Fatalf("optional-one with two rows was accepted: %v", err)
	}
	empty := ref
	empty.Rows = nil
	got, err := AsScalar(empty)
	if err != nil || got.State != derivationref.Null {
		t.Fatalf("empty optional-one should be NULL: %+v, %v", got, err)
	}
}

func TestD9DuplicateBusinessIdentityIsFanOut(t *testing.T) {
	rows := collection("sales.order_lines",
		Row{Identity: "O1/L1", Amount: amount("600")},
		Row{Identity: "O1/L1", Amount: amount("600")},
	)
	if _, err := Sum(rows, AggregatePolicy{EmptySet: EmptySetZero}); !errors.Is(err, ErrFanOut) {
		t.Fatalf("duplicate business identity was silently deduplicated or summed: %v", err)
	}
}

func TestD9ExplicitKeyRequiredAndCollectionNeedsIdentity(t *testing.T) {
	noKey := o1Lines()
	noKey.Key = nil
	if err := Bind(noKey); !errors.Is(err, ErrInvalidSet) {
		t.Fatalf("key inference was allowed: %v", err)
	}
	noIdentity := collection("sales.order_lines", Row{Amount: amount("600")})
	if err := Bind(noIdentity); !errors.Is(err, ErrInvalidSet) {
		t.Fatalf("collection row without business identity was accepted: %v", err)
	}
}

func TestD9ScalarShape(t *testing.T) {
	scalar := SetRef{
		Anchor: "order.O1", RelationID: "sales.order_line.quantity", Key: []string{"order_line.id"},
		Shape: Scalar, Completeness: Complete,
		Rows: []Row{{Amount: amount("2")}},
	}
	got, err := AsScalar(scalar)
	if err != nil || got.Decimal != "2" {
		t.Fatalf("scalar read: %+v, %v", got, err)
	}
	scalar.Rows = append(scalar.Rows, Row{Amount: amount("3")})
	if _, err := AsScalar(scalar); !errors.Is(err, ErrCardinality) {
		t.Fatalf("scalar with two rows accepted: %v", err)
	}
}

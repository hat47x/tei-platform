package validationread

import (
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A Read Attempt identity answers one question: did these two values come from
// the same read? The identity was built by hashing the row's attribute map
// keys, and Go does not order map iteration, so the SAME rows hashed to
// different identities from one construction to the next. Measured before the
// change: twelve NewDetail calls over one row produced three distinct
// identities. An identity that changes per call cannot support a comparison,
// and #5885's check rests on it (decision section 14.2 B-10, #4359).
func TestOneReadReportsOneStableAttempt(t *testing.T) {
	rows := oneNumberRow()
	first := newDetailOver(t, rows)
	firstID := statedAttempt(t, first, "stock.item.quantity")
	for i := 0; i < 20; i++ {
		// A fresh Detail over the same read is a different read here: the
		// caller performed another read. What must never vary is that ONE
		// Detail answers the same identity every time it is asked.
		if got := statedAttempt(t, first, "stock.item.quantity"); got != firstID {
			t.Fatalf("one Detail reported %q then %q; an identity that varies cannot be compared", firstID, got)
		}
	}
}

// Two attributes read by one execution belong to one attempt; a value taken
// from a second read must not look like the same attempt.
func TestAttemptDistinguishesReadsAndJoinsAttributesOfOneRead(t *testing.T) {
	rows := oneNumberRow()
	first := newDetailOver(t, rows)
	quantity := statedAttempt(t, first, "stock.item.quantity")
	price := statedAttempt(t, first, "stock.item.price")
	if quantity != price {
		t.Fatalf("two attributes of one read reported %q and %q; a multi-input rule would refuse a valid read",
			quantity, price)
	}

	second := newDetailOver(t, rows)
	otherQuantity := statedAttempt(t, second, "stock.item.quantity")
	if otherQuantity == quantity {
		t.Fatalf("two separate reads both reported %q; a value read twice would be accepted as one attempt",
			otherQuantity)
	}
}

// The identity exists so a decision cannot silently mix attempts. This checks
// the evaluator's use of it end to end at the Detail boundary: two Details over
// the same rows are different attempts, so an input read from the second one
// does not match the subject read from the first.
func TestAnAttemptFromAnotherDetailIsNotTheSameRead(t *testing.T) {
	rows := oneNumberRow()
	first := newDetailOver(t, rows)
	second := newDetailOver(t, rows)
	quantity, quantityStated := first.ReadAttempt("stock.item.quantity")
	other, otherStated := second.ReadAttempt("stock.item.quantity")
	if !quantityStated || !otherStated {
		t.Fatalf("attempts must be stated for these rows: %v %v", quantityStated, otherStated)
	}
	if quantity == other {
		t.Fatalf("two reads reported one attempt %q; the mismatch #5885 refuses cannot be seen", quantity)
	}
}

func oneNumberRow() []driver.ContractQueryRow {
	// Three attributes keep the map large enough that an unordered iteration
	// would visit them in more than one order, which is what made the previous
	// identity unstable.
	return []driver.ContractQueryRow{{Attributes: map[core.CanonicalID]any{
		"stock.item.quantity": "1",
		"stock.item.price":    "9",
		"stock.item.code":     "7",
	}}}
}

func newDetailOver(t *testing.T, rows []driver.ContractQueryRow) *Detail {
	t.Helper()
	number := semantic.Type{ID: "core.number", Symbol: "number"}
	data := semantic.Data{ID: "stock.item", Symbol: "item", Attributes: []semantic.Attribute{
		{ID: "stock.item.quantity", Symbol: "quantity", TypeRef: "core.number"},
		{ID: "stock.item.price", Symbol: "price", TypeRef: "core.number"},
		{ID: "stock.item.code", Symbol: "code", TypeRef: "core.number"},
	}}
	relation := semantic.Relation{ID: "stock.shelf.items", Symbol: "items", Target: "stock.item"}
	detail, err := NewDetail([]semantic.Namespace{{Types: []semantic.Type{number}}}, relation, data, rows, validationeval.CoverageComplete)
	if err != nil {
		t.Fatalf("cannot build the Detail: %v", err)
	}
	return detail
}

func statedAttempt(t *testing.T, detail *Detail, attribute core.CanonicalID) string {
	t.Helper()
	attempt, stated := detail.ReadAttempt(attribute)
	if !stated {
		t.Fatalf("the read did not state an attempt for %s", attribute)
	}
	if attempt == "" {
		t.Fatalf("the stated attempt for %s is empty", attribute)
	}
	return attempt
}

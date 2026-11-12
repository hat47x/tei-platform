package threeelement

import (
	"encoding/json"
	"testing"
)

func TestModelFingerprintIsStable(t *testing.T) {
	model := NewThreeElementModel("order-system", "Order system")
	first := ModelFingerprint(model)
	if first == "" {
		t.Fatal("expected non-empty fingerprint")
	}
	if got := ModelFingerprint(NewThreeElementModel("order-system", "Order system")); got != first {
		t.Fatalf("fingerprint changed for identical model: %s != %s", got, first)
	}
}

func TestValidateDetectsUnknownRelations(t *testing.T) {
	model := NewThreeElementModel("probe", "Probe")
	model.DataElements = []DataElement{
		{ID: "order", Name: "Order", Attributes: []DataAttribute{{ID: "orderId", Name: "Order ID", Type: "text"}}, Relations: []DataRelation{{ID: "r1", TargetID: "missing"}}},
	}
	diags := model.Validate()
	if len(diags) == 0 {
		t.Fatal("expected diagnostics for unknown relation target")
	}
}

func TestUnusedDataReportsUnreferencedData(t *testing.T) {
	model := NewThreeElementModel("probe", "Probe")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit", Operations: []FunctionOperation{{DataID: "order", Operations: "CRUD"}}}}
	if got := model.UnusedData(); len(got) != 0 {
		t.Fatalf("expected no unused data, got %v", got)
	}
}

func TestSameNameConflicts(t *testing.T) {
	model := NewThreeElementModel("probe", "Probe")
	model.DataElements = []DataElement{{ID: "order1", Name: "Order"}, {ID: "order2", Name: "Order"}}
	conflicts := model.SameNameConflicts()
	if len(conflicts) == 0 {
		t.Fatal("expected same-name conflicts")
	}
}

// TestModelFingerprintPreservesAuthorityAndIgnoresItsOwnDigest pins both
// invariants: calculating a structural fingerprint cannot remove the element's
// authority limits, and storing the derived hash cannot change that same hash.
func TestModelFingerprintPreservesAuthorityAndIgnoresItsOwnDigest(t *testing.T) {
	model := NewThreeElementModel("probe", "Fingerprint purity")
	model.DataElements = []DataElement{{
		ID: "order", Name: "Order",
		AuthorityLimits: []string{"data-owner-approval"},
	}}
	model.FunctionElements = []FunctionElement{{
		ID: "submit", Name: "Submit",
		AuthorityLimits: []string{"function-review-required"},
	}}
	model.BusinessElements = []BusinessElement{{
		ID: "approve", Name: "Approve",
		AuthorityLimits: []string{"business-scope-limited"},
	}}
	model.Diagnostics = []Diagnostic{{Code: "OBSERVED", Severity: "info", Message: "transient"}}

	before, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	first := ModelFingerprint(model)
	second := ModelFingerprint(model)
	if first != second {
		t.Fatalf("repeated fingerprint changed: %q != %q", first, second)
	}
	after, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("fingerprinting mutated the model:\nbefore: %s\nafter: %s", before, after)
	}

	model.ModelFingerprint = first
	if got := ModelFingerprint(model); got != first {
		t.Fatalf("fingerprint self-reference: got %q after storing %q", got, first)
	}

	// Authority and observation are outside the structural fingerprint, but
	// their values must stay intact for downstream consumers and policy checks.
	model.DataElements[0].AuthorityLimits[0] = "data-owner-revoked"
	model.Diagnostics[0].Message = "different observation"
	if got := ModelFingerprint(model); got != first {
		t.Fatalf("transient/authority-only change altered structural hash: %q != %q", got, first)
	}
	model.DataElements[0].Name = "Changed Order"
	if got := ModelFingerprint(model); got == first {
		t.Fatal("structural model change failed to alter fingerprint")
	}
}

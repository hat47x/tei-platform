package semantic

import "testing"

func fingerprintWorld() []Namespace {
	return []Namespace{
		{
			ID: "sales",
			Data: []Data{{
				ID: "sales.order",
				Attributes: []Attribute{
					{ID: "sales.order.order_id", TypeRef: "std.identifier"},
					{ID: "sales.order.customer_id", TypeRef: "customer.customer_id"},
				},
				Relations: []Relation{{ID: "sales.order.customer", Target: "customer.customer"}},
			}},
		},
		{
			ID: "customer",
			Data: []Data{{
				ID:         "customer.customer",
				Attributes: []Attribute{{ID: "customer.customer.customer_id", TypeRef: "std.identifier"}},
			}},
		},
	}
}

func TestFingerprintIsStableAndOrderIndependent(t *testing.T) {
	first, err := Fingerprint(fingerprintWorld())
	if err != nil {
		t.Fatal(err)
	}
	second, err := Fingerprint(fingerprintWorld())
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("fingerprint is not stable: %s != %s", first, second)
	}
	reordered := fingerprintWorld()
	reordered[0], reordered[1] = reordered[1], reordered[0]
	third, err := Fingerprint(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if first != third {
		t.Fatalf("fingerprint depends on namespace order: %s != %s", first, third)
	}
}

func TestFingerprintChangesWhenMeaningChanges(t *testing.T) {
	before, err := Fingerprint(fingerprintWorld())
	if err != nil {
		t.Fatal(err)
	}
	changed := fingerprintWorld()
	changed[0].Data[0].Attributes = append(changed[0].Data[0].Attributes, Attribute{ID: "sales.order.status", TypeRef: "std.code"})
	after, err := Fingerprint(changed)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("adding a semantic attribute did not change the fingerprint")
	}
}

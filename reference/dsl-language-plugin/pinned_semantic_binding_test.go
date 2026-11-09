package languagereference

import (
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
)

func TestPinnedSemanticBinding(t *testing.T) {
	ns := semantic.Namespace{ID: core.CanonicalID("sales"), Data: []semantic.Data{{ID: core.CanonicalID("sales.order")}}}
	pin := OperatorContract{ID: "std.where", Version: "1", Fingerprint: "where-v1", Effect: "pure", InputType: "rows/sales.order", OutputType: "rows/sales.order"}
	calls := []PinnedCall{{Contract: pin, InputType: "rows/sales.order"}}
	out, err := BindPinnedSemanticInput(ns, "sales.order", calls, []OperatorContract{pin})
	if err != nil || out != "rows/sales.order" {
		t.Fatalf("got %q: %v", out, err)
	}
	for _, tc := range []struct {
		name    string
		ns      semantic.Namespace
		catalog []OperatorContract
	}{
		{"missing", semantic.Namespace{ID: ns.ID}, []OperatorContract{pin}},
		{"duplicate", semantic.Namespace{ID: ns.ID, Data: []semantic.Data{ns.Data[0], ns.Data[0]}}, []OperatorContract{pin}},
		{"unaccepted operator", ns, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := BindPinnedSemanticInput(tc.ns, "sales.order", calls, tc.catalog); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

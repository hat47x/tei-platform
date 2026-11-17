package main

import (
	"testing"

	"tei/internal/core"
	"tei/internal/realization"
)

// The report must distinguish a binding that made no statement from one that
// made a statement this entry did not check. Collapsing both into "no field"
// is how a declaration gets read as a measurement.
func TestDestinationDeclarationFrom(t *testing.T) {
	t.Run("undeclared binding reports nothing", func(t *testing.T) {
		got := destinationDeclarationFrom(realization.HeaderDetailContractQueryBinding{})
		if got != nil {
			t.Fatalf("undeclared binding produced %+v", got)
		}
	})

	t.Run("covers every row, unchecked", func(t *testing.T) {
		got := destinationDeclarationFrom(realization.HeaderDetailContractQueryBinding{
			CoversRelation: "sales.order.lines",
		})
		if got == nil {
			t.Fatal("a declaration was dropped")
		}
		if got.CoversRelation != "sales.order.lines" || got.CoversSome {
			t.Fatalf("declaration = %+v", got)
		}
		// This entry does not cross-check the statement, so a reader must not be
		// able to read the presence of the field as a confirmation.
		if got.Checked {
			t.Fatal("an unchecked declaration reported itself as checked")
		}
	})

	t.Run("covers some rows, unchecked", func(t *testing.T) {
		got := destinationDeclarationFrom(realization.HeaderDetailContractQueryBinding{
			CoversSome: true,
		})
		if got == nil {
			t.Fatal("a declaration was dropped")
		}
		if !got.CoversSome || got.CoversRelation != "" {
			t.Fatalf("declaration = %+v", got)
		}
		if got.Checked {
			t.Fatal("an unchecked declaration reported itself as checked")
		}
	})

	// The two declared states must stay distinguishable in the report, because
	// "covers every row" and "covers some" license different conclusions.
	t.Run("the two declarations differ", func(t *testing.T) {
		all := destinationDeclarationFrom(realization.HeaderDetailContractQueryBinding{
			CoversRelation: core.CanonicalID("sales.order.lines"),
		})
		some := destinationDeclarationFrom(realization.HeaderDetailContractQueryBinding{
			CoversSome: true,
		})
		if all == nil || some == nil {
			t.Fatal("a declaration was dropped")
		}
		if all.CoversRelation == some.CoversRelation && all.CoversSome == some.CoversSome {
			t.Fatalf("the two declarations became indistinguishable: %+v %+v", all, some)
		}
	})
}

package main

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/realization"
)

// The report must distinguish a binding that made no statement from one that
// made a statement this entry did not check. Collapsing both into "no field"
// is how a declaration gets read as a measurement.
func TestDestinationDeclarationFrom(t *testing.T) {
	t.Run("undeclared binding reports nothing", func(t *testing.T) {
		got, err := destinationDeclarationFrom(realization.HeaderDetailContractQueryBinding{})
		if err != nil {
			t.Fatalf("undeclared binding refused: %v", err)
		}
		if got != nil {
			t.Fatalf("undeclared binding produced %+v", got)
		}
	})

	t.Run("covers every row, unchecked", func(t *testing.T) {
		got, err := destinationDeclarationFrom(realization.HeaderDetailContractQueryBinding{
			CoversRelation: "sales.order.lines",
		})
		if err != nil {
			t.Fatalf("a single declaration was refused: %v", err)
		}
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
		got, err := destinationDeclarationFrom(realization.HeaderDetailContractQueryBinding{
			CoversSome: true,
		})
		if err != nil {
			t.Fatalf("a single declaration was refused: %v", err)
		}
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
		all, err := destinationDeclarationFrom(realization.HeaderDetailContractQueryBinding{
			CoversRelation: core.CanonicalID("sales.order.lines"),
		})
		if err != nil {
			t.Fatalf("all-rows declaration refused: %v", err)
		}
		some, err := destinationDeclarationFrom(realization.HeaderDetailContractQueryBinding{
			CoversSome: true,
		})
		if err != nil {
			t.Fatalf("some-rows declaration refused: %v", err)
		}
		if all == nil || some == nil {
			t.Fatal("a declaration was dropped")
		}
		if all.CoversRelation == some.CoversRelation && all.CoversSome == some.CoversSome {
			t.Fatalf("the two declarations became indistinguishable: %+v %+v", all, some)
		}
	})

	// Both fields can be set at once on the struct even though the static
	// validator rejects the combination, and the view path runs no validator.
	// The read is refused rather than answered: labelling the contradiction and
	// continuing would decide rules against a binding that contradicts itself
	// about the very scope the rule is asked over.
	t.Run("both stated is refused", func(t *testing.T) {
		got, err := destinationDeclarationFrom(realization.HeaderDetailContractQueryBinding{
			CoversRelation: "sales.order.lines",
			CoversSome:     true,
		})
		if err == nil {
			t.Fatalf("a contradictory binding was accepted: %+v", got)
		}
		if got != nil {
			t.Fatalf("a contradictory binding produced a declaration anyway: %+v", got)
		}
		if !strings.Contains(err.Error(), "sales.order.lines") {
			t.Fatalf("the refusal does not name the Relation it contradicts: %v", err)
		}
	})
}

package workspace

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/ir"
)

// OperatorImplementationSelection is one `implementations:` entry of the
// manifest, as written: the operator and the name of the implementation the
// Workspace selects for it.
//
//	implementations:
//	  - operator: sales.non_negative
//	    use: tei.number.non-negative
//
// The entry is a choice of the composition. It declares no operator and
// changes no meaning: the same documents with another selection, or with
// none, load to the same World.
type OperatorImplementationSelection struct {
	Operator       string `json:"operator"`
	Implementation string `json:"implementation"`
}

func validateOperatorImplementationSelections(selections []OperatorImplementationSelection) error {
	seen := map[string]bool{}
	for _, selection := range selections {
		if _, err := core.ParseCanonicalID(selection.Operator); err != nil || selection.Operator == "" {
			return fmt.Errorf("implementation entry has invalid operator %q", selection.Operator)
		}
		if !validAuthoringLanguageWord(selection.Implementation) {
			return fmt.Errorf("implementation entry for operator %s needs 'use: <implementation-id>'", selection.Operator)
		}
		if seen[selection.Operator] {
			return fmt.Errorf("operator %s has more than one implementation entry", selection.Operator)
		}
		seen[selection.Operator] = true
	}
	return nil
}

func checkOperatorImplementationSelections(selections []OperatorImplementationSelection, world ir.World) error {
	declared := map[string]bool{}
	for _, namespace := range world.Semantics {
		for _, operator := range namespace.Operators {
			declared[string(operator.ID)] = true
		}
	}
	for _, selection := range selections {
		if !declared[selection.Operator] {
			return fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_UNBOUND: an implementation is selected for %s, but no source declares that operator", selection.Operator)
		}
	}
	return nil
}

package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

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

// FingerprintOperatorImplementationSelections records the exact manifest choice
// independently from the Canonical World or the Plugin Lock. Ordering of
// distinct declarations is presentation only; callers use validated selections
// from Workspace.Load. The empty selection has one canonical representation.
func FingerprintOperatorImplementationSelections(selected []OperatorImplementationSelection) (string, error) {
	ordered := append(make([]OperatorImplementationSelection, 0, len(selected)), selected...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Operator != ordered[j].Operator {
			return ordered[i].Operator < ordered[j].Operator
		}
		return ordered[i].Implementation < ordered[j].Implementation
	})
	payload, err := json.Marshal(ordered)
	if err != nil {
		return "", fmt.Errorf("fingerprint selected Operator implementations: %w", err)
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

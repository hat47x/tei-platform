// Package iomapping's selected Semantic preflight is a design-time check.
// It does not authenticate a Semantic Set or authorize a plugin invocation.
package iomapping

import (
	"sort"
	"strings"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/validation"
)

// InspectWithSelectedSemanticView checks the declared mappings against an
// explicitly supplied Semantic view before preserving Inspect's candidate.
// It reuses TEI's existing Semantic validator, type identities, and
// Composition Registry. It does NOT resolve a new type catalog, fingerprint
// declarations, or establish Publisher / Project Acceptance authority.
//
// The caller must obtain the selected namespaces from an independently
// qualified source. A syntactically valid caller-provided view is not itself
// an Accepted Semantic Set.
func InspectWithSelectedSemanticView(
	registry *composition.Registry,
	plan Plan,
	selected []semantic.Namespace,
) Report {
	unknown := func(code, slot, field string) Report {
		return Report{
			Verdict:     composition.ProposalUnknown,
			Diagnostics: []Diagnostic{{Code: code, Slot: slot, Field: field}},
		}
	}
	if len(selected) == 0 {
		return unknown("selected-semantic-view-missing", "", "")
	}

	// Validate the Semantic view alone, as in the existing E4d preflight.
	// This is deliberately not an authority or acceptance verification.
	view := ir.World{Semantics: append([]semantic.Namespace(nil), selected...)}
	if validation.HasErrors(validation.Validate(view)) {
		return unknown("selected-semantic-view-invalid", "", "")
	}

	report := Inspect(registry, plan)
	if report.Verdict != composition.ProposalCandidate {
		return report
	}

	// A temporary local index over the validated caller-selected view is not
	// a second registry or resolver. No namespaces are auto-imported.
	definitions := make(map[core.CanonicalID]semantic.Type)
	for _, namespace := range selected {
		for _, typ := range namespace.Types {
			definitions[typ.ID] = typ
		}
	}
	slots := make(map[string]Slot, len(plan.Slots))
	for _, slot := range plan.Slots {
		slots[slot.ID] = slot
	}
	add := func(code, slot, field string) {
		report.Diagnostics = append(report.Diagnostics, Diagnostic{
			Code: code, Slot: slot, Field: field,
		})
		report.Verdict = composition.ProposalUnknown
		report.Order = nil
		report.ContributionOrder = nil
	}

	// Only connected fields are required to have a SemanticRef; unrelated
	// outputs remain supported by the existing E2 fixture.
	for _, mapping := range plan.Mappings {
		for _, endpoint := range []struct {
			ref   Ref
			input bool
		}{{mapping.From, false}, {mapping.To, true}} {
			field, exists := lookup(slots, endpoint.ref, endpoint.input)
			if !exists {
				add("selected-semantic-field-unresolved", endpoint.ref.Slot, endpoint.ref.Field)
				continue
			}
			id := field.Type.SemanticRef
			if id == "" {
				add("selected-semantic-ref-required", endpoint.ref.Slot, endpoint.ref.Field)
				continue
			}
			if _, ok := definitions[id]; !ok {
				add("selected-semantic-type-unknown", endpoint.ref.Slot, endpoint.ref.Field)
				continue
			}
			primitive, ok := selectedPrimitive(id, definitions)
			if !ok {
				add("selected-semantic-type-unresolved", endpoint.ref.Slot, endpoint.ref.Field)
				continue
			}
			// std's existing primitive IDs provide a known nominal label.
			// Do not infer Representation or an arbitrary user primitive's
			// wire format from a Semantic alias or Symbol.
			if strings.HasPrefix(string(primitive), "std.") &&
				field.Type.Name != strings.TrimPrefix(string(primitive), "std.") {
				add("selected-semantic-representation-unresolved", endpoint.ref.Slot, endpoint.ref.Field)
			}
		}
	}
	sort.Slice(report.Diagnostics, func(i, j int) bool {
		a, b := report.Diagnostics[i], report.Diagnostics[j]
		if a.Slot != b.Slot {
			return a.Slot < b.Slot
		}
		if a.Field != b.Field {
			return a.Field < b.Field
		}
		return a.Code < b.Code
	})
	return report
}

func selectedPrimitive(
	id core.CanonicalID,
	definitions map[core.CanonicalID]semantic.Type,
) (core.CanonicalID, bool) {
	seen := map[core.CanonicalID]bool{}
	for !seen[id] {
		seen[id] = true
		definition, found := definitions[id]
		if !found {
			return "", false
		}
		switch definition.Kind {
		case semantic.TypePrimitive:
			return id, true
		case semantic.TypeAlias:
			id = definition.Underlying
		default:
			return "", false
		}
	}
	return "", false
}

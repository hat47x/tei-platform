// Package iomapping is a design-time experiment. It neither invokes providers
// nor establishes schema authenticity, acceptance, readiness or authority.
package iomapping

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/composition"
	"tei/internal/core"
)

type Multiplicity string

const (
	One      Multiplicity = "one"
	Optional Multiplicity = "optional"
	List     Multiplicity = "list"
	Set      Multiplicity = "set"
)

// Type uses deliberately small nominal metadata, not a production type system.
// Unit must be explicit, including "none" for dimensionless fields.
// SemanticRef is the exact canonical identity of an already selected semantic
// type definition. This reference does not resolve or authenticate definitions;
// the caller must obtain it from the Host's validated semantic set. In particular,
// aliases with the same underlying representation remain distinct.
type Type struct {
	Name, Unit, IdentityDomain string
	SemanticRef                core.CanonicalID
}
type Field struct {
	ID             string
	Type           Type
	Multiplicity   Multiplicity
	Nullable       bool
	Classification string
}
type Slot struct {
	ID                       string
	Contract                 composition.Contract
	ProviderID               string
	Multiplicity             Multiplicity
	WhenAbsent               string
	Critical                 bool
	Inputs, Outputs, Context []Field
	Outcomes                 []string
}

// Ref includes the complete contract and local slot. Equal field names never
// establish an implicit connection between contracts or independent consumers.
type Ref struct {
	Slot     string
	Contract composition.Contract
	Field    string
}
type Mapping struct {
	From, To       Ref
	MissingOutcome string
	Operator       string
	ReadsExternal  bool
}
type Ordering struct{ Before, After, Reason string }

// Contributions are explicitly enumerated; Registry still chooses/assesses each
// pinned provider. OrderKey is a declared key, not the load order or provider ID.
type Contribution struct{ Slot, ProviderID, OrderKey string }
type Plan struct {
	Scope         string
	Slots         []Slot
	Mappings      []Mapping
	After         []Ordering
	Contributions []Contribution
}
type Diagnostic struct{ Code, Slot, Field string }
type Report struct {
	Verdict           composition.ProposalVerdict
	Wiring            composition.WiringSetFinding
	Diagnostics       []Diagnostic
	Order             []string
	ContributionOrder map[string][]string
}
type Compatibility string

const (
	Identical    Compatibility = "identical"
	Widening     Compatibility = "safe-widening"
	Conditional  Compatibility = "conditional"
	Incompatible Compatibility = "incompatible"
	Unknown      Compatibility = "unknown"
)

// Compare is conservative. Integer-to-decimal assumes an exact unbounded
// decimal representation; bounded precision and conversion operators await E3.
func Compare(from, to Field) Compatibility {
	if !complete(from) || !complete(to) {
		return Unknown
	}
	// Ref-backed and legacy nominal fields are not interchangeable. A shared
	// primitive representation does not make two semantic aliases compatible.
	// This checks declared identities only; it does not assert that either
	// definition was issued by an authorized namespace owner.
	if from.Type.SemanticRef != "" || to.Type.SemanticRef != "" {
		if from.Type.SemanticRef == "" || to.Type.SemanticRef == "" {
			return Unknown
		}
		if from.Type.SemanticRef != to.Type.SemanticRef {
			return Incompatible
		}
		// Contradictory representation claims require a verified type
		// projection. Do not infer a numeric conversion from the type name.
		if from.Type.Name != to.Type.Name {
			return Unknown
		}
	}
	if from.Type.Unit != to.Type.Unit || from.Type.IdentityDomain != to.Type.IdentityDomain {
		return Incompatible
	}
	if from.Classification != to.Classification && from.Classification != "public" {
		return Incompatible
	}
	if from.Multiplicity == List || from.Multiplicity == Set || to.Multiplicity == List || to.Multiplicity == Set {
		if from.Multiplicity != to.Multiplicity {
			return Incompatible
		}
	}
	if from.Nullable && !to.Nullable {
		return Conditional
	}
	if from.Multiplicity == Optional && to.Multiplicity == One {
		return Conditional
	}
	if from.Type.Name == "decimal" && to.Type.Name == "integer" {
		return Conditional
	}
	if from.Type.Name != to.Type.Name {
		if from.Type.Name == "integer" && to.Type.Name == "decimal" {
			return Widening
		}
		return Incompatible
	}
	if from.Multiplicity == One && to.Multiplicity == Optional {
		return Widening
	}
	return Identical
}

func complete(f Field) bool {
	if f.ID == "" || f.Type.Name == "" || f.Type.Unit == "" {
		return false
	}
	if f.Type.SemanticRef != "" {
		if _, err := core.ParseCanonicalID(string(f.Type.SemanticRef)); err != nil {
			return false
		}
	}
	if f.Type.Name == "identity" && f.Type.IdentityDomain == "" {
		return false
	}
	if f.Classification != "public" && f.Classification != "personal" && f.Classification != "secret" {
		return false
	}
	return f.Multiplicity == One || f.Multiplicity == Optional || f.Multiplicity == List || f.Multiplicity == Set
}

// Inspect delegates all provider selection to one atomic Registry observation.
// Order is only a proposed data/effect order, never activation or readiness.
func Inspect(registry *composition.Registry, plan Plan) Report {
	out := Report{Verdict: composition.ProposalCandidate, ContributionOrder: map[string][]string{}}
	add := func(code, slot, field string, unknown bool) {
		out.Diagnostics = append(out.Diagnostics, Diagnostic{code, slot, field})
		if !unknown {
			out.Verdict = composition.ProposalConflict
		} else if out.Verdict == composition.ProposalCandidate {
			out.Verdict = composition.ProposalUnknown
		}
	}
	slots := map[string]Slot{}
	for _, slot := range plan.Slots {
		if slot.ID == "" || slots[slot.ID].ID != "" {
			add("invalid-or-duplicate-slot", slot.ID, "", false)
			continue
		}
		slots[slot.ID] = slot
		if slot.Multiplicity != One && slot.Multiplicity != Optional && slot.Multiplicity != Set {
			add("invalid-slot-multiplicity", slot.ID, "", false)
		}
		if slot.Critical && slot.Multiplicity != One {
			add("critical-requirement-must-be-single", slot.ID, "", false)
		}
		if slot.Multiplicity == Optional && (slot.Critical || strings.TrimSpace(slot.WhenAbsent) == "") {
			add("invalid-optional-policy", slot.ID, "", false)
		}
		if slot.Contract.ID == "" || slot.Contract.Version == "" || slot.Contract.Shape == "" {
			add("incomplete-contract", slot.ID, "", true)
		}
		seen := map[string]bool{}
		for _, group := range [][]Field{slot.Inputs, slot.Outputs, slot.Context} {
			for _, field := range group {
				if field.ID == "" || seen[field.ID] {
					add("duplicate-or-empty-field", slot.ID, field.ID, false)
				}
				seen[field.ID] = true
				if !complete(field) {
					add("incomplete-field", slot.ID, field.ID, true)
				}
			}
		}
	}
	ids := keys(slots)
	contributions := map[string][]Contribution{}
	for _, c := range plan.Contributions {
		if slot, ok := slots[c.Slot]; !ok || slot.Multiplicity != Set {
			add("invalid-contribution-slot", c.Slot, "", false)
			continue
		}
		contributions[c.Slot] = append(contributions[c.Slot], c)
	}
	var requests []composition.WiringRequirement
	var requestSlots []string
	for _, id := range ids {
		slot := slots[id]
		if slot.Multiplicity == Set {
			if slot.ProviderID != "" {
				add("collection-single-provider", id, "", false)
			}
			cs := contributions[id]
			sort.Slice(cs, func(i, j int) bool {
				if cs[i].OrderKey == cs[j].OrderKey {
					return cs[i].ProviderID < cs[j].ProviderID
				}
				return cs[i].OrderKey < cs[j].OrderKey
			})
			seenProvider, seenOrder := map[string]bool{}, map[string]bool{}
			for i, c := range cs {
				if c.ProviderID == "" || c.OrderKey == "" || seenProvider[c.ProviderID] || seenOrder[c.OrderKey] {
					add("contribution-collision", id, "", false)
				}
				seenProvider[c.ProviderID], seenOrder[c.OrderKey] = true, true
				out.ContributionOrder[id] = append(out.ContributionOrder[id], c.ProviderID)
				requests = append(requests, composition.WiringRequirement{Contract: slot.Contract, Scope: plan.Scope, ProviderID: c.ProviderID, SlotID: fmt.Sprintf("collection:%q:%d", id, i)})
				requestSlots = append(requestSlots, id)
			}
		} else {
			requests = append(requests, composition.WiringRequirement{Contract: slot.Contract, Scope: plan.Scope, ProviderID: slot.ProviderID, SlotID: fmt.Sprintf("single:%q", id)})
			requestSlots = append(requestSlots, id)
		}
	}
	// Set-only/empty plans still need a registry and an explicit scope. They
	// establish no provider eligibility and cannot grant runtime authority.
	if registry == nil || plan.Scope == "" {
		add("missing-registry-or-scope", "", "", true)
	}
	out.Wiring = registry.InspectWiringSet(requests)
	absent := map[string]bool{}
	for i, f := range out.Wiring.Findings {
		id := requestSlots[i]
		slot := slots[id]
		if f.Verdict == composition.ProposalCandidate {
			continue
		}
		if slot.Multiplicity == Optional && f.Reason == composition.ReasonNoKnownProvider && slot.ProviderID == "" && len(f.Candidates) == 0 {
			absent[id] = true
			continue
		}
		add(string(f.Reason), id, "", f.Verdict != composition.ProposalConflict)
	}
	edges := map[string]map[string]bool{}
	for _, id := range ids {
		edges[id] = map[string]bool{}
	}
	mapped := map[Ref]bool{}
	for _, m := range plan.Mappings {
		if mapped[m.To] {
			add("duplicate-destination", m.To.Slot, m.To.Field, false)
		}
		mapped[m.To] = true
		source, sourceOK := lookup(slots, m.From, false)
		destination, destinationOK := lookup(slots, m.To, true)
		if slot, ok := slots[m.To.Slot]; ok && slot.Contract == m.To.Contract {
			for _, field := range slot.Context {
				if field.ID == m.To.Field {
					add("context-override", m.To.Slot, m.To.Field, false)
				}
			}
		}
		if !sourceOK || !destinationOK {
			add("unknown-field-reference", m.To.Slot, m.To.Field, true)
			continue
		}
		if m.MissingOutcome != "" {
			declared := false
			for _, outcome := range slots[m.To.Slot].Outcomes {
				if outcome == m.MissingOutcome {
					declared = true
				}
			}
			if !declared {
				add("undeclared-missing-outcome", m.To.Slot, m.To.Field, false)
			}
		}
		if slots[m.From.Slot].Multiplicity == Set || slots[m.To.Slot].Multiplicity == Set {
			add("collection-flow-needs-explicit-reduction", m.To.Slot, m.To.Field, true)
		}
		if absent[m.From.Slot] || absent[m.To.Slot] {
			add("mapped-provider-absent", m.To.Slot, m.To.Field, true)
		}
		if m.ReadsExternal {
			add("external-transform-forbidden", m.To.Slot, m.To.Field, false)
		}
		if m.Operator != "" && m.Operator != "identity" {
			add("operator-not-evaluated", m.To.Slot, m.To.Field, true)
		}
		switch Compare(source, destination) {
		case Unknown:
			add("type-undetermined", m.To.Slot, m.To.Field, true)
		case Incompatible:
			add("type-incompatible", m.To.Slot, m.To.Field, false)
		case Conditional:
			// E2 only supports explicit missing outcomes. Lossy numeric and null
			// conversions remain blocked, even with a named outcome.
			if source.Type != destination.Type || source.Nullable && !destination.Nullable || m.MissingOutcome == "" {
				add("explicit-conversion-or-missing-policy-required", m.To.Slot, m.To.Field, false)
			}
		}
		edges[m.From.Slot][m.To.Slot] = true
	}
	for _, id := range ids {
		if absent[id] {
			continue
		}
		for _, f := range slots[id].Inputs {
			ref := Ref{id, slots[id].Contract, f.ID}
			if f.Multiplicity == One && !mapped[ref] {
				add("required-input-unmapped", id, f.ID, false)
			}
		}
	}
	for _, a := range plan.After {
		_, beforeOK := slots[a.Before]
		_, afterOK := slots[a.After]
		if strings.TrimSpace(a.Reason) == "" || !beforeOK || !afterOK {
			add("invalid-ordering", a.After, "", false)
			continue
		}
		edges[a.Before][a.After] = true
	}
	out.Order = order(ids, edges)
	if len(out.Order) != len(ids) {
		add("data-order-cycle", "", "", false)
	}
	sort.Slice(out.Diagnostics, func(i, j int) bool {
		a, b := out.Diagnostics[i], out.Diagnostics[j]
		if a.Slot != b.Slot {
			return a.Slot < b.Slot
		}
		if a.Field != b.Field {
			return a.Field < b.Field
		}
		return a.Code < b.Code
	})
	// Never expose a usable order for a conflicting/unknown declaration.
	if out.Verdict != composition.ProposalCandidate {
		out.Order = nil
		out.ContributionOrder = nil
	}
	return out
}

func lookup(slots map[string]Slot, ref Ref, input bool) (Field, bool) {
	slot, ok := slots[ref.Slot]
	if !ok || slot.Contract != ref.Contract {
		return Field{}, false
	}
	fields := slot.Outputs
	if input {
		fields = slot.Inputs
	}
	for _, f := range fields {
		if f.ID == ref.Field {
			return f, true
		}
	}
	return Field{}, false
}
func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
func order(ids []string, edges map[string]map[string]bool) []string {
	degrees := map[string]int{}
	for _, tos := range edges {
		for to := range tos {
			degrees[to]++
		}
	}
	var out []string
	done := map[string]bool{}
	for len(out) < len(ids) {
		found := ""
		for _, id := range ids {
			if !done[id] && degrees[id] == 0 {
				found = id
				break
			}
		}
		if found == "" {
			return nil
		}
		done[found] = true
		out = append(out, found)
		for to := range edges[found] {
			degrees[to]--
		}
	}
	return out
}

// Package semanticdistribution contains read-only reference experiments.
// This file inventories declared cross-namespace references without selecting
// providers or introducing a second dependency resolver.
package semanticdistribution

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

// NamespaceDependencyUse is one authored reference from a semantic declaration.
// Namespace is the longest explicitly available namespace prefix, or empty
// when no available namespace owns the prefix. It is not a symbol-resolution
// verdict, an import instruction, or an authorization decision.
type NamespaceDependencyUse struct {
	SourceKind string `json:"sourceKind"`
	SourceID   string `json:"sourceId"`
	Field      string `json:"field"`
	TargetID   string `json:"targetId"`
	Namespace  string `json:"namespace,omitempty"`
}

// NamespaceDependencyAudit is a read-only inventory for an explicitly selected
// data-only namespace. Unmatched is a review candidate, not proof that a
// missing namespace exists: a malformed local reference is also unmatched.
// This inspection does not traverse dependencies or validate target symbols.
type NamespaceDependencyAudit struct {
	External  []NamespaceDependencyUse `json:"external,omitempty"`
	Unmatched []NamespaceDependencyUse `json:"unmatched,omitempty"`
}

// AuditNamespaceDependencies inspects references in exactly one namespace.
// Available namespaces are caller-supplied, already selected semantic models;
// they are NOT discovered, downloaded, verified, accepted, or made executable
// here. The namespace being inspected and std are always considered local or
// prelude scopes. Callers must separately run canonical model validation to
// check actual target symbol existence, kind, and semantic consistency.
func AuditNamespaceDependencies(ns semantic.Namespace, available []semantic.Namespace) (NamespaceDependencyAudit, error) {
	own := string(ns.ID)
	if _, err := core.ParseCanonicalID(own); err != nil {
		return NamespaceDependencyAudit{}, fmt.Errorf("namespace being audited: %w", err)
	}
	namespaces := []string{own}
	if own != "std" {
		namespaces = append(namespaces, "std")
	}
	seen := map[string]bool{}
	for _, candidate := range available {
		id := string(candidate.ID)
		if _, err := core.ParseCanonicalID(id); err != nil {
			return NamespaceDependencyAudit{}, fmt.Errorf("available namespace: %w", err)
		}
		if seen[id] {
			return NamespaceDependencyAudit{}, fmt.Errorf("duplicate available namespace %q", id)
		}
		seen[id] = true
		if id != own && id != "std" {
			namespaces = append(namespaces, id)
		}
	}
	// A nested namespace (e.g. os.process) must win over its os parent.
	sort.Slice(namespaces, func(i, j int) bool {
		if len(namespaces[i]) == len(namespaces[j]) {
			return namespaces[i] < namespaces[j]
		}
		return len(namespaces[i]) > len(namespaces[j])
	})
	var out NamespaceDependencyAudit
	add := func(kind, sourceID, field string, target core.CanonicalID) {
		id := string(target)
		if id == "" {
			return // Optional unset references are not dependencies.
		}
		var scope string
		for _, candidate := range namespaces {
			if id == candidate || strings.HasPrefix(id, candidate+".") {
				scope = candidate
				break
			}
		}
		use := NamespaceDependencyUse{
			SourceKind: kind, SourceID: sourceID, Field: field,
			TargetID: id, Namespace: scope,
		}
		switch scope {
		case own, "std":
			// Owned and standard references are not external dependencies.
		case "":
			out.Unmatched = append(out.Unmatched, use)
		default:
			out.External = append(out.External, use)
		}
	}
	for _, typ := range ns.Types {
		add("semantic-type", string(typ.ID), "underlying", typ.Underlying)
	}
	for _, data := range ns.Data {
		for _, attr := range data.Attributes {
			add("semantic-attribute", string(attr.ID), "type", attr.TypeRef)
		}
		for _, rel := range data.Relations {
			relID := string(rel.ID)
			add("semantic-relation", relID, "target", rel.Target)
			for i, key := range rel.Keys {
				prefix := fmt.Sprintf("keys[%d].", i)
				add("semantic-relation", relID, prefix+"sourceAttribute", key.SourceAttribute)
				add("semantic-relation", relID, prefix+"targetAttribute", key.TargetAttribute)
			}
		}
		for _, key := range data.CandidateKeys {
			for i, attr := range key.Attributes {
				add("semantic-candidate-key", string(key.ID), fmt.Sprintf("attributes[%d]", i), attr)
			}
		}
		for _, dependency := range data.FunctionalDependencies {
			for i, attr := range dependency.Determinants {
				add("semantic-functional-dependency", string(dependency.ID), fmt.Sprintf("determinants[%d]", i), attr)
			}
			for i, attr := range dependency.Dependents {
				add("semantic-functional-dependency", string(dependency.ID), fmt.Sprintf("dependents[%d]", i), attr)
			}
		}
	}
	for _, cap := range ns.Capabilities {
		id := string(cap.ID)
		for _, input := range cap.Inputs {
			add("semantic-capability", id, "input:"+string(input.Name), input.Type)
		}
		add("semantic-capability", id, "output", cap.Output)
		for _, outcome := range cap.Outcomes {
			add("semantic-capability", id, "outcome:"+string(outcome.Name), outcome.Payload)
		}
		if cap.Effect != nil {
			add("semantic-capability", id, "effect.data", cap.Effect.Data)
		}
	}
	for _, rule := range ns.Validations {
		id := string(rule.ID)
		add("semantic-validation", id, "attribute", rule.Attribute)
		add("semantic-validation", id, "relation", rule.Relation)
	}
	less := func(a, b NamespaceDependencyUse) bool {
		if a.SourceKind != b.SourceKind {
			return a.SourceKind < b.SourceKind
		}
		if a.SourceID != b.SourceID {
			return a.SourceID < b.SourceID
		}
		if a.Field != b.Field {
			return a.Field < b.Field
		}
		return a.TargetID < b.TargetID
	}
	sort.Slice(out.External, func(i, j int) bool { return less(out.External[i], out.External[j]) })
	sort.Slice(out.Unmatched, func(i, j int) bool { return less(out.Unmatched[i], out.Unmatched[j]) })
	return out, nil
}

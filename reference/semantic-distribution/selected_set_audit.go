package semanticdistribution

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

// SelectedNamespaceDependency is an observed reference from one explicitly
// supplied namespace to another. It does not select or authorize a provider.
type SelectedNamespaceDependency struct {
	From       string                   `json:"from"`
	To         string                   `json:"to"`
	References []NamespaceDependencyUse `json:"references"`
}

// UnmatchedNamespaceReference identifies a reference whose target prefix does
// not belong to any selected namespace or the implicit std prelude. It may be
// a missing selection OR an invalid/typoed symbol; it is not proof of absence.
type UnmatchedNamespaceReference struct {
	From      string                 `json:"from"`
	Reference NamespaceDependencyUse `json:"reference"`
}

// NamespaceCycleCandidate is a strongly connected group in the observed
// reference graph, not a determination of invalid semantic modeling.
// Relations between domain data may legitimately refer in both directions.
type NamespaceCycleCandidate struct {
	Namespaces []string `json:"namespaces"`
}

// SelectedNamespaceSetAudit is a derived read-only report. Only the explicitly
// passed namespaces participate; no recursive provider discovery, import
// resolution, closure construction, Source Index mutation, or acceptance occurs.
type SelectedNamespaceSetAudit struct {
	Selected        []string                      `json:"selected"`
	Dependencies    []SelectedNamespaceDependency `json:"dependencies,omitempty"`
	Unmatched       []UnmatchedNamespaceReference `json:"unmatched,omitempty"`
	CycleCandidates []NamespaceCycleCandidate     `json:"cycleCandidates,omitempty"`
}

// AuditSelectedNamespaceSet inventories direct dependencies and graph cycles
// in an already selected namespace set. Duplicate logical namespaces are
// rejected before inspecting their contents; neither same-content duplicates
// nor differing providers are implicitly merged or given precedence.
//
// Namespace-level prefix matches are not symbol-level resolution. Consumers
// must separately run canonical validation and provider/artifact checks.
// Cycles are reported for review, never treated as unconditional errors.
func AuditSelectedNamespaceSet(selected []semantic.Namespace) (SelectedNamespaceSetAudit, error) {
	byID := make(map[string]semantic.Namespace, len(selected))
	names := make([]string, 0, len(selected))
	for _, ns := range selected {
		id := string(ns.ID)
		canonical, err := core.ParseCanonicalID(id)
		if err != nil || string(canonical) != id {
			return SelectedNamespaceSetAudit{}, fmt.Errorf("invalid selected semantic namespace %q", id)
		}
		if id == "std" {
			return SelectedNamespaceSetAudit{}, fmt.Errorf("std prelude must not be an explicitly selected dependency")
		}
		if _, ok := byID[id]; ok {
			return SelectedNamespaceSetAudit{}, fmt.Errorf("conflicting selected semantic namespace %q: multiple definitions", id)
		}
		byID[id] = ns
		names = append(names, id)
	}
	sort.Strings(names)
	out := SelectedNamespaceSetAudit{Selected: names}
	adjacent := make(map[string][]string, len(names))
	for _, name := range names {
		// The existing single-namespace audit is the only source of the
		// observed reference list. This adds grouping, not a second resolver.
		audit, err := AuditNamespaceDependencies(byID[name], selected)
		if err != nil {
			return SelectedNamespaceSetAudit{}, fmt.Errorf("inspect %s: %w", name, err)
		}
		byTarget := make(map[string][]NamespaceDependencyUse)
		for _, reference := range audit.External {
			byTarget[reference.Namespace] = append(byTarget[reference.Namespace], reference)
		}
		targets := make([]string, 0, len(byTarget))
		for target := range byTarget {
			targets = append(targets, target)
		}
		sort.Strings(targets)
		adjacent[name] = targets
		for _, target := range targets {
			out.Dependencies = append(out.Dependencies, SelectedNamespaceDependency{
				From: name, To: target, References: byTarget[target],
			})
		}
		for _, reference := range audit.Unmatched {
			out.Unmatched = append(out.Unmatched, UnmatchedNamespaceReference{
				From: name, Reference: reference,
			})
		}
	}
	out.CycleCandidates = observedNamespaceCycles(names, adjacent)
	return out, nil
}

// observedNamespaceCycles reports only strongly connected components with
// two or more selected namespaces. It is deterministic for equal graph data.
func observedNamespaceCycles(names []string, adjacent map[string][]string) []NamespaceCycleCandidate {
	ordinal := 0
	indices := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	var cycles []NamespaceCycleCandidate
	var visit func(string)
	visit = func(id string) {
		indices[id] = ordinal
		low[id] = ordinal
		ordinal++
		stack = append(stack, id)
		onStack[id] = true
		for _, target := range adjacent[id] {
			if _, visited := indices[target]; !visited {
				visit(target)
				if low[target] < low[id] {
					low[id] = low[target]
				}
			} else if onStack[target] && indices[target] < low[id] {
				low[id] = indices[target]
			}
		}
		if low[id] != indices[id] {
			return
		}
		var component []string
		for {
			last := len(stack) - 1
			item := stack[last]
			stack = stack[:last]
			onStack[item] = false
			component = append(component, item)
			if item == id {
				break
			}
		}
		if len(component) > 1 {
			sort.Strings(component)
			cycles = append(cycles, NamespaceCycleCandidate{Namespaces: component})
		}
	}
	for _, name := range names {
		if _, seen := indices[name]; !seen {
			visit(name)
		}
	}
	sort.Slice(cycles, func(i, j int) bool {
		return strings.Join(cycles[i].Namespaces, "\x00") < strings.Join(cycles[j].Namespaces, "\x00")
	})
	return cycles
}

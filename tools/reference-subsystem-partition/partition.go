// Package partition provides a candidate-only, read-only exact evaluator for
// fixed-ownership direct writes with allowed memberships and must-link constraints.
// It neither resolves accepted Source nor grants authorization or infers runtime effects.
package partition

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// AuthorityLimit is part of every report, including infeasible ones.
const AuthorityLimit = "caller-supplied candidate graph only; not accepted ownership, complete Source, runtime effect, authorization, or migration safety"

type Function struct {
	ID      string
	Allowed []string
}

type Write struct {
	ID        string // unique direct-write relation occurrence, not an evidence record count
	Function  string
	Data      string // caller-qualified identity, not a display name
	Owner     string // explicit fixed candidate owner for this Data
	Operation string // exactly C, U, or D; reads and indirect effects are excluded
	Source    string // candidate evidence locator; not an acceptance receipt
	Weight    int64  // positive integer; no inferred workload frequency
}

type MustLink struct {
	Left  string
	Right string
}

type Problem struct {
	Functions []Function
	Writes    []Write
	MustLinks []MustLink
}

type Placement struct {
	Function  string
	Subsystem string
}

// Component is a traceable optimality witness for one must-link component.
type Component struct {
	Functions      []string
	Allowed        []string
	Selected       string
	TotalWeight    int64
	InternalWeight int64
	CrossWeight    int64
	CrossWriteIDs  []string
}

type Report struct {
	Status                  string // exact-for-supplied-graph or infeasible
	AuthorityLimit          string
	Reason                  string
	MinimumKnownCrossWeight *int64 // nil for infeasible/unknown problems
	Components              []Component
	Placements              []Placement
}

// Evaluate never mutates the input. Only positive, explicitly declared C/U/D
// occurrences count as writes. No unknown edge is invented or treated as known.
// This is an exact optimum only when all hard constraints are represented by
// individual allowed memberships and must-link relations, with fixed Data owners.
func Evaluate(p Problem) (Report, error) {
	base := Report{AuthorityLimit: AuthorityLimit}
	if len(p.Functions) == 0 {
		return base, fmt.Errorf("no functions: an empty candidate graph is not proof of closure")
	}
	functions := make(map[string]Function, len(p.Functions))
	parents := make(map[string]string, len(p.Functions))
	for _, f := range p.Functions {
		if !validID(f.ID) {
			return base, fmt.Errorf("invalid function identity")
		}
		if _, exists := functions[f.ID]; exists {
			return base, fmt.Errorf("duplicate function %q", f.ID)
		}
		if len(f.Allowed) == 0 {
			return base, fmt.Errorf("function %q has no declared allowed membership", f.ID)
		}
		unique := map[string]bool{}
		for _, s := range f.Allowed {
			if !validID(s) || unique[s] {
				return base, fmt.Errorf("function %q has invalid or duplicate allowed membership", f.ID)
			}
			unique[s] = true
		}
		functions[f.ID] = f
		parents[f.ID] = f.ID
	}
	var find func(string) string
	find = func(id string) string {
		if parents[id] != id {
			parents[id] = find(parents[id])
		}
		return parents[id]
	}
	for _, link := range p.MustLinks {
		if _, ok := functions[link.Left]; !ok {
			return base, fmt.Errorf("must-link refers to unknown left function %q", link.Left)
		}
		if _, ok := functions[link.Right]; !ok {
			return base, fmt.Errorf("must-link refers to unknown right function %q", link.Right)
		}
		parents[find(link.Right)] = find(link.Left)
	}

	ownerByData := map[string]string{}
	seenWrite := map[string]bool{}
	writesByFn := map[string][]Write{}
	for _, w := range p.Writes {
		if !validID(w.ID) || !validID(w.Data) || !validID(w.Owner) || !validID(w.Source) {
			return base, fmt.Errorf("write has an empty or invalid identity, owner, or source")
		}
		if _, exists := functions[w.Function]; !exists {
			return base, fmt.Errorf("write %q refers to unknown function %q", w.ID, w.Function)
		}
		if seenWrite[w.ID] {
			return base, fmt.Errorf("duplicate write relation occurrence %q", w.ID)
		}
		seenWrite[w.ID] = true
		if w.Operation != "C" && w.Operation != "U" && w.Operation != "D" {
			return base, fmt.Errorf("write %q must be exactly C, U, or D", w.ID)
		}
		if w.Weight <= 0 {
			return base, fmt.Errorf("write %q must have positive integral weight", w.ID)
		}
		if prior, ok := ownerByData[w.Data]; ok && prior != w.Owner {
			return base, fmt.Errorf("conflicting candidate owners for Data %q", w.Data)
		}
		ownerByData[w.Data] = w.Owner
		writesByFn[w.Function] = append(writesByFn[w.Function], w)
	}

	groupByRoot := map[string][]string{}
	for id := range functions {
		groupByRoot[find(id)] = append(groupByRoot[find(id)], id)
	}
	groups := make([][]string, 0, len(groupByRoot))
	for _, g := range groupByRoot {
		sort.Strings(g)
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i][0] < groups[j][0] })
	components := make([]Component, 0, len(groups))
	var optimum int64
	for _, ids := range groups {
		allowed := make(map[string]bool)
		for _, s := range functions[ids[0]].Allowed {
			allowed[s] = true
		}
		for _, id := range ids[1:] {
			next := make(map[string]bool)
			for _, s := range functions[id].Allowed {
				if allowed[s] {
					next[s] = true
				}
			}
			allowed = next
		}
		options := make([]string, 0, len(allowed))
		for s := range allowed {
			options = append(options, s)
		}
		sort.Strings(options)
		if len(options) == 0 {
			return Report{
				Status: "infeasible", AuthorityLimit: AuthorityLimit,
				Reason: fmt.Sprintf("must-link component %v has no common allowed subsystem", ids),
			}, nil
		}

		byOwner := map[string]int64{}
		var total int64
		for _, id := range ids {
			for _, w := range writesByFn[id] {
				if total > math.MaxInt64-w.Weight || byOwner[w.Owner] > math.MaxInt64-w.Weight {
					return base, fmt.Errorf("write weights overflow int64")
				}
				total += w.Weight
				byOwner[w.Owner] += w.Weight
			}
		}
		selected := options[0] // lexicographic deterministic tie-break
		var best int64
		for _, s := range options {
			if byOwner[s] > best {
				best, selected = byOwner[s], s
			}
		}
		cross := total - best
		if optimum > math.MaxInt64-cross {
			return base, fmt.Errorf("total optimum overflows int64")
		}
		optimum += cross
		c := Component{
			Functions: ids, Allowed: options, Selected: selected,
			TotalWeight: total, InternalWeight: best, CrossWeight: cross,
			CrossWriteIDs: []string{},
		}
		for _, id := range ids {
			for _, w := range writesByFn[id] {
				if w.Owner != selected {
					c.CrossWriteIDs = append(c.CrossWriteIDs, w.ID)
				}
			}
		}
		sort.Strings(c.CrossWriteIDs)
		components = append(components, c)
	}
	placements := make([]Placement, 0, len(p.Functions))
	for _, c := range components {
		for _, id := range c.Functions {
			placements = append(placements, Placement{Function: id, Subsystem: c.Selected})
		}
	}
	sort.Slice(placements, func(i, j int) bool { return placements[i].Function < placements[j].Function })
	return Report{
		Status: "exact-for-supplied-graph", AuthorityLimit: AuthorityLimit,
		MinimumKnownCrossWeight: &optimum,
		Components:              components, Placements: placements,
	}, nil
}

func validID(s string) bool {
	return s != "" && strings.TrimSpace(s) == s
}

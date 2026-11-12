package partition

import (
	"fmt"
	"sort"
	"strings"
)

// ConstraintKind identifies one hard constraint type.
type ConstraintKind string

const (
	ConstraintMustLink          ConstraintKind = "must-link"
	ConstraintMustSeparate      ConstraintKind = "must-separate"
	ConstraintCapacity          ConstraintKind = "capacity"
	ConstraintFixed             ConstraintKind = "fixed"
	ConstraintExternalOwned     ConstraintKind = "external-owned"
	ConstraintSharedUpdate      ConstraintKind = "shared-update"
	ConstraintBusinessInvariant ConstraintKind = "business-invariant"
)

// Constraint is one caller-supplied hard constraint.
type Constraint struct {
	Kind          ConstraintKind `json:"kind"`
	ID            string         `json:"id"`
	Left          string         `json:"left,omitempty"` // function/data/subsystem id depending on kind
	Right         string         `json:"right,omitempty"`
	Subsystem     string         `json:"subsystem,omitempty"`
	MaxSize       int            `json:"maxSize,omitempty"`
	Members       []string       `json:"members,omitempty"`
	SourceLocator string         `json:"sourceLocator,omitempty"`
	Notes         string         `json:"notes"`
}

// ConstraintProblem bundles the graph and the constraints.
type ConstraintProblem struct {
	Graph       AccessGraph
	Constraints []Constraint
}

// MustLinkComponents computes transitive closure and returns grouped members.
func MustLinkComponents(functions []string, links []MustLink) ([][]string, error) {
	fnSet := map[string]bool{}
	for _, f := range functions {
		fnSet[f] = true
	}
	parents := map[string]string{}
	for _, f := range functions {
		parents[f] = f
	}
	var find func(string) string
	find = func(id string) string {
		if parents[id] != id {
			parents[id] = find(parents[id])
		}
		return parents[id]
	}
	for _, l := range links {
		if !fnSet[l.Left] || !fnSet[l.Right] {
			return nil, fmt.Errorf("must-link references unknown function")
		}
		parents[find(l.Right)] = find(l.Left)
	}
	groupByRoot := map[string][]string{}
	for _, id := range functions {
		groupByRoot[find(id)] = append(groupByRoot[find(id)], id)
	}
	out := make([][]string, 0, len(groupByRoot))
	for _, g := range groupByRoot {
		sort.Strings(g)
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })
	return out, nil
}

// MustSeparateCheck returns true when any pair is within the same must-link component.
func MustSeparateCheck(components [][]string, left, right string) bool {
	for _, c := range components {
		foundL, foundR := false, false
		for _, m := range c {
			if m == left {
				foundL = true
			}
			if m == right {
				foundR = true
			}
		}
		if foundL && foundR {
			return true
		}
	}
	return false
}

// ValidateConstraints deterministically inspects constraints against the graph and returns issues.
func ValidateConstraints(p ConstraintProblem) ([]string, error) {
	var issues []string
	fnSet := map[string]bool{}
	dataSet := map[string]bool{}
	for _, f := range p.Graph.Functions {
		fnSet[f] = true
	}
	for _, d := range p.Graph.DataIDs {
		dataSet[d] = true
	}

	for _, c := range p.Constraints {
		if strings.TrimSpace(c.ID) == "" {
			issues = append(issues, "constraint with empty id")
		}
		switch c.Kind {
		case ConstraintMustLink:
			if c.Left == "" || c.Right == "" {
				issues = append(issues, fmt.Sprintf("must-link %s missing endpoints", c.ID))
				continue
			}
			if !fnSet[c.Left] || !fnSet[c.Right] {
				issues = append(issues, fmt.Sprintf("must-link %s references unknown function", c.ID))
			}
		case ConstraintMustSeparate:
			if c.Left == "" || c.Right == "" {
				issues = append(issues, fmt.Sprintf("must-separate %s missing endpoints", c.ID))
				continue
			}
			if !fnSet[c.Left] || !fnSet[c.Right] {
				issues = append(issues, fmt.Sprintf("must-separate %s references unknown function", c.ID))
			}
		case ConstraintCapacity:
			if c.Subsystem == "" || c.MaxSize <= 0 {
				issues = append(issues, fmt.Sprintf("capacity %s missing subsystem or max size", c.ID))
			}
		case ConstraintFixed:
			if c.Left == "" || c.Subsystem == "" {
				issues = append(issues, fmt.Sprintf("fixed %s missing function or subsystem", c.ID))
				continue
			}
			if !fnSet[c.Left] {
				issues = append(issues, fmt.Sprintf("fixed %s references unknown function", c.ID))
			}
		case ConstraintExternalOwned:
			if c.Left == "" {
				issues = append(issues, fmt.Sprintf("external-owned %s missing data", c.ID))
				continue
			}
			if !dataSet[c.Left] {
				issues = append(issues, fmt.Sprintf("external-owned %s references unknown data", c.ID))
			}
		case ConstraintSharedUpdate:
			if c.Left == "" || len(c.Members) == 0 {
				issues = append(issues, fmt.Sprintf("shared-update %s missing data or members", c.ID))
				continue
			}
			if !dataSet[c.Left] {
				issues = append(issues, fmt.Sprintf("shared-update %s references unknown data", c.ID))
			}
		case ConstraintBusinessInvariant:
			if len(c.Members) < 2 {
				issues = append(issues, fmt.Sprintf("business-invariant %s needs 2+ members", c.ID))
			}
		default:
			issues = append(issues, fmt.Sprintf("unknown constraint kind %s (%s)", c.Kind, c.ID))
		}
	}
	sort.Strings(issues)
	return issues, nil
}

// ExternalOwnedDataIDs returns the set of data ids marked external-owned.
func ExternalOwnedDataIDs(constraints []Constraint) map[string]bool {
	out := map[string]bool{}
	for _, c := range constraints {
		if c.Kind == ConstraintExternalOwned {
			out[c.Left] = true
		}
	}
	return out
}

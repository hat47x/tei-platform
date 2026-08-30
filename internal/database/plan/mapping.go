package plan

import (
	"strings"

	"tei/internal/query"
)

type MappingConfidence string

const (
	MappingExact    MappingConfidence = "exact"
	MappingInferred MappingConfidence = "inferred"
)

type LogicalMapping struct {
	RelNodeID  string            `json:"relNodeId"`
	PlanNodeID string            `json:"planNodeId"`
	Confidence MappingConfidence `json:"confidence"`
	Reason     string            `json:"reason,omitempty"`
}

// MapLogical conservatively overlays a provider execution plan onto TEI's
// logical relational IR. Absence of a mapping is meaningful: optimizers do not
// preserve a one-to-one relationship between authored logical operators and
// physical plan nodes.
func MapLogical(q query.DerivedIR, p IR) []LogicalMapping {
	var out []LogicalMapping
	usedPlan := map[string]bool{}
	usedRel := map[string]bool{}

	// Physical sources can usually be mapped exactly because both layers retain
	// relation identity and alias/source occurrence.
	for _, s := range q.Sources {
		if s.SourceKind != "table" {
			continue
		}
		relID := s.ID
		candidates := []Node{}
		for _, n := range p.Nodes {
			if n.Kind != NodeScan && n.Kind != NodeIndexScan {
				continue
			}
			if physicalEqual(s, n) {
				candidates = append(candidates, n)
			}
		}
		if len(candidates) == 1 {
			n := candidates[0]
			out = append(out, LogicalMapping{RelNodeID: relID, PlanNodeID: n.ID, Confidence: MappingExact, Reason: "same physical source occurrence"})
			usedPlan[n.ID], usedRel[relID] = true, true
		}
	}

	// A PostgreSQL filter may be fused into a scan rather than represented by
	// its own plan node. Preserve that as an inferred overlay, not an exact
	// equivalence.
	var relFilters []query.RelNode
	var predicateNodes []Node
	for _, n := range q.Relational.Nodes {
		if n.Kind == query.RelFilter && !usedRel[n.ID] {
			relFilters = append(relFilters, n)
		}
	}
	for _, n := range p.Nodes {
		if usedPlan[n.ID] {
			// The same physical scan may legitimately realize both Source and
			// Filter, so source mapping does not exclude this inference.
		}
		if _, ok := n.ProviderDetails["Filter"]; ok {
			predicateNodes = append(predicateNodes, n)
		}
		if _, ok := n.ProviderDetails["Index Cond"]; ok {
			predicateNodes = appendUniqueNode(predicateNodes, n)
		}
	}
	if len(relFilters) == 1 && len(predicateNodes) == 1 {
		out = append(out, LogicalMapping{RelNodeID: relFilters[0].ID, PlanNodeID: predicateNodes[0].ID, Confidence: MappingInferred, Reason: "logical predicate executed inside physical scan"})
		usedRel[relFilters[0].ID] = true
	}

	mapUniqueKind := func(relKind query.RelNodeKind, physicalKind NodeKind, reason string) {
		var rs []query.RelNode
		var ps []Node
		for _, r := range q.Relational.Nodes {
			if r.Kind == relKind && !usedRel[r.ID] {
				rs = append(rs, r)
			}
		}
		for _, n := range p.Nodes {
			if n.Kind == physicalKind && !usedPlan[n.ID] {
				ps = append(ps, n)
			}
		}
		if len(rs) == 1 && len(ps) == 1 {
			out = append(out, LogicalMapping{RelNodeID: rs[0].ID, PlanNodeID: ps[0].ID, Confidence: MappingInferred, Reason: reason})
			usedRel[rs[0].ID], usedPlan[ps[0].ID] = true, true
		}
	}
	mapUniqueKind(query.RelJoin, NodeJoin, "unique logical join and physical join")
	mapUniqueKind(query.RelAggregate, NodeAggregate, "unique logical aggregate and physical aggregate")
	mapUniqueKind(query.RelSort, NodeSort, "unique logical sort and physical sort")
	mapUniqueKind(query.RelLimit, NodeLimit, "unique logical limit and physical limit")
	return out
}

func physicalEqual(s query.SourceRef, n Node) bool {
	if normalizePhysical(s.Name) != normalizePhysical(n.PhysicalObject) {
		// A resolved source may still have been authored without schema.
		if lastPart(normalizePhysical(s.Name)) != lastPart(normalizePhysical(n.PhysicalObject)) || s.Resolution != "schema-resolved" {
			return false
		}
	}
	if s.Alias != "" && n.Alias != "" && !strings.EqualFold(s.Alias, n.Alias) {
		return false
	}
	return true
}

func normalizePhysical(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
func lastPart(s string) string {
	if i := strings.LastIndex(s, "."); i >= 0 {
		return s[i+1:]
	}
	return s
}

func appendUniqueNode(in []Node, n Node) []Node {
	for _, existing := range in {
		if existing.ID == n.ID {
			return in
		}
	}
	return append(in, n)
}

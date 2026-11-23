package iomapping

import (
	"fmt"

	"tei/internal/composition"
)

// PlanGraph is a disposable, read-only view of declared I/O mapping.
// It is not an accepted semantic graph, a runtime connection, or a writer
// target. In particular, Report.Order must not be used to reconstruct edges:
// data mappings and ordering constraints contribute to the same sort.
type PlanGraph struct {
	Scope            string
	RegistryRevision uint64
	Verdict          composition.ProposalVerdict
	Diagnostics      []Diagnostic
	Nodes            []PlanGraphNode
	Ports            []PlanGraphPort
	DataEdges        []PlanGraphDataEdge
	OrderConstraints []PlanGraphOrder
}

// All ViewID values are ephemeral projection keys, not Subject IDs or
// stable Source locators. Reordering the input may change them.
type PlanGraphNode struct {
	ViewID     string
	SlotID     string
	Scope      string
	Contract   composition.Contract
	ProviderID string
}

type PlanGraphPort struct {
	ViewID     string
	NodeViewID string
	Direction  string // input or output; Context fields are not ports
	Ref        Ref
	Field      Field
}

// Declared edges remain visible even when Inspect returns unknown/conflict.
// EndpointsResolved checks only declared port references, NOT type compatibility,
// provider readiness, execution authority or eligibility to transfer data.
type PlanGraphDataEdge struct {
	ViewID            string
	Mapping           Mapping
	FromPortViewID     string
	ToPortViewID       string
	EndpointsResolved bool
}

type PlanGraphOrder struct {
	ViewID            string
	Ordering          Ordering
	FromNodeViewID    string
	ToNodeViewID      string
	EndpointsResolved bool
}

type planGraphPortKey struct {
	Ref       Ref
	Direction string
}

// ProjectPlanGraph inspects the exact supplied plan through the existing
// Composition Registry, then projects its declared topology without mutation.
// This does not pin the plan to a Source revision; a consumer must establish
// the Source/Workspace snapshot separately before using it for navigation.
func ProjectPlanGraph(registry *composition.Registry, plan Plan) PlanGraph {
	report := Inspect(registry, plan)
	out := PlanGraph{
		Scope:            plan.Scope,
		RegistryRevision: report.Wiring.Revision,
		Verdict:          report.Verdict,
		Diagnostics:      append([]Diagnostic(nil), report.Diagnostics...),
	}
	nodes := make(map[string][]string)
	ports := make(map[planGraphPortKey][]string)

	for i, slot := range plan.Slots {
		nodeID := fmt.Sprintf("node:%d", i)
		nodes[slot.ID] = append(nodes[slot.ID], nodeID)
		out.Nodes = append(out.Nodes, PlanGraphNode{
			ViewID: nodeID, SlotID: slot.ID, Scope: plan.Scope,
			Contract: slot.Contract, ProviderID: slot.ProviderID,
		})
		for _, group := range []struct {
			name   string
			fields []Field
		}{
			{name: "input", fields: slot.Inputs},
			{name: "output", fields: slot.Outputs},
		} {
			for j, field := range group.fields {
				portID := fmt.Sprintf("port:%d:%s:%d", i, group.name, j)
				ref := Ref{Slot: slot.ID, Contract: slot.Contract, Field: field.ID}
				out.Ports = append(out.Ports, PlanGraphPort{
					ViewID: portID, NodeViewID: nodeID,
					Direction: group.name, Ref: ref, Field: field,
				})
				key := planGraphPortKey{Ref: ref, Direction: group.name}
				ports[key] = append(ports[key], portID)
			}
		}
	}

	// A unique match is a structural endpoint only. Duplicate Slot/Field
	// declarations are never repaired by choosing the first view occurrence.
	unique := func(ids []string) string {
		if len(ids) == 1 {
			return ids[0]
		}
		return ""
	}
	for i, m := range plan.Mappings {
		from := unique(ports[planGraphPortKey{Ref: m.From, Direction: "output"}])
		to := unique(ports[planGraphPortKey{Ref: m.To, Direction: "input"}])
		out.DataEdges = append(out.DataEdges, PlanGraphDataEdge{
			ViewID: fmt.Sprintf("data:%d", i), Mapping: m,
			FromPortViewID: from, ToPortViewID: to,
			EndpointsResolved: from != "" && to != "",
		})
	}
	for i, ordering := range plan.After {
		from, to := unique(nodes[ordering.Before]), unique(nodes[ordering.After])
		out.OrderConstraints = append(out.OrderConstraints, PlanGraphOrder{
			ViewID: fmt.Sprintf("order:%d", i), Ordering: ordering,
			FromNodeViewID: from, ToNodeViewID: to,
			EndpointsResolved: from != "" && to != "",
		})
	}
	return out
}

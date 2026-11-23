package iomapping

import (
	"fmt"
	"reflect"
	"testing"

	"tei/internal/composition"
)

func hasGraphDiagnostic(graph PlanGraph, code string) bool {
	for _, d := range graph.Diagnostics {
		if d.Code == code {
			return true
		}
	}
	return false
}

func TestPlanGraphProjectsAllPortsAndOneDataMapping(t *testing.T) {
	r, p := fixture(t)
	beforeRegistry := r.Snapshot()
	beforePlan := fmt.Sprintf("%#v", p)
	got := ProjectPlanGraph(r, p)

	// Independent expected from the fixture: producer has two outputs,
	// consumer has one input, and only producer.value is wired.
	if len(got.Nodes) != 2 || len(got.Ports) != 3 ||
		len(got.DataEdges) != 1 || len(got.OrderConstraints) != 0 {
		t.Fatalf("UG-01: wrong topology (unused port must remain visible): %+v", got)
	}
	if got.Scope != "app" || got.RegistryRevision != beforeRegistry.Revision ||
		got.Verdict != composition.ProposalCandidate || len(got.Diagnostics) != 0 {
		t.Fatalf("UG-01: missing scope/revision/structural verdict: %+v", got)
	}
	edge := got.DataEdges[0]
	if !edge.EndpointsResolved || edge.FromPortViewID == "" || edge.ToPortViewID == "" ||
		edge.Mapping.From != p.Mappings[0].From || edge.Mapping.To != p.Mappings[0].To {
		t.Fatalf("UG-01: lost exact field refs: %+v", edge)
	}
	unused := ""
	for _, port := range got.Ports {
		if port.Ref.Field == "unused" {
			unused = port.ViewID
			if port.Direction != "output" || port.Ref.Contract != sourceContract {
				t.Fatalf("UG-01: unused output lost contract/direction: %+v", port)
			}
		}
	}
	if unused == "" || unused == edge.FromPortViewID || unused == edge.ToPortViewID {
		t.Fatalf("UG-01: unused output silently dropped or wired: %+v", got.Ports)
	}
	if !reflect.DeepEqual(beforeRegistry, r.Snapshot()) || beforePlan != fmt.Sprintf("%#v", p) {
		t.Fatal("projection changed registry or plan")
	}
}

func TestPlanGraphKeepsOrderingSeparateFromData(t *testing.T) {
	r, p := fixture(t)
	p.After = []Ordering{{Before: "producer", After: "consumer", Reason: "verification"}}
	got := ProjectPlanGraph(r, p)
	if got.Verdict != composition.ProposalCandidate ||
		len(got.DataEdges) != 1 || len(got.OrderConstraints) != 1 {
		t.Fatalf("UG-02/10: ordering constraint became a data edge: %+v", got)
	}
	order := got.OrderConstraints[0]
	if !order.EndpointsResolved || order.Ordering.Reason != "verification" ||
		order.FromNodeViewID == "" || order.ToNodeViewID == "" {
		t.Fatalf("UG-02: declared order lost: %+v", order)
	}
}

func TestPlanGraphProviderDependencyDoesNotBecomeDataOrControl(t *testing.T) {
	r, p := fixture(t)
	old := r.Snapshot()
	_, err := r.Apply(old.Revision, []composition.Provider{{
		ID: "target-p", Scope: "app", Offers: targetContract,
		Requires: []composition.Dependency{{Contract: sourceContract}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := ProjectPlanGraph(r, p)
	if got.Verdict != composition.ProposalCandidate ||
		len(got.DataEdges) != 1 || len(got.OrderConstraints) != 0 ||
		got.RegistryRevision != old.Revision+1 {
		t.Fatalf("UG-03: provider dependency was promoted to a data/control edge: %+v", got)
	}
}

func TestPlanGraphTwoConsumersPreserveDistinctUseAndPorts(t *testing.T) {
	r, p := fixture(t)
	p.Slots = append(p.Slots, Slot{
		ID: "consumer-2", Contract: targetContract,
		Multiplicity: One, Inputs: []Field{field("value")},
	})
	p.Mappings = append(p.Mappings, Mapping{
		From: p.Mappings[0].From,
		To:   Ref{Slot: "consumer-2", Contract: targetContract, Field: "value"},
	})
	got := ProjectPlanGraph(r, p)
	if got.Verdict != composition.ProposalCandidate ||
		len(got.Nodes) != 3 || len(got.Ports) != 4 || len(got.DataEdges) != 2 ||
		got.DataEdges[0].ToPortViewID == got.DataEdges[1].ToPortViewID ||
		got.DataEdges[0].FromPortViewID != got.DataEdges[1].FromPortViewID {
		t.Fatalf("UG-04/07: use occurrences/typed ports conflated: %+v", got)
	}
	// The same selected Provider may serve both slots. This does not attest
	// two runtime processes or an activated provider instance.
}

func TestPlanGraphInvalidContextMappingRemainsUnresolved(t *testing.T) {
	r, p := fixture(t)
	p.Slots[1].Context = []Field{field("principal")}
	p.Mappings[0].To.Field = "principal"
	got := ProjectPlanGraph(r, p)
	if got.Verdict != composition.ProposalConflict || len(got.DataEdges) != 1 ||
		got.DataEdges[0].EndpointsResolved || got.DataEdges[0].ToPortViewID != "" ||
		!hasGraphDiagnostic(got, "context-override") {
		t.Fatalf("UG-05: Context field falsely exposed as input port: %+v", got)
	}
}

func TestPlanGraphTypeConflictStillShowsOnlyDeclaredConnection(t *testing.T) {
	r, p := fixture(t)
	p.Slots[1].Inputs[0].Type.Unit = "JPY"
	got := ProjectPlanGraph(r, p)
	if got.Verdict != composition.ProposalConflict || len(got.DataEdges) != 1 ||
		!got.DataEdges[0].EndpointsResolved || !hasGraphDiagnostic(got, "type-incompatible") {
		t.Fatalf("UG-06: physical-looking edge concealed incompatible contract: %+v", got)
	}
	// EndpointsResolved is not an authorization to transfer data.
}

func TestPlanGraphMissingRefDoesNotGuessPort(t *testing.T) {
	r, p := fixture(t)
	p.Mappings[0].From.Contract = targetContract // name and Field still look identical
	got := ProjectPlanGraph(r, p)
	if len(got.DataEdges) != 1 || got.DataEdges[0].EndpointsResolved ||
		got.DataEdges[0].FromPortViewID != "" || !hasGraphDiagnostic(got, "unknown-field-reference") {
		t.Fatalf("UG-04/10: guessed endpoint by visible name: %+v", got)
	}
}

func TestPlanGraphDuplicateSlotNotResolvedByFirstMatch(t *testing.T) {
	r, p := fixture(t)
	p.Slots = append(p.Slots, p.Slots[1])
	got := ProjectPlanGraph(r, p)
	if got.Verdict != composition.ProposalConflict ||
		got.DataEdges[0].EndpointsResolved || !hasGraphDiagnostic(got, "invalid-or-duplicate-slot") {
		t.Fatalf("duplicate local slot was silently chosen: %+v", got)
	}
}

func TestPlanGraphOrderOnlyNeverInventsDataMapping(t *testing.T) {
	r, p := fixture(t)
	p.Mappings = nil
	p.After = []Ordering{{Before: "producer", After: "consumer", Reason: "dependency-only"}}

	got := ProjectPlanGraph(r, p)
	if len(got.Nodes) != 2 || len(got.Ports) != 3 ||
		len(got.DataEdges) != 0 || len(got.OrderConstraints) != 1 ||
		!got.OrderConstraints[0].EndpointsResolved {
		t.Fatalf("UG-02/10: order-only plan inferred a transfer: %+v", got)
	}
	// The missing required consumer input makes the plan invalid, but does
	// not erase the declarative ordering constraint or invent a data edge.
	if got.Verdict != composition.ProposalConflict ||
		!hasGraphDiagnostic(got, "required-input-unmapped") {
		t.Fatalf("UG-02: missing input was accepted: %+v", got)
	}
}

func TestPlanGraphDuplicateOutputFieldsRemainAmbiguous(t *testing.T) {
	r, p := fixture(t)
	p.Slots[0].Outputs = append(p.Slots[0].Outputs, field("value"))
	got := ProjectPlanGraph(r, p)
	if len(got.Nodes) != 2 || len(got.Ports) != 4 ||
		len(got.DataEdges) != 1 || got.DataEdges[0].EndpointsResolved ||
		got.DataEdges[0].FromPortViewID != "" ||
		got.Verdict != composition.ProposalConflict ||
		!hasGraphDiagnostic(got, "duplicate-or-empty-field") {
		t.Fatalf("UG-04: same-named output was silently chosen: %+v", got)
	}
}

func TestPlanGraphRegistryABARetainsRevisionChange(t *testing.T) {
	r, p := fixture(t)
	original := ProjectPlanGraph(r, p)
	first := r.Snapshot()
	_, err := r.Apply(first.Revision, []composition.Provider{{
		ID: "unrelated-p", Scope: "app",
		Offers: composition.Contract{ID: "unrelated", Version: "1", Shape: "other"},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	middle := ProjectPlanGraph(r, p)
	_, err = r.Apply(r.Snapshot().Revision, nil, []string{"unrelated-p"})
	if err != nil {
		t.Fatal(err)
	}
	restored := ProjectPlanGraph(r, p)
	if original.RegistryRevision != first.Revision ||
		middle.RegistryRevision <= original.RegistryRevision ||
		restored.RegistryRevision <= middle.RegistryRevision ||
		restored.Verdict != composition.ProposalCandidate ||
		len(restored.DataEdges) != 1 {
		t.Fatalf("UG-09: A-B-A reused old inspection revision: %+v / %+v / %+v",
			original, middle, restored)
	}
	// The caller must additionally pin Source/Workspace and reject the
	// original report against the latest revision. ViewIDs are not grants.
}

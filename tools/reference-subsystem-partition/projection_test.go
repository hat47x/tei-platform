package partition

import (
	"encoding/json"
	"os"
	"testing"
)

func fixtureOccurrences(t *testing.T) ([]AccessOccurrence, []ReadEdge, []InvocationEdge, []string) {
	t.Helper()
	fixtureBytes, err := os.ReadFile("testdata/fixture.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Functions   []Function         `json:"functions"`
		DataIDs     []string           `json:"dataIds"`
		WriteEdges  []AccessOccurrence `json:"writeEdges"`
		ReadEdges   []AccessOccurrence `json:"readEdges"`
		Invocations []InvocationEdge   `json:"invocations"`
	}
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		t.Fatal(err)
	}
	reads := make([]ReadEdge, 0, len(fixture.ReadEdges))
	for _, e := range fixture.ReadEdges {
		reads = append(reads, ReadEdge{ID: e.ID, Function: e.Function, Data: e.Data, Provenance: string(e.Provenance), SourceLocator: e.SourceLocator, Notes: e.Notes})
	}
	fnIDs := make([]string, len(fixture.Functions))
	for i, f := range fixture.Functions {
		fnIDs[i] = f.ID
	}
	return fixture.WriteEdges, reads, fixture.Invocations, fnIDs
}

func fixtureDataIDs(t *testing.T) []string {
	t.Helper()
	fixtureBytes, err := os.ReadFile("testdata/fixture.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		DataIDs []string `json:"dataIds"`
	}
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture.DataIDs
}

func fixtureProjectionInput(t *testing.T, candidateID string) ProjectionInput {
	t.Helper()
	p, err := fixtureProblem()
	if err != nil {
		t.Fatal(err)
	}
	candidates, _ := GenerateCandidates(p)
	if len(candidates) == 0 {
		t.Fatal("no candidates from fixture")
	}
	var chosen Candidate
	if candidateID == "" {
		chosen = candidates[0]
	} else {
		for _, c := range candidates {
			if c.ID == candidateID {
				chosen = c
				break
			}
		}
		if chosen.ID == "" {
			t.Fatalf("candidate %s not found", candidateID)
		}
	}
	writeEdges, readEdges, invocations, fnIDs := fixtureOccurrences(t)
	external := ExternalOwnedDataIDs(p.Constraints)
	return ProjectionInput{
		CandidateID:   chosen.ID,
		Assignments:   chosen.Assignments,
		WriteEdges:    writeEdges,
		ReadEdges:     readEdges,
		Invocations:   invocations,
		Functions:     fnIDs,
		DataIDs:       fixtureDataIDs(t),
		ExternalOwned: external,
	}
}

func TestBuildProjectionReportExplicit(t *testing.T) {
	assign := map[string]string{
		"F1_order_submit": "sales",
		"F3_order_cancel": "billing",
	}
	writes := []Write{
		{ID: "w01", Function: "F1_order_submit", Data: "sales.order", Owner: "sales", Operation: "C", Weight: 1},
		{ID: "w04", Function: "F3_order_cancel", Data: "billing.invoice", Owner: "billing", Operation: "U", Weight: 1},
	}
	reads := []ReadEdge{
		{ID: "r01", Function: "F1_order_submit", Data: "inventory.stock", Provenance: "parsed-extracted"},
	}
	invocations := []InvocationEdge{
		{Caller: "F3_order_cancel", Callee: "F1_order_submit", Pattern: "notify", Provenance: "declared-source", SourceLocator: "x:1"},
		{Caller: "F1_order_submit", Callee: "UnknownFn", Pattern: "ext", Provenance: "declared-source", SourceLocator: "x:2"},
	}
	functions := []string{"F1_order_submit", "F3_order_cancel"}
	dataIDs := []string{"sales.order", "billing.invoice", "inventory.stock", "unlisted.data"}

	report := BuildProjectionReport(assign, writes, reads, invocations, functions, dataIDs)

	if report.SchemaVersion != "tei.subsystem-partition.projection/v0" {
		t.Fatalf("unexpected schema version %s", report.SchemaVersion)
	}
	if report.AuthorityLimit != AuthorityLimit {
		t.Fatalf("missing authority limit")
	}
	if len(report.Subsystems) != 2 {
		t.Fatalf("expected 2 subsystems, got %d", len(report.Subsystems))
	}
	if report.Subsystems[0].Subsystem != "billing" || report.Subsystems[1].Subsystem != "sales" {
		t.Fatalf("subsystems not sorted: %s, %s", report.Subsystems[0].Subsystem, report.Subsystems[1].Subsystem)
	}

	billing := report.Subsystems[0]
	if len(billing.Functions) != 1 || billing.Functions[0].Function != "F3_order_cancel" {
		t.Fatalf("unexpected billing functions: %+v", billing.Functions)
	}
	if len(billing.Functions[0].Writes) != 1 || billing.Functions[0].Writes[0] != "billing.invoice" {
		t.Fatalf("unexpected billing writes: %+v", billing.Functions[0].Writes)
	}
	if len(billing.DataIDs) != 1 || billing.DataIDs[0] != "billing.invoice" {
		t.Fatalf("unexpected billing data ids: %+v", billing.DataIDs)
	}
	if len(billing.Dependencies) != 1 {
		t.Fatalf("expected 1 billing dependency, got %d", len(billing.Dependencies))
	}
	dep := billing.Dependencies[0]
	if dep.Caller != "F3_order_cancel" || dep.Callee != "F1_order_submit" {
		t.Fatalf("unexpected dependency endpoints: %+v", dep)
	}
	if dep.CalleeSubsystem != "sales" {
		t.Fatalf("expected callee subsystem sales, got %q", dep.CalleeSubsystem)
	}
	if dep.Internal {
		t.Fatalf("cross-subsystem dependency must not be internal")
	}
	if dep.Pattern != "notify" || dep.SourceLocator != "x:1" {
		t.Fatalf("dependency metadata not projected: %+v", dep)
	}

	sales := report.Subsystems[1]
	if len(sales.Functions) != 1 || sales.Functions[0].Function != "F1_order_submit" {
		t.Fatalf("unexpected sales functions: %+v", sales.Functions)
	}
	if len(sales.Functions[0].Writes) != 1 || sales.Functions[0].Writes[0] != "sales.order" {
		t.Fatalf("unexpected sales writes: %+v", sales.Functions[0].Writes)
	}
	if len(sales.Functions[0].Reads) != 1 || sales.Functions[0].Reads[0] != "inventory.stock" {
		t.Fatalf("unexpected sales reads: %+v", sales.Functions[0].Reads)
	}
	if len(sales.DataIDs) != 2 || sales.DataIDs[0] != "inventory.stock" || sales.DataIDs[1] != "sales.order" {
		t.Fatalf("unexpected sales data ids: %+v", sales.DataIDs)
	}
	for _, id := range sales.DataIDs {
		if id == "unlisted.data" {
			t.Fatalf("unlisted data must not leak into the projection")
		}
	}
	if len(sales.Dependencies) != 1 {
		t.Fatalf("expected 1 sales dependency, got %d", len(sales.Dependencies))
	}
	if sales.Dependencies[0].CalleeSubsystem != "" || sales.Dependencies[0].Internal {
		t.Fatalf("unassigned callee must project empty subsystem and not be internal: %+v", sales.Dependencies[0])
	}
}

func TestBuildProjectionReportInternalDependency(t *testing.T) {
	assign := map[string]string{
		"F_a": "sales",
		"F_b": "sales",
	}
	invocations := []InvocationEdge{
		{Caller: "F_a", Callee: "F_b", Pattern: "internal-call", Provenance: "declared-source", SourceLocator: "y:1"},
	}
	functions := []string{"F_a", "F_b"}
	report := BuildProjectionReport(assign, nil, nil, invocations, functions, nil)
	if len(report.Subsystems) != 1 {
		t.Fatalf("expected 1 subsystem, got %d", len(report.Subsystems))
	}
	if len(report.Subsystems[0].Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(report.Subsystems[0].Dependencies))
	}
	if !report.Subsystems[0].Dependencies[0].Internal {
		t.Fatalf("same-subsystem invocation must be internal")
	}
	if report.Subsystems[0].Dependencies[0].CalleeSubsystem != "sales" {
		t.Fatalf("expected callee subsystem sales")
	}
}

func TestBuildProjectionReportFromFixture(t *testing.T) {
	p, err := fixtureProblem()
	if err != nil {
		t.Fatal(err)
	}
	candidates, _ := GenerateCandidates(p)
	if len(candidates) == 0 {
		t.Fatal("no candidates generated from fixture")
	}
	assign := map[string]string{}
	for _, a := range candidates[0].Assignments {
		assign[a.Function] = a.Subsystem
	}
	fnIDs := make([]string, len(p.Functions))
	for i, f := range p.Functions {
		fnIDs[i] = f.ID
	}
	dataIDs := []string{"sales.order", "sales.order_line", "billing.invoice", "inventory.stock", "crm.customer", "external.tax_ledger"}
	_, readEdges, invocations, _ := fixtureOccurrences(t)
	report := BuildProjectionReport(assign, p.Writes, readEdges, invocations, fnIDs, dataIDs)
	if len(report.Subsystems) == 0 {
		t.Fatal("projection produced no subsystems")
	}
	seen := map[string]int{}
	for _, sub := range report.Subsystems {
		for _, fn := range sub.Functions {
			seen[fn.Function]++
		}
	}
	if len(seen) != len(assign) {
		t.Fatalf("projection functions %d != assigned functions %d", len(seen), len(assign))
	}
	for fn, count := range seen {
		if count != 1 {
			t.Fatalf("function %s appears %d times in projection", fn, count)
		}
	}
	for _, sub := range report.Subsystems {
		for _, id := range sub.DataIDs {
			listed := false
			for _, d := range dataIDs {
				if d == id {
					listed = true
					break
				}
			}
			if !listed {
				t.Fatalf("unlisted data id %s leaked into %s projection", id, sub.Subsystem)
			}
		}
	}
}

func TestBuildERProjectionPreservesExternalNodes(t *testing.T) {
	in := fixtureProjectionInput(t, "")
	report := BuildERProjection(in)
	if report.SchemaVersion != "tei.subsystem-partition.er-projection/v0" {
		t.Fatalf("unexpected schema version %s", report.SchemaVersion)
	}
	if report.AuthorityLimit != AuthorityLimit {
		t.Fatalf("missing authority limit")
	}
	if report.Digest == "" {
		t.Fatalf("missing digest")
	}
	externalCount := 0
	for _, sub := range report.Subsystems {
		for _, node := range sub.Nodes {
			if node.Ownership == "external" {
				externalCount++
				if len(node.ReferencedBy) == 0 {
					t.Fatalf("external node %s missing referencing occurrence ids", node.DataID)
				}
			}
		}
	}
	if externalCount == 0 {
		t.Fatalf("fixture must contain at least one external node")
	}
}

func TestBuildERProjectionExternalOwnedAccessIsExternal(t *testing.T) {
	in := ProjectionInput{
		CandidateID: "cand-x",
		Assignments: []Assignment{{Function: "F1", Subsystem: "sales"}},
		WriteEdges: []AccessOccurrence{
			{ID: "w01", Function: "F1", Data: "external.tax_ledger", Operation: OpUpdate, Provenance: ProvDeclaredSource, Owner: "external", Multiplicity: 1},
		},
		Functions:     []string{"F1"},
		DataIDs:       []string{"external.tax_ledger"},
		ExternalOwned: map[string]bool{"external.tax_ledger": true},
	}
	report := BuildERProjection(in)
	found := false
	for _, sub := range report.Subsystems {
		for _, node := range sub.Nodes {
			if node.DataID == "external.tax_ledger" {
				found = true
				if node.Ownership != "external" {
					t.Fatalf("external-owned data must project as external, got %s", node.Ownership)
				}
				if len(node.ReferencedBy) != 1 || node.ReferencedBy[0] != "w01" {
					t.Fatalf("external node must keep referencing occurrence id: %+v", node.ReferencedBy)
				}
			}
		}
	}
	if !found {
		t.Fatalf("external-owned data node must not be deleted")
	}
}

func TestBuildERProjectionNonDisclosedNotRemoved(t *testing.T) {
	in := fixtureProjectionInput(t, "")
	// withhold one referenced data identity that is otherwise present as a node.
	in.PublicSlice = map[string]bool{"sales.order": true, "sales.order_line": true, "billing.invoice": true, "crm.customer": true, "external.tax_ledger": true}
	report := BuildERProjection(in)
	if len(report.NonDisclosed) == 0 {
		t.Fatalf("expected non-disclosed entries for subjects outside the public slice")
	}
	for _, entry := range report.NonDisclosed {
		if entry.ReasonCode == "" {
			t.Fatalf("non-disclosed entry missing reason code")
		}
		if entry.SubjectID != "inventory.stock" {
			t.Fatalf("unexpected non-disclosed subject %s", entry.SubjectID)
		}
	}
	found := false
	for _, sub := range report.Subsystems {
		for _, node := range sub.Nodes {
			if node.DataID == "inventory.stock" {
				found = true
				if node.Disclosure != "non-disclosed" {
					t.Fatalf("withheld node must be marked non-disclosed, got %s", node.Disclosure)
				}
			}
		}
	}
	if !found {
		t.Fatalf("withheld node must not be removed from the projection")
	}
}

func TestBuildCRUDMatrixOrigins(t *testing.T) {
	in := fixtureProjectionInput(t, "")
	matrix := BuildCRUDMatrix(in)
	if matrix.SchemaVersion != "tei.subsystem-partition.crud-matrix/v0" {
		t.Fatalf("unexpected schema version %s", matrix.SchemaVersion)
	}
	if matrix.AuthorityLimit != AuthorityLimit {
		t.Fatalf("missing authority limit")
	}
	if matrix.Digest == "" {
		t.Fatalf("missing digest")
	}
	byKey := map[string]CRUDCell{}
	for _, c := range matrix.Cells {
		byKey[c.Function+"|"+c.DataID] = c
	}
	cell, ok := byKey["F1_order_submit|sales.order"]
	if !ok {
		t.Fatalf("missing F1_order_submit|sales.order cell")
	}
	if len(cell.Operations) != 1 || cell.Operations[0] != "C" {
		t.Fatalf("unexpected operations %+v", cell.Operations)
	}
	if len(cell.Provenance) == 0 {
		t.Fatalf("missing provenance origin column")
	}
	read, ok := byKey["F1_order_submit|inventory.stock"]
	if !ok {
		t.Fatalf("missing read cell F1_order_submit|inventory.stock")
	}
	if len(read.Operations) != 1 || read.Operations[0] != "R" {
		t.Fatalf("unexpected read operations %+v", read.Operations)
	}
	if read.Undetermined {
		t.Fatalf("parsed-extracted read must not be marked undetermined")
	}
	// no cell may render "no operation" as an empty list.
	for _, c := range matrix.Cells {
		if len(c.Operations) == 0 {
			t.Fatalf("cell %s|%s has empty operations", c.Function, c.DataID)
		}
	}
}

func TestBuildDependencyGraphTypedEdges(t *testing.T) {
	in := fixtureProjectionInput(t, "")
	graph := BuildDependencyGraph(in)
	if graph.SchemaVersion != "tei.subsystem-partition.dependency-graph/v0" {
		t.Fatalf("unexpected schema version %s", graph.SchemaVersion)
	}
	if graph.AuthorityLimit != AuthorityLimit {
		t.Fatalf("missing authority limit")
	}
	if graph.Digest == "" {
		t.Fatalf("missing digest")
	}
	kinds := map[string]int{}
	invocationExternal := 0
	for _, e := range graph.Edges {
		kinds[e.Kind]++
		if e.Kind == "invocation" && e.External {
			invocationExternal++
		}
	}
	if kinds["invocation"] == 0 {
		t.Fatalf("expected invocation edges")
	}
	if kinds["function-data-write"] == 0 {
		t.Fatalf("expected function-data-write edges")
	}
	if invocationExternal != 0 {
		t.Fatalf("delegation must never be drawn as an external boundary")
	}
	for _, e := range graph.Edges {
		if e.Kind == "function-data-write" && e.To == "external.tax_ledger" && !e.External {
			t.Fatalf("external-owned data write must be marked external")
		}
	}
}

func TestProjectionDigestSharedAcrossOutputs(t *testing.T) {
	in := fixtureProjectionInput(t, "")
	er := BuildERProjection(in)
	crud := BuildCRUDMatrix(in)
	graph := BuildDependencyGraph(in)
	if er.Digest != crud.Digest || crud.Digest != graph.Digest {
		t.Fatalf("digests differ across projections: er=%s crud=%s graph=%s", er.Digest, crud.Digest, graph.Digest)
	}
	if er.CandidateID != crud.CandidateID || crud.CandidateID != graph.CandidateID {
		t.Fatalf("candidate ids differ across projections")
	}
}

func TestProjectionDigestChangesWithCandidate(t *testing.T) {
	first := fixtureProjectionInput(t, "cand-001")
	second := fixtureProjectionInput(t, "cand-002")
	a := BuildERProjection(first)
	b := BuildERProjection(second)
	if a.Digest == b.Digest {
		t.Fatalf("different candidates must produce different digests")
	}
}

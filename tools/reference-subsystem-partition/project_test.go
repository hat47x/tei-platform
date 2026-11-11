package partition

import (
	"testing"
)

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
	report := BuildProjectionReport(assign, p.Writes, p.ReadEdges, nil, fnIDs, dataIDs)
	if len(report.Subsystems) == 0 {
		t.Fatal("projection produced no subsystems")
	}
	// every assigned function must appear in exactly one subsystem.
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
	// no unlisted data id may leak.
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

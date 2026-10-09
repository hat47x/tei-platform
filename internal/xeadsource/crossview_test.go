package xeadsource

import (
	"encoding/json"
	"strings"
	"testing"
)

func crossViewMappingFingerprint() string {
	return strings.Repeat("a", 64)
}

func crossViewInspection(t *testing.T, source string, fingerprint string) Inspection {
	t.Helper()
	return Inspection{
		Document:                         parseCrossViewFixture(t, source),
		TranscriptionContractFingerprint: fingerprint,
	}
}

func parseCrossViewFixture(t *testing.T, source string) Document {
	t.Helper()
	doc, err := Parse(FormatXead, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestCrossViewSeparatesSourceEdgesAndPotentialActionEffects(t *testing.T) {
	doc := parseCrossViewFixture(t, `<System>
  <Table ID="T1"/>
  <Function ID="F1">
    <IOTable ID="IO1" TableID="T1" OpR="+"/>
    <IOPanel ID="P1"/>
  </Function>
  <Task ID="TASK1">
    <TaskAction SortKey="1">
      <TaskFunctionIO FunctionID="F1" IOID="P1" Operations="press"/>
      <TaskFunctionIO FunctionID="F1" IOID="P1" Operations="enter"/>
    </TaskAction>
    <TaskAction ID="A2"/>
  </Task>
</System>`)

	projection, err := BuildCrossViewProjection(Inspection{Document: doc, TranscriptionContractFingerprint: crossViewMappingFingerprint()})
	if err != nil {
		t.Fatal(err)
	}
	accounting := CrossViewPublicAccounting(projection)

	if accounting.TaskCount != 1 ||
		accounting.TaskActionCount != 2 ||
		accounting.TaskActionWithoutFunctionIOCount != 1 ||
		accounting.TaskActionMissingIDCount != 1 ||
		accounting.TaskFunctionIOCount != 2 ||
		accounting.IOTableCount != 1 {
		t.Fatalf("unexpected accounting: %+v", accounting)
	}
	if accounting.SourceEdgeCounts["task-action"] != 2 {
		t.Fatalf("task-action edges=%d", accounting.SourceEdgeCounts["task-action"])
	}
	if accounting.SourceEdgeCounts["action-function-use"] != 2 {
		t.Fatalf("action-function-use edges=%d", accounting.SourceEdgeCounts["action-function-use"])
	}
	if accounting.SourceEdgeCounts["task-function-io-target"] != 2 {
		t.Fatalf("target edges=%d", accounting.SourceEdgeCounts["task-function-io-target"])
	}
	if accounting.SourceEdgeCounts["function-table-operation"] != 1 {
		t.Fatalf("function-table-operation edges=%d", accounting.SourceEdgeCounts["function-table-operation"])
	}
	if accounting.DistinctActionFunctionPairCount != 1 {
		t.Fatalf("distinct action/function pairs=%d want=1", accounting.DistinctActionFunctionPairCount)
	}
	if accounting.PotentialActionFunctionTablePathCount != 1 ||
		accounting.PotentialOperationCombinationCounts["R"] != 1 {
		t.Fatalf("potential paths=%+v", accounting)
	}

	if len(projection.DerivedEdges) != 1 {
		t.Fatalf("derived edges=%d want=1", len(projection.DerivedEdges))
	}
	edge := projection.DerivedEdges[0]
	if edge.Kind != "action-function-table-potential-operation" ||
		edge.Nature != CrossViewNatureDerived ||
		edge.Operations != "R" {
		t.Fatalf("derived edge=%+v", edge)
	}
	if edge.DerivationRuleFingerprint != projection.DerivationPolicyFingerprint {
		t.Fatalf("derived rule fingerprint=%s projection=%s", edge.DerivationRuleFingerprint, projection.DerivationPolicyFingerprint)
	}
	if !containsString(edge.AuthorityLimits, "derived-path-is-not-business-action-specific-effect") {
		t.Fatalf("action-specific effect limit missing: %+v", edge.AuthorityLimits)
	}
	if len(edge.SourceTokens) != 6 {
		t.Fatalf("derived support token count=%d want=6: %v", len(edge.SourceTokens), edge.SourceTokens)
	}

	if len(projection.Actions) != 2 {
		t.Fatalf("actions=%d want=2", len(projection.Actions))
	}
	var manualCount int
	for _, action := range projection.Actions {
		if action.ManualOrExternalCandidate {
			manualCount++
		}
	}
	if manualCount != 1 {
		t.Fatalf("manual/external candidate count=%d want=1", manualCount)
	}
}

func TestCrossViewFailsClosedOnAmbiguousFunctionIdentity(t *testing.T) {
	doc := parseCrossViewFixture(t, `<System>
  <Table ID="T1"/>
  <Function ID="F1"><IOTable ID="IO1" TableID="T1" OpR="+"/><IOPanel ID="P1"/></Function>
  <Function ID="F1"><IOPanel ID="P1"/></Function>
  <Task ID="TASK1">
    <TaskAction ID="A1"><TaskFunctionIO FunctionID="F1" IOID="P1"/></TaskAction>
  </Task>
</System>`)
	projection, err := BuildCrossViewProjection(Inspection{Document: doc, TranscriptionContractFingerprint: crossViewMappingFingerprint()})
	if err != nil {
		t.Fatal(err)
	}
	if hasCrossViewEdge(projection.SourceEdges, "action-function-use") {
		t.Fatal("ambiguous Function ID produced an action/function edge")
	}
	if len(projection.DerivedEdges) != 0 {
		t.Fatalf("ambiguous Function ID produced derived paths: %+v", projection.DerivedEdges)
	}
	if !hasCrossViewUnresolved(projection.Unresolved, "task-function-io-function-ambiguous") {
		t.Fatalf("missing ambiguity diagnostic: %+v", projection.Unresolved)
	}
}

func TestCrossViewKeepsUnresolvedIOTableInSourceTotals(t *testing.T) {
	doc := parseCrossViewFixture(t, `<System>
  <Table ID="T1"/>
  <Table ID="T1"/>
  <Function ID="F1"><IOTable ID="IO1" TableID="T1" OpU="+"/></Function>
  <Task ID="TASK1"><TaskAction ID="A1"><TaskFunctionIO FunctionID="F1" IOID="0"/></TaskAction></Task>
</System>`)
	projection, err := BuildCrossViewProjection(Inspection{Document: doc, TranscriptionContractFingerprint: crossViewMappingFingerprint()})
	if err != nil {
		t.Fatal(err)
	}
	accounting := CrossViewPublicAccounting(projection)
	if accounting.IOTableCount != 1 {
		t.Fatalf("IOTable source total=%d want=1", accounting.IOTableCount)
	}
	if accounting.SourceEdgeCounts["function-table-operation"] != 0 {
		t.Fatalf("ambiguous table produced resolved edge: %+v", accounting.SourceEdgeCounts)
	}
	if accounting.TaskFunctionIONoTargetCount != 1 {
		t.Fatalf("IOID=0 count=%d want=1", accounting.TaskFunctionIONoTargetCount)
	}
	if !hasCrossViewUnresolved(projection.Unresolved, "iotable-table-ambiguous") {
		t.Fatalf("missing table ambiguity diagnostic: %+v", projection.Unresolved)
	}
	if len(projection.DerivedEdges) != 0 {
		t.Fatalf("unresolved IOTable produced potential path: %+v", projection.DerivedEdges)
	}
}

func TestCrossViewDoesNotPromoteNamespacedLookalike(t *testing.T) {
	doc := parseCrossViewFixture(t, `<System xmlns:x="urn:private">
  <Table ID="T1"/>
  <Function ID="F1"><IOTable ID="IO1" TableID="T1" OpR="+"/><IOPanel ID="P1"/></Function>
  <Task ID="TASK1">
    <TaskAction ID="A1"><x:TaskFunctionIO FunctionID="F1" IOID="P1"/></TaskAction>
  </Task>
</System>`)
	projection, err := BuildCrossViewProjection(Inspection{Document: doc, TranscriptionContractFingerprint: crossViewMappingFingerprint()})
	if err != nil {
		t.Fatal(err)
	}
	accounting := CrossViewPublicAccounting(projection)
	if accounting.TaskFunctionIOCount != 0 {
		t.Fatalf("namespaced lookalike counted as TaskFunctionIO: %+v", accounting)
	}
	if accounting.TaskActionWithoutFunctionIOCount != 1 {
		t.Fatalf("action with only private lookalike must remain candidate manual/external: %+v", accounting)
	}
	if hasCrossViewEdge(projection.SourceEdges, "action-function-use") {
		t.Fatal("namespaced TaskFunctionIO produced standard source edge")
	}
}

func TestCrossViewDoesNotTreatNamespacedAttributeAsFunctionID(t *testing.T) {
	doc := parseCrossViewFixture(t, `<System xmlns:x="urn:private">
  <Function ID="F1"><IOPanel ID="P1"/></Function>
  <Task ID="TASK1">
    <TaskAction ID="A1"><TaskFunctionIO x:FunctionID="F1" IOID="P1"/></TaskAction>
  </Task>
</System>`)
	projection, err := BuildCrossViewProjection(Inspection{Document: doc, TranscriptionContractFingerprint: crossViewMappingFingerprint()})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCrossViewUnresolved(projection.Unresolved, "task-function-io-function-id-missing") {
		t.Fatalf("namespaced FunctionID was treated as standard attribute: %+v", projection.Unresolved)
	}
	if hasCrossViewEdge(projection.SourceEdges, "action-function-use") {
		t.Fatal("namespaced FunctionID produced action/function edge")
	}
}

func TestCrossViewRejectsPartialCRUDWhenMarkerIsUnsupported(t *testing.T) {
	doc := parseCrossViewFixture(t, `<System>
  <Table ID="T1"/>
  <Function ID="F1"><IOTable ID="IO1" TableID="T1" OpR="+" OpU="?"/></Function>
  <Task ID="TASK1"><TaskAction ID="A1"><TaskFunctionIO FunctionID="F1" IOID="0"/></TaskAction></Task>
</System>`)
	projection, err := BuildCrossViewProjection(Inspection{Document: doc, TranscriptionContractFingerprint: crossViewMappingFingerprint()})
	if err != nil {
		t.Fatal(err)
	}
	if hasCrossViewEdge(projection.SourceEdges, "function-table-operation") {
		t.Fatalf("partially understood CRUD produced a resolved source edge: %+v", projection.SourceEdges)
	}
	if len(projection.DerivedEdges) != 0 {
		t.Fatalf("partially understood CRUD produced a derived action/table path: %+v", projection.DerivedEdges)
	}
	if !hasCrossViewUnresolved(projection.Unresolved, "iotable-operation-marker-unsupported") {
		t.Fatalf("unsupported CRUD marker was not retained as unresolved: %+v", projection.Unresolved)
	}
	accounting := CrossViewPublicAccounting(projection)
	if accounting.IOTableCount != 1 {
		t.Fatalf("source IOTable count=%d want=1", accounting.IOTableCount)
	}
}


func TestCrossViewBindsProjectionToSourceContractAndRule(t *testing.T) {
	sourceA := `<System><Table ID="T1"/><Function ID="F1"><IOTable TableID="T1" OpR="+"/></Function></System>`
	sourceB := `<System><Table ID="T1"/><Function ID="F1"><IOTable TableID="T1" OpU="+"/></Function></System>`
	left, err := BuildCrossViewProjection(crossViewInspection(t, sourceA, strings.Repeat("1", 64)))
	if err != nil {
		t.Fatal(err)
	}
	contractChanged, err := BuildCrossViewProjection(crossViewInspection(t, sourceA, strings.Repeat("2", 64)))
	if err != nil {
		t.Fatal(err)
	}
	sourceChanged, err := BuildCrossViewProjection(crossViewInspection(t, sourceB, strings.Repeat("1", 64)))
	if err != nil {
		t.Fatal(err)
	}

	if left.ProjectionFingerprint == contractChanged.ProjectionFingerprint {
		t.Fatal("mapping contract drift did not stale the cross-view projection")
	}
	if left.ProjectionFingerprint == sourceChanged.ProjectionFingerprint {
		t.Fatal("source drift did not stale the cross-view projection")
	}
	if left.DerivationPolicyFingerprint != contractChanged.DerivationPolicyFingerprint ||
		left.DerivationPolicyFingerprint != sourceChanged.DerivationPolicyFingerprint {
		t.Fatal("source/contract drift unexpectedly changed derivation rule identity")
	}
}

func TestCrossViewPublicAccountingOmitsIndividualRelationTokens(t *testing.T) {
	doc := parseCrossViewFixture(t, `<System>
  <Table ID="private-table"/>
  <Function ID="private-function"><IOTable ID="private-io" TableID="private-table" OpR="+"/></Function>
  <Task ID="private-task"><TaskAction ID="private-action"><TaskFunctionIO FunctionID="private-function" IOID="0"/></TaskAction></Task>
</System>`)
	projection, err := BuildCrossViewProjection(Inspection{Document: doc, TranscriptionContractFingerprint: crossViewMappingFingerprint()})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(CrossViewPublicAccounting(projection))
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"private-table", "private-function", "private-task", "private-action", "fromToken", "toToken", "sourceTokens"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("public accounting leaked relation detail %q: %s", forbidden, text)
		}
	}
}

func TestCrossViewRejectsInspectionWithoutExactContractBinding(t *testing.T) {
	doc := parseCrossViewFixture(t, `<System/>`)
	for _, fingerprint := range []string{"", "abc", "sha256:" + strings.Repeat("A", 64), "sha256:" + strings.Repeat("1", 63)} {
		if _, err := BuildCrossViewProjection(Inspection{
			Document:                         doc,
			TranscriptionContractFingerprint: fingerprint,
		}); err == nil {
			t.Fatalf("invalid mapping fingerprint accepted: %q", fingerprint)
		}
	}
}

func hasCrossViewEdge(edges []CrossViewEdge, kind string) bool {
	for _, edge := range edges {
		if edge.Kind == kind {
			return true
		}
	}
	return false
}

func hasCrossViewUnresolved(values []CrossViewUnresolved, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

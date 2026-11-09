package xeadsource

import (
	"strings"
	"testing"
)

// This synthetic scenario tests cross-view preservation, not an accepted
// X-TEA-to-TEI mapping or an observed business effect.
func TestCrossViewHeaderDetailPotentialOperationsStayUnaccepted(t *testing.T) {
	source := `<System FormatVersion="1.2">
  <Table ID="Header"/>
  <Table ID="Detail"/>
  <Function ID="Edit">
    <IOTable ID="HeaderIO" TableID="Header" OpR="+" OpU="+"/>
    <IOTable ID="DetailIO" TableID="Detail" OpC="+" OpR="+" OpU="+" OpD="+"/>
    <IOPanel ID="Panel"/>
  </Function>
  <Task ID="Task">
    <TaskAction SortKey="0040">
      <TaskFunctionIO FunctionID="Edit" IOID="Panel" Operations="press update"/>
    </TaskAction>
  </Task>
</System>`

	project := func(input string) (CrossViewProjection, CrossViewAccounting) {
		t.Helper()
		p, err := BuildCrossViewProjection(
			FormatXead, []byte(input), validCrossViewTranscriptionContract(),
		)
		if err != nil {
			t.Fatal(err)
		}
		return p, crossViewAccountingForTest(t, p)
	}

	projection, got := project(source)
	if got.TaskCount != 1 ||
		got.TaskActionCount != 1 ||
		got.TaskActionMissingIDCount != 1 ||
		got.TaskFunctionIOCount != 1 ||
		got.TaskFunctionIOTargetKindCounts["IOPanel"] != 1 {
		t.Fatalf("business/interaction source was lost: %+v", got)
	}
	if got.IOTableCount != 2 ||
		got.SourceEdgeCounts["function-table-operation"] != 2 ||
		got.PotentialActionFunctionTablePathCount != 2 ||
		got.PotentialOperationCombinationCounts["RU"] != 1 ||
		got.PotentialOperationCombinationCounts["CRUD"] != 1 {
		t.Fatalf("header/detail potential operations were lost: %+v", got)
	}

	seen := map[string]bool{}
	if len(projection.DerivedEdges) != 2 {
		t.Fatalf("expected two derived paths, got: %+v", projection.DerivedEdges)
	}
	for _, edge := range projection.DerivedEdges {
		if edge.Kind != "action-function-table-potential-operation" ||
			edge.Nature != CrossViewNatureDerived ||
			edge.DerivationRuleFingerprint != projection.DerivationPolicyFingerprint ||
			len(edge.SourceTokens) != 5 ||
			!containsString(edge.AuthorityLimits, "derived-path-is-not-business-action-specific-effect") ||
			!containsString(edge.AuthorityLimits, "derived-path-is-not-runtime-observation") {
			t.Fatalf("derived path gained unsupported authority or lost provenance: %+v", edge)
		}
		seen[edge.Operations] = true
	}
	if !seen["RU"] || !seen["CRUD"] {
		t.Fatalf("unexpected derived operations: %+v", projection.DerivedEdges)
	}

	// The detail IOTable still exists, but its target table is missing.
	// Source accounting must retain it without inventing an update effect.
	changed := strings.Replace(source, `<Table ID="Detail"/>`, `<Table ID="Other"/>`, 1)
	changedProjection, changedGot := project(changed)
	if changedGot.IOTableCount != 2 ||
		changedGot.SourceEdgeCounts["function-table-operation"] != 1 ||
		changedGot.UnresolvedCounts["iotable-table-not-found"] != 1 ||
		changedGot.PotentialActionFunctionTablePathCount != 1 ||
		changedGot.PotentialOperationCombinationCounts["RU"] != 1 ||
		changedGot.PotentialOperationCombinationCounts["CRUD"] != 0 {
		t.Fatalf("unresolved detail was silently promoted or discarded: %+v", changedGot)
	}
	if projection.SourceArtifactDigest == changedProjection.SourceArtifactDigest ||
		projection.ProjectionFingerprint == changedProjection.ProjectionFingerprint {
		t.Fatal("a source change reused the old projection identity")
	}
}

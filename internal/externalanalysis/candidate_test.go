package externalanalysis_test

import (
	"strings"
	"testing"

	"tei/internal/externalanalysis"
)

// TestReadCandidatesFromTheRealEvidence is the T1-A input shape: one received
// Evidence, read for the observations, relations and unknowns it actually
// carries. Every field the coordination needs is asserted against the captured
// bytes, so a producer change fails here instead of quietly redefining what a
// candidate holds.
func TestReadCandidatesFromTheRealEvidence(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	receipt := externalanalysis.Receive(raw, presentTool("6.8.0"), acquisition)
	if receipt.State != externalanalysis.StateComplete {
		t.Fatalf("state=%s reason=%s", receipt.State, receipt.Reason)
	}
	candidates, err := externalanalysis.ReadCandidates(receipt.EvidenceBody())
	if err != nil {
		t.Fatalf("read candidates: %v", err)
	}

	if candidates.EvidenceID != evidenceID68 {
		t.Fatalf("evidence id=%s want %s", candidates.EvidenceID, evidenceID68)
	}
	if candidates.Operation != "analyze_javascript_application" {
		t.Fatalf("operation=%s", candidates.Operation)
	}
	if len(candidates.Observations) != 7 {
		t.Fatalf("observations=%d want 7", len(candidates.Observations))
	}
	if len(candidates.Relations) != 2 {
		t.Fatalf("relations=%d want 2", len(candidates.Relations))
	}
	if len(candidates.Unknowns) == 0 {
		t.Fatal("the producer declared unknowns; a candidate set that drops them overstates the graph")
	}
	if candidates.GraphID == "" {
		t.Fatal("the producer named its graph; the candidates must carry that name")
	}
	if len(candidates.Limitations) == 0 {
		t.Fatal("the producer declared limitations; a candidate set that drops them overstates the graph")
	}
	if candidates.Coverage == nil {
		t.Fatal("the producer declared coverage; a candidate set without it cannot say how much was seen")
	}
	if candidates.Coverage.Status != "unknown" {
		t.Fatalf("coverage status=%s: the producer said the graph is not fully covered", candidates.Coverage.Status)
	}
	if len(candidates.Coverage.Families) == 0 {
		t.Fatal("the producer stated coverage per family; a blanket status loses that")
	}

	// One node observation, field by field.
	node := candidates.Observations[0]
	if node.Kind != "call-site" {
		t.Fatalf("kind=%s", node.Kind)
	}
	if !strings.HasPrefix(node.NodeID, "jsrg_node_") {
		t.Fatalf("node id=%s: the producer's identity is not a TEI subject", node.NodeID)
	}
	if node.ArtifactSHA256 != "e8dc069bd1398a5b3d90db155e379df84a707b163d593a533970e5d67dadaaea" {
		t.Fatalf("artifact digest=%s", node.ArtifactSHA256)
	}
	if node.Location != "source-range:index.js" {
		t.Fatalf("location=%s", node.Location)
	}
	if node.Span == nil || node.Span.StartLine != 2 || node.Span.StartColumn != 6 || node.Span.EndLine != 2 || node.Span.EndColumn != 50 {
		t.Fatalf("span=%+v", node.Span)
	}
	if node.EvidenceContextID == "" {
		t.Fatal("the observation rests on a named evidence context; it must travel with the observation")
	}
	if node.State == "" || node.Confidence == "" || node.Authority == "" {
		t.Fatalf("the evidence context qualifies the claim: state=%q confidence=%q authority=%q", node.State, node.Confidence, node.Authority)
	}
	if !strings.HasPrefix(node.Extractor, "rea-javascript-semantic-analysis@") {
		t.Fatalf("extractor=%s: the producer component and its version must travel with the observation", node.Extractor)
	}

	// One directed edge: the producer's own relation identity, direction,
	// kind, and the producer's resolution.
	edge := candidates.Relations[0]
	if edge.From == "" || edge.To == "" || edge.From == edge.To {
		t.Fatalf("edge endpoints=%+v", edge)
	}
	if edge.Kind != "defines" {
		t.Fatalf("relation kind=%s", edge.Kind)
	}
	if edge.Resolution != "resolved" {
		t.Fatalf("resolution=%s", edge.Resolution)
	}
	if edge.Derived {
		t.Fatal("this producer states its own relation identity; a derived handle must not replace it")
	}
	if !strings.HasPrefix(edge.RelationID, "jsrg_relation_") {
		t.Fatalf("relation id=%s: the producer's identity is not a TEI subject", edge.RelationID)
	}

	// One unknown: the producer named the endpoint it could not resolve, so
	// the candidate set says which node is unbound instead of staying silent.
	unknown := candidates.Unknowns[0]
	if unknown.UnknownID == "" || unknown.NodeID == "" {
		t.Fatalf("unknown=%+v", unknown)
	}
	if unknown.Family == "" || unknown.Reason == "" {
		t.Fatalf("unknown=%+v: the family and the reason are what make the unknown actionable", unknown)
	}

	// A second read of the same Evidence yields the same candidates, so a
	// later review refers to the same things.
	again, err := externalanalysis.ReadCandidates(receipt.EvidenceBody())
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Observations) != len(candidates.Observations) || len(again.Relations) != len(candidates.Relations) {
		t.Fatal("the same Evidence must read as the same candidates")
	}
	if again.Relations[0].RelationID != edge.RelationID {
		t.Fatal("the producer's relation identity must be read, not regenerated")
	}
}

// TestDerivedRelationHandleIsUsedOnlyWhenTheProducerStatesNone: the fallback
// exists for a producer that names an edge only by its endpoints, and it says
// it is TEI's own handle.
func TestDerivedRelationHandleIsUsedOnlyWhenTheProducerStatesNone(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	receipt := externalanalysis.Receive(raw, presentTool("6.8.0"), acquisition)
	body := string(receipt.EvidenceBody())
	// The comma goes with the field: removing only the value would leave a
	// trailing comma and the test would fail on JSON, not on the fallback.
	stripped := strings.Replace(body, `,"relation_id":"jsrg_relation_0f6757a241c56fd84ecaec91f538d8297d222eb5625e4cdbcb2de7603f330718"`, "", 1)
	if stripped == body {
		t.Fatal("the fixture does not carry the relation id the test expects")
	}
	candidates, err := externalanalysis.ReadCandidates([]byte(stripped))
	if err != nil {
		t.Fatalf("read candidates: %v", err)
	}
	edge := candidates.Relations[0]
	if !edge.Derived || !strings.HasPrefix(edge.RelationID, "sha256:") {
		t.Fatalf("relation id=%s derived=%t: without the producer's identity the handle is TEI's own and must say so", edge.RelationID, edge.Derived)
	}
	// The same edge read again gets the same handle, and a different edge does
	// not, so the fallback is stable rather than a fresh random name.
	again, err := externalanalysis.ReadCandidates([]byte(stripped))
	if err != nil {
		t.Fatal(err)
	}
	if again.Relations[0].RelationID != edge.RelationID {
		t.Fatal("the derived handle must be stable across reads of the same Evidence")
	}
	if len(candidates.Relations) > 1 && candidates.Relations[1].RelationID == edge.RelationID {
		t.Fatal("two different edges must not share a derived handle")
	}
}

// TestReadCandidatesNamesWhatItCannotRead: an Evidence that states no semantic
// graph is told apart from one that states an empty graph, and from bytes that
// are not an Evidence at all.
func TestReadCandidatesNamesWhatItCannotRead(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	receipt := externalanalysis.Receive(raw, presentTool("6.8.0"), acquisition)

	noGraph := strings.Replace(string(receipt.EvidenceBody()), `"semantic_graph"`, `"renamed_graph"`, 1)
	if noGraph == string(receipt.EvidenceBody()) {
		t.Fatal("the fixture does not carry the semantic graph the test expects")
	}
	if _, err := externalanalysis.ReadCandidates([]byte(noGraph)); err == nil {
		t.Fatal("an Evidence with no semantic graph must say so")
	} else if !strings.Contains(err.Error(), "semantic graph") {
		t.Fatalf("the failure must name what is absent: %v", err)
	}

	if _, err := externalanalysis.ReadCandidates(nil); err == nil {
		t.Fatal("empty bytes must be refused")
	}
	if _, err := externalanalysis.ReadCandidates([]byte("not json")); err == nil {
		t.Fatal("bytes that are not an Evidence must be refused")
	}
}

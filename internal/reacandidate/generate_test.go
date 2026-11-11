package reacandidate

import (
	"strings"
	"testing"
)

// The first implementation target: one Observation and one directed Relation
// from the same attempt must reach the three-element structure candidates
// with their facts, inference, source provenance and unresolved items intact.
func TestGenerateMinimalObservationAndDirectedRelation(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	if set.SchemaVersion != candidateSetSchemaVersion || set.Effect != EffectReadOnly {
		t.Fatalf("set envelope = %q %q", set.SchemaVersion, set.Effect)
	}
	if set.Qualification != "complete" {
		t.Fatalf("qualification = %q, want complete", set.Qualification)
	}
	data := candidatesOfKind(set, CandidateKindDataElement)
	relations := candidatesOfKind(set, CandidateKindCrossReference)
	if len(data) != 2 || len(relations) != 1 {
		t.Fatalf("candidates = %d data %d relations", len(data), len(relations))
	}

	relation := relations[0]
	if relation.Facts.Direction == nil ||
		relation.Facts.Direction.FromExternalID != "node-order" ||
		relation.Facts.Direction.ToExternalID != "node-line" {
		t.Fatalf("direction lost: %+v", relation.Facts.Direction)
	}
	if relation.Facts.ExternalOccurrenceID != "edge-order-line" {
		t.Fatalf("edge identity lost: %q", relation.Facts.ExternalOccurrenceID)
	}
	if relation.Facts.ObservedKind != "imports" {
		t.Fatalf("relation label lost: %q", relation.Facts.ObservedKind)
	}
	if relation.Inference.TargetElement != "cross-reference" || relation.Inference.SuggestedKind != "data-to-data" {
		t.Fatalf("inference = %+v", relation.Inference)
	}
	if !hasString(relation.Inference.Basis, "rea-directed-edge") {
		t.Fatalf("basis = %v", relation.Inference.Basis)
	}
	if !hasString(relation.Inference.Unresolved, "relation-business-meaning") {
		t.Fatalf("relation business meaning was inferred: %v", relation.Inference.Unresolved)
	}
	if relation.ReviewStatus != ReviewStatusRequiresOwner || relation.Resolution != ResolutionUnresolved {
		t.Fatalf("review/resolution = %q %q", relation.ReviewStatus, relation.Resolution)
	}
	if relation.SourceRevision.Binding != SourceBindingProviderReported ||
		relation.SourceRevision.ArtifactPath != "src/order.js" ||
		relation.SourceRevision.Fingerprint != "sha256:"+digestA {
		t.Fatalf("source revision = %+v", relation.SourceRevision)
	}
	if len(relation.MeaningLoss) == 0 {
		t.Fatal("no meaning loss recorded")
	}
}

// Structural facts and inferred business meaning must stay separable: the
// facts carry no business claim, and the inference names only structural
// basis plus what it did not decide.
func TestGenerateSeparatesStructuralFactsFromInferredMeaning(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	for _, candidate := range set.Candidates {
		facts := candidate.Facts
		if facts.ExternalOccurrenceID == "" || facts.ExternalIdentity == "" {
			t.Fatalf("candidate %s lost an external identity fact", candidate.CandidateID)
		}
		// No fact field may name a business concept.
		if strings.Contains(strings.ToLower(facts.ObservedKind), "customer") ||
			strings.Contains(strings.ToLower(facts.ObservedKind), "order") {
			t.Fatalf("candidate %s carries a business name as a fact", candidate.CandidateID)
		}
		if len(candidate.Inference.Basis) == 0 {
			t.Fatalf("candidate %s has no basis", candidate.CandidateID)
		}
		for _, basis := range candidate.Inference.Basis {
			if strings.HasPrefix(basis, "business-") {
				t.Fatalf("candidate %s claims a business basis: %q", candidate.CandidateID, basis)
			}
		}
		for _, unresolved := range candidate.Inference.Unresolved {
			if unresolved == "" {
				t.Fatalf("candidate %s has an empty unresolved entry", candidate.CandidateID)
			}
		}
	}
}

// Two distinct external nodes that share a display name stay two candidates;
// the collision is reported, never merged by name.
func TestGenerateSameNameDifferentElementsStayDistinct(t *testing.T) {
	fixture := defaultEvidence()
	fixture.nodes[1].observations[0].locSource = "other/order.js"
	fixture.nodes[1].observations[0].locPath = "other/order.js"
	set := mustGenerate(t, fixture)
	data := candidatesOfKind(set, CandidateKindDataElement)
	if len(data) != 2 {
		t.Fatalf("data candidates = %d", len(data))
	}
	if data[0].CandidateID == data[1].CandidateID {
		t.Fatal("distinct external nodes share one candidate identity")
	}
	if data[0].Inference.SuggestedName != data[1].Inference.SuggestedName {
		t.Fatalf("suggested names differ: %q %q", data[0].Inference.SuggestedName, data[1].Inference.SuggestedName)
	}
	for _, candidate := range data {
		if !hasNote(candidate, "same-suggested-name-distinct-external-identity") {
			t.Fatalf("candidate %s does not record the name collision", candidate.CandidateID)
		}
	}
	if !hasNotePrefix(set, "same-suggested-name-distinct-external-identity:order:") {
		t.Fatalf("set does not report the collision: %v", set.Notes)
	}
}

// Parallel relations between the same endpoints are distinct candidates; the
// tool never collapses them into one edge.
func TestGenerateParallelRelationsStayDistinct(t *testing.T) {
	fixture := defaultEvidence()
	second := fixture.edges[0]
	second.id = "edge-order-line-alt"
	second.rel = "requires"
	fixture.edges = append(fixture.edges, second)
	set := mustGenerate(t, fixture)
	relations := candidatesOfKind(set, CandidateKindCrossReference)
	if len(relations) != 2 {
		t.Fatalf("relation candidates = %d, want 2", len(relations))
	}
	if relations[0].CandidateID == relations[1].CandidateID {
		t.Fatal("parallel edges share one candidate identity")
	}
	for _, candidate := range relations {
		if !hasNote(candidate, "parallel-relation") {
			t.Fatalf("candidate %s is not marked parallel", candidate.CandidateID)
		}
		if !hasString(candidate.Inference.Unresolved, "parallel-relation-meaning") {
			t.Fatalf("candidate %s did not leave the parallel meaning unresolved", candidate.CandidateID)
		}
	}
}

// Two edges with the same endpoints AND the same label are a stronger
// ambiguity than parallel edges: both are kept and named separately.
func TestGenerateSameEndpointsSameLabelIsASeparateAmbiguity(t *testing.T) {
	fixture := defaultEvidence()
	second := fixture.edges[0]
	second.id = "edge-order-line-copy"
	fixture.edges = append(fixture.edges, second)
	set := mustGenerate(t, fixture)
	relations := candidatesOfKind(set, CandidateKindCrossReference)
	if len(relations) != 2 {
		t.Fatalf("relation candidates = %d, want 2", len(relations))
	}
	for _, candidate := range relations {
		if !hasNote(candidate, "parallel-relation") || !hasNote(candidate, "same-endpoints-same-label") {
			t.Fatalf("candidate %s notes = %v", candidate.CandidateID, candidate.Notes)
		}
		if !hasString(candidate.Inference.Unresolved, "same-endpoints-same-label-meaning") {
			t.Fatalf("candidate %s did not leave the duplicate label unresolved", candidate.CandidateID)
		}
	}
}

// A→B and B→A are different facts. Both are kept, and the conflict is left
// for the Owner instead of being resolved here.
func TestGenerateDirectionDifferenceIsReportedNotResolved(t *testing.T) {
	fixture := defaultEvidence()
	reverse := fixture.edges[0]
	reverse.id = "edge-line-order"
	reverse.from, reverse.to = "node-line", "node-order"
	fixture.edges = append(fixture.edges, reverse)
	set := mustGenerate(t, fixture)
	relations := candidatesOfKind(set, CandidateKindCrossReference)
	if len(relations) != 2 {
		t.Fatalf("relation candidates = %d, want 2", len(relations))
	}
	forward, backward := 0, 0
	for _, candidate := range relations {
		if !hasNote(candidate, "direction-conflict-pair") {
			t.Fatalf("candidate %s is not marked as a direction conflict", candidate.CandidateID)
		}
		if !hasString(candidate.Inference.Unresolved, "direction-conflict-resolution") {
			t.Fatalf("candidate %s resolved the direction conflict", candidate.CandidateID)
		}
		if candidate.Facts.Direction.FromExternalID == "node-order" {
			forward++
		}
		if candidate.Facts.Direction.FromExternalID == "node-line" {
			backward++
		}
	}
	if forward != 1 || backward != 1 {
		t.Fatalf("directions = forward %d backward %d", forward, backward)
	}
}

// An endpoint the attempt did not observe stays unresolved; no node is
// fabricated to complete the relation.
func TestGenerateUnresolvedEndpointIsNotFabricated(t *testing.T) {
	fixture := defaultEvidence()
	fixture.nodes = fixture.nodes[:1]
	fixture.edges[0].to = "node-missing"
	set := mustGenerate(t, fixture)
	data := candidatesOfKind(set, CandidateKindDataElement)
	if len(data) != 1 {
		t.Fatalf("data candidates = %d, want 1 (no fabricated node)", len(data))
	}
	relation := candidatesOfKind(set, CandidateKindCrossReference)[0]
	if !hasNote(relation, "unresolved-target-endpoint") {
		t.Fatalf("relation does not record the unresolved endpoint: %v", relation.Notes)
	}
	if !hasString(relation.Inference.Unresolved, "target-endpoint-not-observed-in-this-attempt:node-missing") {
		t.Fatalf("unresolved endpoint not named: %v", relation.Inference.Unresolved)
	}
	if set.Qualification != "partial" {
		t.Fatalf("qualification = %q, want partial", set.Qualification)
	}
}

// A self-relation is the degenerate A→A cycle: kept, flagged, not interpreted.
func TestGenerateSelfRelationIsFlagged(t *testing.T) {
	fixture := defaultEvidence()
	fixture.edges[0].from, fixture.edges[0].to = "node-order", "node-order"
	set := mustGenerate(t, fixture)
	relation := candidatesOfKind(set, CandidateKindCrossReference)[0]
	if !hasNote(relation, "self-relation") {
		t.Fatalf("self-relation not flagged: %v", relation.Notes)
	}
	if !hasString(relation.Inference.Unresolved, "self-relation-meaning") {
		t.Fatalf("self-relation meaning inferred: %v", relation.Inference.Unresolved)
	}
}

// REAL REA 6.8.0 reports coverage as {status, truncated, omitted_count, limits}
// and never writes `exhausted`. A decoder that only understood `exhausted`
// would silently call every real attempt "unknown". This test pins the real
// shape so that mistake cannot recur.
func TestGenerateQualifiesWithRealREA6_8_0CoverageShape(t *testing.T) {
	fixture := defaultEvidence()
	fixture.graphExhausted = nil
	fixture.semExhausted = nil
	fixture.realCoverage = true
	fixture.semanticStatus = "unknown"
	set := mustGenerate(t, fixture)
	if set.Qualification == "unknown" {
		t.Fatalf("real coverage shape was not decoded: %+v", set.Attempt.GraphCoverage)
	}
	if set.Qualification != "partial" {
		t.Fatalf("qualification = %q, want partial (semantic coverage says unknown)", set.Qualification)
	}
	if set.Attempt.GraphCoverage == nil || set.Attempt.GraphCoverage.Truncated == nil {
		t.Fatalf("truncated was not carried: %+v", set.Attempt.GraphCoverage)
	}
	if complete, specified := set.Attempt.GraphCoverage.Complete(); !specified || !complete {
		t.Fatalf("truncated=false was not read as an uncut enumeration: %v %v", complete, specified)
	}
}

// truncated=true is a provider statement that the enumeration was cut short,
// which caps the attempt at partial even with no limitations.
func TestGenerateTruncatedCoverageIsPartial(t *testing.T) {
	fixture := defaultEvidence()
	fixture.graphExhausted = nil
	fixture.semExhausted = nil
	fixture.realCoverage = true
	fixture.omittedCount = 3
	set := mustGenerate(t, fixture)
	if set.Qualification != "partial" {
		t.Fatalf("qualification = %q, want partial", set.Qualification)
	}
	if omitted, ok := set.Attempt.GraphCoverage.Omitted(); !ok || omitted != 3 {
		t.Fatalf("omitted_count = %d known=%t", omitted, ok)
	}
}

// REA names the analyzed root "artifact-root", which its own limitation text
// calls a path-independent alias. The value is carried verbatim and marked
// unresolved, so nobody treats it as a resolvable file.
func TestGenerateMarksArtifactRootAlias(t *testing.T) {
	fixture := defaultEvidence()
	fixture.nodes[0].observations[0].artifact = digestA
	fixture.nodes[0].observations[0].locKind = "artifact-path"
	fixture.nodes[0].observations[0].locPath = "artifact-root"
	fixture.nodes[0].observations[0].locSource = "artifact-root"
	set := mustGenerate(t, fixture)
	for _, candidate := range set.Candidates {
		if candidate.Facts.ExternalIdentity != "node-order" || candidate.Kind != CandidateKindDataElement {
			continue
		}
		if candidate.SourceRevision.ArtifactPath != "artifact-root" {
			t.Fatalf("alias was rewritten: %q", candidate.SourceRevision.ArtifactPath)
		}
		if !hasString(candidate.Unresolved, "source-path-is-an-alias-not-a-file") {
			t.Fatalf("alias is not marked unresolved: %v", candidate.Unresolved)
		}
		return
	}
	t.Fatal("the aliased candidate is missing")
}

// REA's own limitations, unknowns and non-exhausted coverage cap the
// qualification at partial. Nothing is promoted to complete.
func TestGeneratePartialAcquisitionIsNeverComplete(t *testing.T) {
	withLimitations := defaultEvidence()
	withLimitations.limitations = []string{"one module unreadable"}
	if set := mustGenerate(t, withLimitations); set.Qualification != "partial" {
		t.Fatalf("limitations qualification = %q", set.Qualification)
	}

	withUnknowns := defaultEvidence()
	withUnknowns.unknowns = []unknownFixture{{id: "u1", family: "unresolved-reference", reason: "target not found"}}
	set := mustGenerate(t, withUnknowns)
	if set.Qualification != "partial" {
		t.Fatalf("unknowns qualification = %q", set.Qualification)
	}
	if len(set.Unknowns) != 1 || set.Unknowns[0].UnknownID != "u1" {
		t.Fatalf("unknowns not carried: %+v", set.Unknowns)
	}

	notExhausted := defaultEvidence()
	notExhausted.graphExhausted = boolPtr(false)
	if set := mustGenerate(t, notExhausted); set.Qualification != "partial" {
		t.Fatalf("non-exhausted qualification = %q", set.Qualification)
	}

	semanticPartial := defaultEvidence()
	semanticPartial.semExhausted = boolPtr(false)
	if set := mustGenerate(t, semanticPartial); set.Qualification != "partial" {
		t.Fatalf("semantic partial qualification = %q", set.Qualification)
	}

	silentCoverage := defaultEvidence()
	silentCoverage.graphExhausted = nil
	silentCoverage.semExhausted = nil
	if set := mustGenerate(t, silentCoverage); set.Qualification != "unknown" {
		t.Fatalf("silent coverage qualification = %q, want unknown", set.Qualification)
	}
}

// A node observed several times yields one candidate per observation, and the
// set records that the candidates describe one external node.
func TestGenerateMultipleObservationsOfOneNode(t *testing.T) {
	fixture := defaultEvidence()
	second := fixture.nodes[0].observations[0]
	second.id = "obs-order-again"
	fixture.nodes[0].observations = append(fixture.nodes[0].observations, second)
	set := mustGenerate(t, fixture)
	data := candidatesOfKind(set, CandidateKindDataElement)
	if len(data) != 3 {
		t.Fatalf("data candidates = %d, want 3", len(data))
	}
	if !hasNotePrefix(set, "node-observed-multiple-times:node-order") {
		t.Fatalf("set does not record the repeated node: %v", set.Notes)
	}
	seen := map[string]bool{}
	for _, candidate := range data {
		if candidate.Facts.ExternalIdentity != "node-order" {
			continue
		}
		if seen[candidate.Facts.ExternalOccurrenceID] {
			t.Fatalf("duplicate observation candidate: %s", candidate.Facts.ExternalOccurrenceID)
		}
		seen[candidate.Facts.ExternalOccurrenceID] = true
	}
}

// A node with no observation produces no candidate and is recorded.
func TestGenerateNodeWithoutObservationProducesNoCandidate(t *testing.T) {
	fixture := defaultEvidence()
	fixture.nodes = append(fixture.nodes, nodeFixture{id: "node-silent", kind: "module"})
	set := mustGenerate(t, fixture)
	if len(candidatesOfKind(set, CandidateKindDataElement)) != 2 {
		t.Fatalf("data candidates = %d, want 2", len(candidatesOfKind(set, CandidateKindDataElement)))
	}
	if !hasNotePrefix(set, "node-without-observation:node-silent") {
		t.Fatalf("silent node not recorded: %v", set.Notes)
	}
}

// An attempt with nothing in it yields an empty, honest set rather than an
// error or a fabricated candidate.
func TestGenerateEmptyAttemptIsEmptyNotAnError(t *testing.T) {
	fixture := defaultEvidence()
	fixture.nodes = nil
	fixture.edges = nil
	fixture.graphExhausted = boolPtr(true)
	fixture.semExhausted = boolPtr(true)
	set := mustGenerate(t, fixture)
	if len(set.Candidates) != 0 {
		t.Fatalf("candidates = %d, want 0", len(set.Candidates))
	}
	if set.Qualification != "complete" {
		t.Fatalf("qualification = %q", set.Qualification)
	}
	if set.Fingerprint == "" {
		t.Fatal("empty set has no fingerprint")
	}
}

// Candidate identity follows the external occurrence, not the bytes: the same
// occurrence over changed bytes keeps its identity (so a revision change is
// reported as a change of that candidate), while the set fingerprint, which
// includes the source revision, does change.
func TestGenerateCandidateIdentityFollowsOccurrenceNotAttempt(t *testing.T) {
	first := mustGenerate(t, defaultEvidence())
	second := mustGenerate(t, defaultEvidence())
	if first.Fingerprint != second.Fingerprint {
		t.Fatal("identical attempts produced different set fingerprints")
	}
	otherAttempt := defaultEvidence()
	otherAttempt.evidenceID = "ev-minimal-2"
	third := mustGenerate(t, otherAttempt)
	if third.Fingerprint != first.Fingerprint {
		t.Fatal("a new attempt over the same bytes changed the candidate content")
	}
	changedBytes := defaultEvidence()
	changedBytes.nodes[0].observations[0].artifact = digestB
	fourth := mustGenerate(t, changedBytes)
	if fourth.Fingerprint == first.Fingerprint {
		t.Fatal("changed source bytes kept the same candidate content")
	}
	if len(fourth.Candidates) != len(first.Candidates) {
		t.Fatalf("changed bytes changed the candidate count: %d vs %d", len(fourth.Candidates), len(first.Candidates))
	}
	for _, before := range first.Candidates {
		after, ok := candidateByID(fourth, before.CandidateID)
		if !ok {
			t.Fatalf("candidate %s changed identity when only the bytes changed", before.CandidateID)
		}
		// Only the node whose artifact changed must show a new revision; the
		// relation and the other node keep their own provider-reported source.
		if before.Kind != CandidateKindDataElement || before.Facts.ExternalIdentity != "node-order" {
			continue
		}
		if before.SourceRevision.Fingerprint == after.SourceRevision.Fingerprint {
			t.Fatalf("candidate %s kept the old source fingerprint", before.CandidateID)
		}
		if after.SourceRevision.Fingerprint != "sha256:"+digestB {
			t.Fatalf("candidate %s fingerprint = %q", before.CandidateID, after.SourceRevision.Fingerprint)
		}
	}
}

func TestGenerateSetFingerprintExcludesAttemptProvenance(t *testing.T) {
	first := mustGenerate(t, defaultEvidence())
	other := defaultEvidence()
	other.evidenceID = "ev-other"
	other.providerVersion = "6.9.0"
	second := mustGenerate(t, other)
	if first.Fingerprint != second.Fingerprint {
		t.Fatal("attempt provenance leaked into the content fingerprint")
	}
	if first.Attempt.EvidenceID == second.Attempt.EvidenceID {
		t.Fatal("attempt provenance was not recorded")
	}
}

package reacandidate

import (
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

// One question names a producer edge occurrence, not a generated Go
// method or an accepted Function→Data Read/Write relation. Endpoint source
// occurrences remain distinct even if they share the same display name.
func TestRelationQuestionPreservesProducerIdentityAndAllEndpointObservations(t *testing.T) {
	fixture := defaultEvidence()
	again := fixture.nodes[0].observations[0]
	again.id = "obs-order-second"
	fixture.nodes[0].observations = append(fixture.nodes[0].observations, again)
	parallel := fixture.edges[0]
	parallel.id = "edge-order-line-second"
	fixture.edges = append(fixture.edges, parallel)
	packet := BuildReviewPacket(mustGenerate(t, fixture), nil, nil, nil)
	question, ok := SelectRelationQuestion(packet, "edge-order-line-second")
	if !ok {
		t.Fatal("a distinct producer edge must be selectable")
	}
	if question.Relation.Facts.ExternalOccurrenceID != "edge-order-line-second" ||
		question.Relation.Facts.ExternalIdentity != "edge-order-line-second" ||
		question.Relation.Facts.Direction.FromExternalID != "node-order" ||
		question.Relation.Facts.Direction.ToExternalID != "node-line" {
		t.Fatalf("producer relation occurrence/direction was changed: %+v", question.Relation)
	}
	if len(question.FromObservations) != 2 || len(question.ToObservations) != 1 ||
		question.FromObservations[0].Facts.ExternalOccurrenceID == question.FromObservations[1].Facts.ExternalOccurrenceID {
		t.Fatalf("endpoint observation multiplicity was lost: %+v", question)
	}
	if question.Relation.SourceRevision.ArtifactPath != "src/order.js" ||
		question.Relation.Facts.ObservedKind != "imports" ||
		question.Relation.SourceRevision.Binding != SourceBindingProviderReported {
		t.Fatalf("source/label was lost or silently independently bound: %+v", question.Relation)
	}
	if question.Effect != EffectReadOnly ||
		question.FunctionDataClaim != "not-established-by-external-structure" ||
		question.AcceptedMeaning || question.OwnerDecision != "pending" ||
		question.AttemptEvidenceID != packet.Attempt.EvidenceID ||
		question.CandidateSetFingerprint != packet.CandidateSetFingerprint ||
		!hasString(question.Unresolved, "function-data-read-write-effect-not-observed") {
		t.Fatalf("static relation was promoted to Function/Data meaning: %+v", question)
	}
	if !hasString(question.Relation.Notes, "parallel-relation") {
		t.Fatalf("parallel occurrence is no longer reviewable: %+v", question.Relation.Notes)
	}
}

func TestRelationQuestionDoesNotInventMissingEndpointsOrUnknownCoverage(t *testing.T) {
	fixture := defaultEvidence()
	fixture.nodes = fixture.nodes[:1]
	fixture.edges[0].to = "node-not-in-this-attempt"
	fixture.unknowns = []unknownFixture{{id: "unknown-input", family: "source", reason: "unavailable"}}
	packet := BuildReviewPacket(mustGenerate(t, fixture), nil, nil, nil)
	question, ok := SelectRelationQuestion(packet, "edge-order-line")
	if !ok || question.Qualification != "partial" ||
		len(question.FromObservations) != 1 || len(question.ToObservations) != 0 ||
		len(question.MissingEndpoints) != 1 ||
		question.MissingEndpoints[0] != "target:node-not-in-this-attempt" ||
		len(question.Unknowns) != 1 || !hasString(question.Unresolved, "provider-unknown:unknown-input") {
		t.Fatalf("missing/unresolved endpoint was silently filled: %+v (selected=%t)", question, ok)
	}
}

func TestRelationQuestionCannotPairUnselectedOrDuplicateEdgeIDs(t *testing.T) {
	packet := BuildReviewPacket(mustGenerate(t, defaultEvidence()), nil, nil, nil)
	for _, id := range []string{"", "missing-edge-id", "node-order", "obs-order"} {
		if selected, ok := SelectRelationQuestion(packet, id); ok {
			t.Fatalf("%q invented a relation selection: %+v", id, selected)
		}
	}
	// A corrupt or manually assembled review packet must not arbitrarily
	// choose one of two candidates with the same producer occurrence ID.
	for _, candidate := range packet.Candidates {
		if candidate.Kind == CandidateKindCrossReference {
			packet.Candidates = append(packet.Candidates, candidate)
			break
		}
	}
	if selected, ok := SelectRelationQuestion(packet, "edge-order-line"); ok {
		t.Fatalf("duplicate relation occurrence was selected: %+v", selected)
	}
}

// The packet is the single review surface. It reports what was attempted and
// what was not, and it never claims an acceptance.
func TestReviewPacketReportsSectionsAndAcceptsNothing(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	binding, err := BindCandidates(fakeReadOnlyModel{index: boundFakeIndex()}, set, nil)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	preview, err := PreviewAdoption(set, testModel())
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	before := mustGenerate(t, defaultEvidence())
	comparison := CompareCandidateSets(before, set)

	packet := BuildReviewPacket(set, &binding, &preview, &comparison)
	if packet.SchemaVersion != candidateReviewSchemaVers || packet.Effect != EffectReadOnly {
		t.Fatalf("packet envelope = %q %q", packet.SchemaVersion, packet.Effect)
	}
	// The literal, not only the constant: a caller branches on this string.
	if packet.SchemaVersion != "tei.rea.candidate-review/v0" {
		t.Fatalf("review schemaVersion = %q", packet.SchemaVersion)
	}
	if packet.AcceptedMeaning {
		t.Fatal("packet claims an accepted meaning")
	}
	if packet.OwnerDecision != "pending" {
		t.Fatalf("owner decision = %q", packet.OwnerDecision)
	}
	if !packet.BindingAttempted || !packet.AdoptionPreviewed || !packet.ComparedWithBefore {
		t.Fatalf("sections = %t %t %t", packet.BindingAttempted, packet.AdoptionPreviewed, packet.ComparedWithBefore)
	}
	if len(packet.Candidates) != len(set.Candidates) {
		t.Fatalf("candidates = %d", len(packet.Candidates))
	}
	if packet.CandidateSetFingerprint != set.Fingerprint {
		t.Fatal("candidate set fingerprint lost")
	}
	if len(packet.ReviewSequence) == 0 || len(packet.AuthorityLimits) == 0 {
		t.Fatal("packet lost the review sequence or authority limits")
	}
	if !hasString(packet.AuthorityLimits, "accepted-meaning-belongs-to-the-owner-outside-this-tool") {
		t.Fatalf("authority limits = %v", packet.AuthorityLimits)
	}
	if len(packet.Unresolved) == 0 {
		t.Fatal("packet reports nothing unresolved")
	}
	if !hasString(packet.Unresolved, "canonical-subject-mapping") {
		t.Fatalf("unresolved = %v", packet.Unresolved)
	}
	if !hasString(packet.Unresolved, "no-owner-mapping") {
		t.Fatalf("binding notes not aggregated: %v", packet.Unresolved)
	}
	joined := strings.Join(packet.Summary(), "\n")
	for _, want := range []string{
		"attempt: ev-minimal-1", "qualification: complete", "accepted-meaning: false",
		"owner-decision: pending", "binding: resolved=0", "adoption-preview: in-memory only, not written",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("summary is missing %q:\n%s", want, joined)
		}
	}
}

// A packet built without the optional sections reports them as not attempted
// rather than dropping them from the summary.
func TestReviewPacketWithoutOptionalSections(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	packet := BuildReviewPacket(set, nil, nil, nil)
	if packet.BindingAttempted || packet.AdoptionPreviewed || packet.ComparedWithBefore {
		t.Fatal("packet claims a section it does not have")
	}
	if packet.Bindings != nil || packet.Adoption != nil || packet.Comparison != nil {
		t.Fatal("packet carries a nil section as data")
	}
	joined := strings.Join(packet.Summary(), "\n")
	if !strings.Contains(joined, "binding: not-attempted") ||
		!strings.Contains(joined, "comparison: not-attempted") {
		t.Fatalf("summary:\n%s", joined)
	}
}

// A source digest conflict is aggregated as an unresolved item, because the
// analyzed bytes and the current bytes disagree.
func TestReviewPacketAggregatesSourceDigestConflict(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	model := fakeReadOnlyModel{index: SourceIndexView{
		Artifacts: []ArtifactRecord{{Path: "src/order.js", Fingerprint: "sha256:" + digestB}},
	}}
	binding, err := BindCandidates(model, set, nil)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	packet := BuildReviewPacket(set, &binding, nil, nil)
	if !hasString(packet.Unresolved, "source-digest-conflict-requires-fresh-analysis") {
		t.Fatalf("unresolved = %v", packet.Unresolved)
	}
}

// A partial attempt is aggregated as an unresolved qualification.
func TestReviewPacketAggregatesPartialQualification(t *testing.T) {
	fixture := defaultEvidence()
	fixture.limitations = []string{"one module unreadable"}
	set := mustGenerate(t, fixture)
	packet := BuildReviewPacket(set, nil, nil, nil)
	if packet.Qualification != "partial" {
		t.Fatalf("qualification = %q", packet.Qualification)
	}
	if !hasString(packet.Unresolved, "attempt-qualification:partial") {
		t.Fatalf("unresolved = %v", packet.Unresolved)
	}
}

// Provider unknowns are aggregated with their identity, not dropped.
func TestReviewPacketAggregatesProviderUnknowns(t *testing.T) {
	fixture := defaultEvidence()
	fixture.unknowns = []unknownFixture{{id: "u1", family: "unresolved-reference", reason: "missing target"}}
	set := mustGenerate(t, fixture)
	packet := BuildReviewPacket(set, nil, nil, nil)
	if !hasString(packet.Unresolved, "provider-unknown:u1") {
		t.Fatalf("unresolved = %v", packet.Unresolved)
	}
	if len(packet.Unknowns) != 1 || packet.Unknowns[0].Reason != "missing target" {
		t.Fatalf("unknowns = %+v", packet.Unknowns)
	}
}

// The unresolved summary is bounded so a hostile attempt cannot turn a review
// packet into an unreadable document.
func TestReviewPacketBoundsUnresolvedSummary(t *testing.T) {
	fixture := defaultEvidence()
	fixture.unknowns = nil
	for i := 0; i < maxUnresolvedSummary+50; i++ {
		fixture.unknowns = append(fixture.unknowns, unknownFixture{
			id:     "u" + itoa(i),
			family: "f",
			reason: "r",
		})
	}
	set := mustGenerate(t, fixture)
	packet := BuildReviewPacket(set, nil, nil, nil)
	if len(packet.Unresolved) > maxUnresolvedSummary+1 {
		t.Fatalf("unresolved = %d, bound %d", len(packet.Unresolved), maxUnresolvedSummary)
	}
	if !hasString(packet.Unresolved, "unresolved-summary-truncated") {
		t.Fatalf("truncation not reported: %d entries", len(packet.Unresolved))
	}
}

// A candidate listing names the external identity, the source and the review
// state without ever including source bytes.
func TestCandidateLineNamesIdentitySourceAndReviewState(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	for _, candidate := range set.Candidates {
		line := CandidateLine(candidate)
		if !strings.Contains(line, candidate.Facts.ExternalIdentity) {
			t.Fatalf("line = %q", line)
		}
		if !strings.Contains(line, candidate.SourceRevision.Binding) {
			t.Fatalf("line = %q", line)
		}
		if !strings.Contains(line, ResolutionUnresolved) {
			t.Fatalf("line = %q", line)
		}
		if strings.Contains(line, "\n") {
			t.Fatalf("line contains a newline: %q", line)
		}
	}
}

func TestCandidateLineIncludesDirection(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	relation := candidatesOfKind(set, CandidateKindCrossReference)[0]
	line := CandidateLine(relation)
	if !strings.Contains(line, "node-order->node-line") {
		t.Fatalf("line = %q", line)
	}
}

// The packet is JSON-serializable so a CLI or a reviewer tool can read it.
func TestReviewPacketIsJSONSerializable(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	packet := BuildReviewPacket(set, nil, nil, nil)
	raw, err := jsonMarshal(packet)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"acceptedMeaning":false`) {
		t.Fatalf("serialized packet = %s", raw)
	}
}

func TestSubjectOutcomeAndLinkTypesArePlainData(t *testing.T) {
	outcome := SubjectOutcome{Status: "found", SourceStatus: "indexed"}
	link := ModelMapLink{FromKind: "semantic-data", FromID: "a", ToKind: "semantic-data", ToID: "b", Relation: "refers-to", Resolution: "resolved"}
	if outcome.Status == "" || link.Relation == "" {
		t.Fatal("port types lost their data")
	}
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	if subject.Kind != "semantic-data" {
		t.Fatal("subject identity lost")
	}
}

// Each projection in this package publishes its own response identifier. This
// case names every literal the source writes, so a rename is a deliberate
// contract change rather than a quiet edit.
func TestProjectionsNameTheirSchemaVersions(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	binding, err := BindCandidates(fakeReadOnlyModel{index: boundFakeIndex()}, set, nil)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	preview, err := PreviewAdoption(set, testModel())
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	before := mustGenerate(t, defaultEvidence())
	comparison := CompareCandidateSets(before, set)
	coverage := EvaluateThreeElementCoverage(set)

	want := []struct {
		label string
		got   string
		value string
	}{
		{"candidate-set", set.SchemaVersion, "tei.rea.candidate-set/v0"},
		{"candidate-binding", binding.SchemaVersion, "tei.rea.candidate-binding/v0"},
		{"adoption-preview", preview.SchemaVersion, "tei.rea.adoption-preview/v0"},
		{"candidate-set-comparison", comparison.SchemaVersion, "tei.rea.candidate-set-comparison/v0"},
		{"three-element-coverage", coverage.SchemaVersion, "tei.rea.three-element-coverage/v0"},
	}
	for _, item := range want {
		if item.got != item.value {
			t.Errorf("%s schemaVersion = %q, want %q", item.label, item.got, item.value)
		}
	}
}

// The candidate set comparison is not merely present in JSON: a read-only
// reviewer seeing only Summary must still notice unmatched parallel relations.
func TestReviewSummaryReportsUnpairedParallelRelations(t *testing.T) {
	beforeFixture := defaultEvidence()
	parallel := beforeFixture.edges[0]
	parallel.id = "edge-parallel-2"
	beforeFixture.edges = append(beforeFixture.edges, parallel)
	before := mustGenerate(t, beforeFixture)

	afterFixture := defaultEvidence()
	afterFixture.evidenceID = "ev-summary-renumbered"
	first := afterFixture.edges[0]
	first.id = "edge-renumbered-1"
	second := first
	second.id = "edge-renumbered-2"
	afterFixture.edges = []edgeFixture{second, first}
	after := mustGenerate(t, afterFixture)

	comparison := CompareCandidateSets(before, after)
	packet := BuildReviewPacket(after, nil, nil, &comparison)
	joined := strings.Join(packet.Summary(), "\n")
	for _, want := range []string{
		"comparison: read-only; no owner acceptance or source-edit authority",
		"comparison: ambiguous-structural-match=4",
		"accepted-meaning: false",
		"owner-decision: pending",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("review summary lost %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "comparison: external-identity-renumbered=") {
		t.Fatalf("summary claimed a definite relation counterpart:\n%s", joined)
	}
}

// Unknown statuses or inconsistent comparison sections remain visible rather
// than being silently treated as unchanged by a summary-only consumer.
func TestReviewSummaryDoesNotCoerceUnknownOrInvalidComparison(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	comparison := CandidateSetComparison{
		SchemaVersion: "tei.rea.candidate-set-comparison/v0",
		Effect:        EffectReadOnly,
		Entries: []CandidateComparisonEntry{
			{CandidateID: "first", Status: ComparisonUnchanged},
			{CandidateID: "second", Status: "future-private-status-value"},
		},
	}
	packet := BuildReviewPacket(set, nil, nil, &comparison)
	joined := strings.Join(packet.Summary(), "\n")
	for _, want := range []string{
		"comparison: unchanged=1",
		"comparison: unknown-status=1 (review required)",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("review summary lost %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "future-private-status-value") {
		t.Fatalf("unknown status leaked into review summary:\n%s", joined)
	}
	packet.Comparison = nil
	if !strings.Contains(strings.Join(packet.Summary(), "\n"), "comparison: inconsistent-section") {
		t.Fatal("missing comparison silently accepted as complete")
	}
	packet.ComparedWithBefore = false
	packet.Comparison = &comparison
	if !strings.Contains(strings.Join(packet.Summary(), "\n"), "comparison: inconsistent-section") {
		t.Fatal("unreported comparison silently accepted as not attempted")
	}
	packet.ComparedWithBefore = true
	invalid := comparison
	invalid.SchemaVersion = "tei.rea.candidate-set-comparison/v999"
	packet.Comparison = &invalid
	if !strings.Contains(strings.Join(packet.Summary(), "\n"), "comparison: unsupported-contract") {
		t.Fatal("unknown comparison contract silently accepted")
	}
	invalid = comparison
	invalid.Effect = "write"
	packet.Comparison = &invalid
	if !strings.Contains(strings.Join(packet.Summary(), "\n"), "comparison: unsupported-contract") {
		t.Fatal("write effect was described as read-only")
	}
}

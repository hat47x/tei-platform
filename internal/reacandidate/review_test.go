package reacandidate

import (
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

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
	if !strings.Contains(joined, "binding: not-attempted") {
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

package teild0a

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadFixture(t *testing.T) Corpus {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "corpus_v0.json"))
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := DecodeCorpus(raw)
	if err != nil {
		t.Fatal(err)
	}
	return corpus
}

func TestSyntheticFixtureAndDerivedViews(t *testing.T) {
	corpus := loadFixture(t)
	if len(corpus.Rows) != 4 {
		t.Fatalf("rows=%d want=4", len(corpus.Rows))
	}

	ranking, err := BuildReuseRankingView(corpus)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranking) != 1 {
		t.Fatalf("ranking cases=%d want=1: %+v", len(ranking), ranking)
	}
	if ranking[0].CandidateSetID != "set-current" ||
		ranking[0].ContextID != "field-unit-price" ||
		ranking[0].PositiveCandidate.CanonicalID != "std.sales.UnitPrice" ||
		ranking[0].PositiveCandidate.Revision != "sha256:"+strings.Repeat("2", 64) {
		t.Fatalf("unexpected ranking case: %+v", ranking[0])
	}
	if len(ranking[0].NegativeCandidates) != 1 ||
		ranking[0].NegativeCandidates[0].CanonicalID != "std.sales.Amount" ||
		ranking[0].NegativeCandidates[0].Revision != "sha256:"+strings.Repeat("3", 64) {
		t.Fatalf("unknown/unresolved must not be forced negative: %+v", ranking[0])
	}

	incompatibility, err := BuildIncompatibilityView(corpus)
	if err != nil {
		t.Fatal(err)
	}
	if len(incompatibility) != 3 {
		t.Fatalf("incompatibility examples=%d want=3", len(incompatibility))
	}
	incompatible := 0
	for _, item := range incompatibility {
		if item.Incompatible {
			incompatible++
		}
	}
	if incompatible != 1 {
		t.Fatalf("incompatible examples=%d want=1: %+v", incompatible, incompatibility)
	}

	escalation, err := BuildEscalationView(corpus)
	if err != nil {
		t.Fatal(err)
	}
	if len(escalation) != 2 {
		t.Fatalf("escalation cases=%d want=2: %+v", len(escalation), escalation)
	}
	got := map[string]bool{}
	for _, item := range escalation {
		got[item.ContextID] = item.Escalate
	}
	if got["field-unit-price"] || !got["field-unit-price-v2"] {
		t.Fatalf("unexpected escalation view: %+v", got)
	}
}

func TestEscalationIsDecisionContextLevel(t *testing.T) {
	corpus := loadFixture(t)
	currentOnly := Corpus{
		Schema: corpus.Schema,
		Rows:   append([]Row(nil), corpus.Rows[:3]...),
	}
	escalation, err := BuildEscalationView(currentOnly)
	if err != nil {
		t.Fatal(err)
	}
	if len(escalation) != 1 || escalation[0].Escalate {
		t.Fatalf("accepted current context must not escalate because an explicit unknown candidate exists: %+v", escalation)
	}

	unknownOnly := Corpus{
		Schema: corpus.Schema,
		Rows:   []Row{corpus.Rows[2]},
	}
	escalation, err = BuildEscalationView(unknownOnly)
	if err != nil {
		t.Fatal(err)
	}
	if len(escalation) != 1 || !escalation[0].Escalate {
		t.Fatalf("unresolved context without a current accepted mapping must escalate: %+v", escalation)
	}
}

func TestReuseRankingSeparatesDecisionContextsSharingCandidateSet(t *testing.T) {
	corpus := loadFixture(t)

	accepted := corpus.Rows[1]
	accepted.RowID = "row-context-two-accepted-amount"
	accepted.SourceID = "source.sales.order.amount"
	accepted.SourceRevision = "sha256:" + strings.Repeat("6", 64)
	accepted.ApplicationID = "app-sales-context-two"
	accepted.ContextID = "field-amount"
	accepted.ProposalID = "proposal-003"
	accepted.TemporalGroupID = "temporal-sales-amount-early-003"
	accepted.Split.ApplicationPattern = "pattern-sales-context-two"
	accepted.Split.SourceFamily = "source-amount-v1"
	accepted.Split.RevisionLineage = "lineage-amount-v1"
	accepted.Split.ProposalEpisode = "episode-003"
	accepted.Outcome.ProposalDecision = DecisionAccepted
	accepted.Outcome.DecisionProvenance = "shared-acceptance:decision-003"
	accepted.Outcome.Staleness = StalenessCurrent
	accepted.Outcome.PostAcceptConformance = ConformancePass
	accepted.Outcome.FinalCanonicalMapping = &CanonicalRef{
		CanonicalID: accepted.Candidate.CanonicalID,
		Revision:    accepted.Candidate.CanonicalRevision,
	}
	accepted.Features.Candidate.Compatibility = CompatibilityCompatible
	accepted.Features.Candidate.IncompatibilityReasons = nil

	rejected := corpus.Rows[0]
	rejected.RowID = "row-context-two-rejected-unit-price"
	rejected.SourceID = accepted.SourceID
	rejected.SourceRevision = accepted.SourceRevision
	rejected.ApplicationID = accepted.ApplicationID
	rejected.ContextID = accepted.ContextID
	rejected.ProposalID = accepted.ProposalID
	rejected.TemporalGroupID = accepted.TemporalGroupID
	rejected.Split = accepted.Split
	rejected.Outcome.ProposalDecision = DecisionRejected
	rejected.Outcome.DecisionProvenance = accepted.Outcome.DecisionProvenance
	rejected.Outcome.Staleness = StalenessCurrent
	rejected.Outcome.PostAcceptConformance = ConformanceNA
	rejected.Outcome.FinalCanonicalMapping = nil
	rejected.Features.Candidate.Compatibility = CompatibilityIncompatible
	rejected.Features.Candidate.IncompatibilityReasons = []string{"semantic-role-mismatch"}

	corpus.Rows = append(corpus.Rows, accepted, rejected)
	ranking, err := BuildReuseRankingView(corpus)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranking) != 2 {
		t.Fatalf("ranking cases=%d want=2: %+v", len(ranking), ranking)
	}
	got := map[string]ReuseRankingCase{}
	for _, item := range ranking {
		got[item.ContextID] = item
	}
	second, ok := got["field-amount"]
	if !ok {
		t.Fatalf("missing second decision context: %+v", ranking)
	}
	if second.PositiveCandidate.CanonicalID != "std.sales.Amount" ||
		second.PositiveCandidate.Revision != "sha256:"+strings.Repeat("3", 64) {
		t.Fatalf("unexpected second-context positive: %+v", second)
	}
	if len(second.NegativeCandidates) != 1 ||
		second.NegativeCandidates[0].CanonicalID != "std.sales.UnitPrice" ||
		second.NegativeCandidates[0].Revision != "sha256:"+strings.Repeat("2", 64) {
		t.Fatalf("unexpected second-context negatives: %+v", second)
	}
}

func TestReuseRankingRequiresExactCandidateRevision(t *testing.T) {
	corpus := loadFixture(t)
	corpus.Rows[0].Outcome.FinalCanonicalMapping.Revision = "sha256:" + strings.Repeat("9", 64)
	ranking, err := BuildReuseRankingView(corpus)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranking) != 0 {
		t.Fatalf("mapping to a revision outside the candidate set must not become a ranking positive: %+v", ranking)
	}
}

func TestResolverBackedValidationRejectsUnresolvableRecordedRevision(t *testing.T) {
	corpus := loadFixture(t)
	known := map[string]bool{
		canonicalRefKey(CanonicalRef{
			CanonicalID: "std.sales.UnitPrice",
			Revision:    "sha256:" + strings.Repeat("2", 64),
		}): true,
		canonicalRefKey(CanonicalRef{
			CanonicalID: "std.sales.Amount",
			Revision:    "sha256:" + strings.Repeat("3", 64),
		}): true,
	}
	resolve := func(ref CanonicalRef) bool {
		return known[canonicalRefKey(ref)]
	}
	if err := ValidateCorpusWithCanonicalResolver(corpus, resolve); err != nil {
		t.Fatalf("resolver-backed validation rejected known fixture identities: %v", err)
	}

	delete(known, canonicalRefKey(CanonicalRef{
		CanonicalID: "std.sales.Amount",
		Revision:    "sha256:" + strings.Repeat("3", 64),
	}))
	if err := ValidateCorpusWithCanonicalResolver(corpus, resolve); err == nil ||
		!strings.Contains(err.Error(), "cannot be resolved at recorded revision") {
		t.Fatalf("expected unresolved candidate revision failure, got %v", err)
	}
}

func TestResolverBackedValidationChecksFinalMappingOutsideOriginalCandidateSet(t *testing.T) {
	corpus := loadFixture(t)
	manual := CanonicalRef{
		CanonicalID: "std.sales.ManualCorrection",
		Revision:    "sha256:" + strings.Repeat("7", 64),
	}
	corpus.Rows[0].Outcome.FinalCanonicalMapping = &manual

	resolve := func(ref CanonicalRef) bool {
		return ref.CanonicalID == "std.sales.UnitPrice" ||
			ref.CanonicalID == "std.sales.Amount"
	}
	if err := ValidateCorpusWithCanonicalResolver(corpus, resolve); err == nil ||
		!strings.Contains(err.Error(), "final_canonical_mapping") {
		t.Fatalf("expected unresolved corrected final mapping failure, got %v", err)
	}
}

func TestResolverBackedValidationRequiresResolver(t *testing.T) {
	corpus := loadFixture(t)
	if err := ValidateCorpusWithCanonicalResolver(corpus, nil); err == nil ||
		!strings.Contains(err.Error(), "resolver is required") {
		t.Fatalf("expected missing resolver failure, got %v", err)
	}
}

func TestCandidateSetFingerprintIsOrderIndependent(t *testing.T) {
	corpus := loadFixture(t)
	row := corpus.Rows[0]
	first, err := CandidateSetFingerprint(row.BaseModelRevision, row.CandidateSet)
	if err != nil {
		t.Fatal(err)
	}
	reversed := append([]CandidateRef(nil), row.CandidateSet...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	second, err := CandidateSetFingerprint(row.BaseModelRevision, reversed)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first != row.CandidateSetSHA256 {
		t.Fatalf("fingerprint drift first=%s second=%s row=%s", first, second, row.CandidateSetSHA256)
	}
}

func TestCandidateSetHashMismatchFailsClosed(t *testing.T) {
	corpus := loadFixture(t)
	corpus.Rows[0].CandidateSetSHA256 = "sha256:" + strings.Repeat("0", 64)
	if err := ValidateCorpus(corpus); err == nil ||
		!strings.Contains(err.Error(), "candidate_set_sha256 mismatch") {
		t.Fatalf("expected candidate-set mismatch, got %v", err)
	}
}

func TestCandidateSetRequiresExplicitUnknown(t *testing.T) {
	corpus := loadFixture(t)
	filtered := []CandidateRef{}
	for _, candidate := range corpus.Rows[0].CandidateSet {
		if candidate.Kind != CandidateKindUnknown {
			filtered = append(filtered, candidate)
		}
	}
	corpus.Rows[0].CandidateSet = filtered
	if err := ValidateCorpus(corpus); err == nil ||
		!strings.Contains(err.Error(), "exactly one explicit unknown") {
		t.Fatalf("expected explicit-unknown failure, got %v", err)
	}
}

func TestPostAcceptEvidenceCannotExistBeforeAccept(t *testing.T) {
	corpus := loadFixture(t)
	corpus.Rows[1].Outcome.PostAcceptConformance = ConformancePass
	if err := ValidateCorpus(corpus); err == nil ||
		!strings.Contains(err.Error(), "cannot exist before accepted") {
		t.Fatalf("expected post-Accept ordering failure, got %v", err)
	}
}

func TestFinalMappingRequiresAcceptedDecision(t *testing.T) {
	corpus := loadFixture(t)
	corpus.Rows[1].Outcome.FinalCanonicalMapping = &CanonicalRef{
		CanonicalID: "std.sales.Amount",
		Revision:    "sha256:" + strings.Repeat("3", 64),
	}
	if err := ValidateCorpus(corpus); err == nil ||
		!strings.Contains(err.Error(), "final canonical mapping requires accepted") {
		t.Fatalf("expected mapping authority failure, got %v", err)
	}
}

func TestProposalEpisodeCannotCrossSplit(t *testing.T) {
	corpus := loadFixture(t)
	corpus.Rows[3].Split.ProposalEpisode = "episode-001"
	if err := ValidateCorpus(corpus); err == nil ||
		!strings.Contains(err.Error(), "proposal_episode") {
		t.Fatalf("expected split leakage failure, got %v", err)
	}
}


func TestRowIdentityRequiresRecordedSourceTemporalGroupAndSchema(t *testing.T) {
	cases := []struct {
		name string
		mutate func(*Row)
		want string
	}{
		{"schema-version", func(row *Row) { row.SchemaVersion = "" }, "schema_version"},
		{"source-id", func(row *Row) { row.SourceID = "" }, "source_id"},
		{"source-revision", func(row *Row) { row.SourceRevision = "not-a-revision" }, "source_revision"},
		{"temporal-group", func(row *Row) { row.TemporalGroupID = "" }, "temporal_group_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			corpus := loadFixture(t)
			tc.mutate(&corpus.Rows[0])
			if err := ValidateCorpus(corpus); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %s identity failure, got %v", tc.want, err)
			}
		})
	}
}

func TestSplitContractUsesPreregisteredSlices(t *testing.T) {
	corpus := loadFixture(t)
	if corpus.Rows[3].Split.Name != SplitTestC {
		t.Fatalf("stale fixture split=%q want=%q", corpus.Rows[3].Split.Name, SplitTestC)
	}

	corpus.Rows[3].Split.Name = "test"
	if err := ValidateCorpus(corpus); err == nil ||
		!strings.Contains(err.Error(), "invalid split") {
		t.Fatalf("expected generic test split to be rejected, got %v", err)
	}
}

func TestTemporalGroupCannotCrossSplit(t *testing.T) {
	corpus := loadFixture(t)
	corpus.Rows[3].TemporalGroupID = corpus.Rows[0].TemporalGroupID
	if err := ValidateCorpus(corpus); err == nil ||
		!strings.Contains(err.Error(), "temporal_group") {
		t.Fatalf("expected temporal-group split leakage failure, got %v", err)
	}
}

func TestUnknownFieldIsRejected(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "corpus_v0.json"))
	if err != nil {
		t.Fatal(err)
	}
	mutated := bytes.Replace(raw, []byte("{"), []byte("{\"unexpected\":true,"), 1)
	if _, err := DecodeCorpus(mutated); err == nil {
		t.Fatal("unknown top-level field was accepted")
	}
}

func TestAcceptedDecisionRequiresProvenance(t *testing.T) {
	corpus := loadFixture(t)
	corpus.Rows[0].Outcome.DecisionProvenance = ""
	if err := ValidateCorpus(corpus); err == nil ||
		!strings.Contains(err.Error(), "decision_provenance") {
		t.Fatalf("expected decision provenance failure, got %v", err)
	}
}

func TestSourceFeatureIdentityProjectionFailsClosed(t *testing.T) {
	corpus := loadFixture(t)

	corpus.Rows[0].Features.Source.ReferencedSourceIDs = nil
	if err := ValidateCorpus(corpus); err == nil ||
		!strings.Contains(err.Error(), "referenced_source_ids") {
		t.Fatalf("expected missing referenced source identity failure, got %v", err)
	}

	corpus = loadFixture(t)
	corpus.Rows[0].Features.Source.EvidenceIDs = []string{"evidence:source-shape"}
	if err := ValidateCorpus(corpus); err == nil ||
		!strings.Contains(err.Error(), "does not match evidence_count") {
		t.Fatalf("expected evidence identity/count mismatch, got %v", err)
	}

	corpus = loadFixture(t)
	corpus.Rows[0].Features.Source.EvidenceIDs = []string{"evidence:dup", "evidence:dup"}
	if err := ValidateCorpus(corpus); err == nil ||
		!strings.Contains(err.Error(), "duplicate identity") {
		t.Fatalf("expected duplicate evidence identity failure, got %v", err)
	}
}

func TestPostAcceptConformanceRequiresReloadEvidenceLink(t *testing.T) {
	corpus := loadFixture(t)
	corpus.Rows[0].Outcome.PostAcceptEvidenceRef = ""
	if err := ValidateCorpus(corpus); err == nil ||
		!strings.Contains(err.Error(), "post_accept_evidence_ref") {
		t.Fatalf("expected pass-without-evidence-link failure, got %v", err)
	}

	corpus = loadFixture(t)
	corpus.Rows[3].Outcome.PostAcceptEvidenceRef = "post-accept:invented"
	if err := ValidateCorpus(corpus); err == nil ||
		!strings.Contains(err.Error(), "must not claim post_accept_evidence_ref") {
		t.Fatalf("expected unavailable-with-evidence-link failure, got %v", err)
	}

	corpus = loadFixture(t)
	corpus.Rows[1].Outcome.PostAcceptEvidenceRef = "post-accept:invalid-before-accept"
	if err := ValidateCorpus(corpus); err == nil ||
		!strings.Contains(err.Error(), "must not claim post_accept_evidence_ref") {
		t.Fatalf("expected rejected-row evidence-link failure, got %v", err)
	}
}

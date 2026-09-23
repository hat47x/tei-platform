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
		ranking[0].PositiveCanonicalID != "std.sales.UnitPrice" {
		t.Fatalf("unexpected ranking case: %+v", ranking[0])
	}
	if len(ranking[0].NegativeCanonicalIDs) != 1 ||
		ranking[0].NegativeCanonicalIDs[0] != "std.sales.Amount" {
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
	got := map[string]bool{}
	for _, item := range escalation {
		got[item.RowID] = item.Escalate
	}
	if got["row-accepted-unit-price"] ||
		got["row-rejected-amount"] ||
		!got["row-unknown"] ||
		!got["row-stale-accepted"] {
		t.Fatalf("unexpected escalation view: %+v", got)
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

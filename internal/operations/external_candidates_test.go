package operations_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tei/internal/externalanalysis"
	"tei/internal/operations"
)

// The fixtures are the real captured REA output under
// internal/externalanalysis/testdata, with the run record in the manifest next
// to them. The assertions below pin the projection field by field, so a
// producer change fails here instead of quietly redefining what a candidate
// view holds.
const (
	candidateEvidenceID = "ev_a88c70bcd066b2f04fdf97cb349369f0779b9c68cc4ac6c0e5f38f3539477a93"
	candidateNodeID     = "jsrg_node_237478f2a27c9614a522cbfc7ab60fa96943a458d164945d25af634f1240b652"
	candidateRelationID = "jsrg_relation_0f6757a241c56fd84ecaec91f538d8297d222eb5625e4cdbcb2de7603f330718"
	candidateDigest     = "sha256:7ea3780d8dc23d00c380145af62d63bf194e39decbaefab3d82c5a3b4134bba9"
)

var candidateReadAt = time.Date(2026, 11, 11, 5, 5, 0, 0, time.UTC)

func candidateFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "externalanalysis", "testdata", "rea-6.8.0", "analysis.json")
	if _, err := os.ReadFile(path); err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return path
}

// TestExternalCandidatesProjectsOneObservationAndOneRelation is the T1-A
// vertical slice: one external Observation and one directed Relation from the
// same attempt, received losslessly.
func TestExternalCandidatesProjectsOneObservationAndOneRelation(t *testing.T) {
	view, err := operations.ExternalCandidates(operations.ExternalCandidateRequest{
		EvidencePath:  candidateFixture(t),
		ToolVersion:   "6.8.0",
		ObservationID: candidateNodeID,
		RelationID:    candidateRelationID,
		Tool:          externalanalysis.Tool{Name: "rea", Present: true, ResolvedPath: "test"},
	}, candidateReadAt)
	if err != nil {
		t.Fatalf("project: %v", err)
	}

	if view.SchemaVersion != "tei.operations.external-candidates/v0" {
		t.Fatalf("schemaVersion=%s", view.SchemaVersion)
	}
	if view.Effect != "read-only" {
		t.Fatalf("effect=%s", view.Effect)
	}
	// A1: the attempt names the Evidence, the bytes that were read, and when.
	if view.Attempt.EvidenceID != candidateEvidenceID {
		t.Fatalf("attempt evidence id=%s", view.Attempt.EvidenceID)
	}
	if view.Attempt.ReceivedDigest != candidateDigest {
		t.Fatalf("attempt digest=%s want %s", view.Attempt.ReceivedDigest, candidateDigest)
	}
	if view.Attempt.ReadAt != "2026-11-11T05:05:00Z" {
		t.Fatalf("attempt readAt=%s", view.Attempt.ReadAt)
	}
	if view.Attempt.Provider.ID != "rea-javascript-application" {
		t.Fatalf("attempt provider=%s", view.Attempt.Provider.ID)
	}
	if view.Attempt.Envelope == nil || view.Attempt.Envelope.Command != "analyze-javascript-application" {
		t.Fatalf("attempt envelope=%+v: the producer's own execution record travels with the attempt", view.Attempt.Envelope)
	}

	// A2: one observation, with every field the coordination needs.
	if view.Observation == nil {
		t.Fatal("the selected observation is missing")
	}
	if view.Observation.NodeID != candidateNodeID {
		t.Fatalf("node id=%s", view.Observation.NodeID)
	}
	if view.Observation.Kind != "call-site" {
		t.Fatalf("kind=%s", view.Observation.Kind)
	}
	if view.Observation.ArtifactSHA256 != "e8dc069bd1398a5b3d90db155e379df84a707b163d593a533970e5d67dadaaea" {
		t.Fatalf("artifact digest=%s", view.Observation.ArtifactSHA256)
	}
	if view.Observation.Location != "source-range:index.js" {
		t.Fatalf("location=%s", view.Observation.Location)
	}
	if view.Observation.Span == nil || view.Observation.Span.StartLine != 2 {
		t.Fatalf("span=%+v", view.Observation.Span)
	}
	if view.Observation.State == "" || view.Observation.Confidence == "" {
		t.Fatalf("the evidence context qualifies the claim: state=%q confidence=%q", view.Observation.State, view.Observation.Confidence)
	}

	// A3: one directed edge, with the producer's own relation identity and
	// direction.
	if view.Relation == nil {
		t.Fatal("the selected relation is missing")
	}
	if view.Relation.RelationID != candidateRelationID || view.Relation.Derived {
		t.Fatalf("relation id=%s derived=%t: the producer states its own identity", view.Relation.RelationID, view.Relation.Derived)
	}
	if view.Relation.From == "" || view.Relation.To == "" || view.Relation.From == view.Relation.To {
		t.Fatalf("relation endpoints=%+v", view.Relation)
	}
	if view.Relation.Kind != "defines" || view.Relation.Resolution != "resolved" {
		t.Fatalf("relation=%+v", view.Relation)
	}

	// A4: unknown and partial survive the projection.
	if view.Coverage == nil || view.Coverage.Status != "unknown" {
		t.Fatalf("coverage=%+v: the producer said the graph is not fully covered", view.Coverage)
	}
	if len(view.Coverage.Families) == 0 {
		t.Fatal("coverage per family must survive")
	}
	if len(view.Unknowns) == 0 {
		t.Fatal("the producer's unknowns must survive the projection")
	}
	if len(view.Limitations) == 0 {
		t.Fatal("the producer's limitations must survive the projection")
	}

	// Two candidates are not the whole graph.
	if view.Unselected.Observations == 0 || view.Unselected.Relations == 0 {
		t.Fatalf("unselected=%+v: the Evidence carries more than the two projected candidates", view.Unselected)
	}

	// A5: the projection states what it does not grant.
	for _, want := range []string{
		"external-candidate-is-not-accepted-meaning",
		"external-identity-is-not-a-tei-semantic-subject",
		"external-observation-is-not-a-database-observation",
	} {
		if !containsString(view.AuthorityLimits, want) {
			t.Fatalf("authorityLimits=%v want %s", view.AuthorityLimits, want)
		}
	}
}

// TestExternalCandidatesRefusesWhatItCannotProject: an undeclared version, an
// unreadable path, bytes that are not an Evidence, a delivery the producer
// failed, and a selection the Evidence does not carry are five different
// answers.
func TestExternalCandidatesRefusesWhatItCannotProject(t *testing.T) {
	dir := t.TempDir()
	base := operations.ExternalCandidateRequest{
		EvidencePath:  candidateFixture(t),
		ToolVersion:   "6.8.0",
		ObservationID: candidateNodeID,
		RelationID:    candidateRelationID,
		Tool:          externalanalysis.Tool{Name: "rea", Present: true},
	}
	if _, err := operations.ExternalCandidates(base, candidateReadAt); err != nil {
		t.Fatalf("the fixture must project: %v", err)
	}

	for name, mutate := range map[string]func(*operations.ExternalCandidateRequest){
		"undeclared version":  func(r *operations.ExternalCandidateRequest) { r.ToolVersion = "" },
		"missing path":        func(r *operations.ExternalCandidateRequest) { r.EvidencePath = filepath.Join(dir, "absent.json") },
		"unknown observation": func(r *operations.ExternalCandidateRequest) { r.ObservationID = "jsrg_node_absent" },
		"unknown relation":    func(r *operations.ExternalCandidateRequest) { r.RelationID = "jsrg_relation_absent" },
	} {
		t.Run(name, func(t *testing.T) {
			request := base
			mutate(&request)
			_, err := operations.ExternalCandidates(request, candidateReadAt)
			if err == nil {
				t.Fatal("the projection must refuse")
			}
			var candidate *operations.ExternalCandidateError
			if !errors.As(err, &candidate) {
				t.Fatalf("the failure must be a typed candidate error: %v", err)
			}
			if candidate.Code == "" {
				t.Fatal("the failure must name a code")
			}
		})
	}

	// Bytes that are not the declared shape, and bytes the producer failed to
	// produce, are refused with their own codes.
	notJSON := filepath.Join(dir, "not.json")
	if err := os.WriteFile(notJSON, []byte("this is not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := base
	request.EvidencePath = notJSON
	_, err := operations.ExternalCandidates(request, candidateReadAt)
	if err == nil || !strings.Contains(err.Error(), "JSON") {
		t.Fatalf("bytes that are not an Evidence must be refused as unusable: %v", err)
	}

	refusal := filepath.Join("..", "externalanalysis", "testdata", "rea-6.8.0", "error-source-kind.jsonl")
	request = base
	request.EvidencePath = refusal
	_, err = operations.ExternalCandidates(request, candidateReadAt)
	if err == nil {
		t.Fatal("a refusal delivery must be refused as a failure, not projected")
	}
	var candidate *operations.ExternalCandidateError
	if !errors.As(err, &candidate) || candidate.Code != operations.ExternalCandidateFailed {
		t.Fatalf("code=%v want %s", candidate, operations.ExternalCandidateFailed)
	}
}

// TestExternalCandidatesRecordsTheToolAbsence: an environment without the
// external tool still projects the saved bytes, and the attempt says the bytes
// could not have been produced here.
func TestExternalCandidatesRecordsTheToolAbsence(t *testing.T) {
	view, err := operations.ExternalCandidates(operations.ExternalCandidateRequest{
		EvidencePath:  candidateFixture(t),
		ToolVersion:   "6.8.0",
		ObservationID: candidateNodeID,
		RelationID:    candidateRelationID,
		Tool:          externalanalysis.Tool{Name: "rea", Present: false, AbsentReason: `exec: "rea": executable file not found in $PATH`},
	}, candidateReadAt)
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	if view.Attempt.Tool.Present {
		t.Fatal("the attempt must record the absence")
	}
	if view.Attempt.Tool.AbsentReason == "" {
		t.Fatal("the absence needs a reason a reader can act on")
	}
	if view.Attempt.ReceivedDigest != candidateDigest {
		t.Fatalf("digest=%s: the bytes are still the bytes", view.Attempt.ReceivedDigest)
	}
	if view.Observation == nil || view.Relation == nil {
		t.Fatal("the candidates are still projected from the saved bytes")
	}
}

// TestExternalCandidateSelectionIsNotTheWholeGraph pins the boundary the
// projection states: two selected candidates, and a count of what was left
// behind, so a caller cannot read a selection as the graph.
func TestExternalCandidateSelectionIsNotTheWholeGraph(t *testing.T) {
	view, err := operations.ExternalCandidates(operations.ExternalCandidateRequest{
		EvidencePath:  candidateFixture(t),
		ToolVersion:   "6.8.0",
		ObservationID: candidateNodeID,
		RelationID:    candidateRelationID,
		Tool:          externalanalysis.Tool{Name: "rea", Present: true},
	}, candidateReadAt)
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	if view.Unselected.Observations != 6 || view.Unselected.Relations != 1 {
		t.Fatalf("unselected=%+v want observations 6 and relations 1", view.Unselected)
	}
	if view.Unselected.Unknowns != len(view.Unknowns) {
		t.Fatalf("unselected unknowns=%d want %d", view.Unselected.Unknowns, len(view.Unknowns))
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

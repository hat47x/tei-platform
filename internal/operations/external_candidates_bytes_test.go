package operations_test

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"tei/internal/externalanalysis"
	"tei/internal/operations"
)

// The local saved-file reader and the Host-owned byte entry must use exactly
// the same R2 receipt/parser/selection, without re-reading an internal path.
func TestExternalCandidateFromBytesMatchesLocalR2Projection(t *testing.T) {
	path := candidateFixture(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	request := operations.ExternalCandidateRequest{
		ToolVersion:   "6.8.0",
		ObservationID: candidateNodeID,
		RelationID:    candidateRelationID,
		Tool:          externalanalysis.Tool{Name: "rea", Present: true, ResolvedPath: "test"},
	}
	fromBytes, err := operations.ExternalCandidatesFromBytes(request, raw, candidateReadAt)
	if err != nil {
		t.Fatalf("byte projection: %v", err)
	}
	local := request
	local.EvidencePath = path
	fromFile, err := operations.ExternalCandidates(local, candidateReadAt)
	if err != nil {
		t.Fatalf("local projection: %v", err)
	}
	if !reflect.DeepEqual(fromBytes, fromFile) {
		t.Fatalf("Host-owned bytes and local file disagree\nbytes=%+v\nfile=%+v", fromBytes, fromFile)
	}
	if fromBytes.Attempt.EvidenceID != candidateEvidenceID || fromBytes.Attempt.ReceivedDigest != candidateDigest ||
		fromBytes.Observation == nil || fromBytes.Observation.NodeID != candidateNodeID ||
		fromBytes.Relation == nil || fromBytes.Relation.RelationID != candidateRelationID ||
		fromBytes.Coverage == nil || len(fromBytes.Unknowns) == 0 ||
		fromBytes.Unselected.Relations == 0 || len(fromBytes.AuthorityLimits) == 0 {
		t.Fatalf("R2 evidence qualification lost: %+v", fromBytes)
	}

	// A previously inspected attempt may pin all three distinct identifiers.
	pinned := request
	pinned.ExpectedEvidenceID = fromBytes.Attempt.EvidenceID
	pinned.ExpectedReceivedDigest = fromBytes.Attempt.ReceivedDigest
	pinned.ExpectedSourceRevision = fromBytes.Attempt.SourceRevision
	if got, err := operations.ExternalCandidatesFromBytes(pinned, raw, candidateReadAt); err != nil ||
		!reflect.DeepEqual(fromBytes, got) {
		t.Fatalf("matching Host-owned pins must not change the projection: err=%v", err)
	}
}

func TestExternalCandidateFromBytesRefusesPathInjectionAndStaleEvidence(t *testing.T) {
	raw, err := os.ReadFile(candidateFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	request := operations.ExternalCandidateRequest{
		ToolVersion:   "6.8.0",
		ObservationID: candidateNodeID,
		RelationID:    candidateRelationID,
		Tool:          externalanalysis.Tool{Name: "rea", Present: false},
	}
	assertCode := func(r operations.ExternalCandidateRequest, data []byte, code string) {
		t.Helper()
		_, err := operations.ExternalCandidatesFromBytes(r, data, candidateReadAt)
		var candidateErr *operations.ExternalCandidateError
		if !errors.As(err, &candidateErr) || candidateErr.Code != code {
			t.Fatalf("error=%v, want typed %s", err, code)
		}
	}
	pathInjection := request
	pathInjection.EvidencePath = "/private/should-never-be-read"
	assertCode(pathInjection, raw, operations.ExternalCandidateInvalidRequest)

	noVersion := request
	noVersion.ToolVersion = ""
	assertCode(noVersion, raw, operations.ExternalCandidateInvalidRequest)

	wrongPin := request
	wrongPin.ExpectedReceivedDigest = "sha256:not-the-received-evidence"
	assertCode(wrongPin, raw, operations.ExternalCandidateStale)

	wrongEvidence := request
	wrongEvidence.ExpectedEvidenceID = "ev_unknown"
	assertCode(wrongEvidence, raw, operations.ExternalCandidateStale)

	// A malformed Host-delivered payload fails as unusable Evidence; it cannot
	// become a filesystem ReadFailed or silently fall back to a filename.
	assertCode(request, []byte("not an REA Evidence"), operations.ExternalCandidateUnusable)
}

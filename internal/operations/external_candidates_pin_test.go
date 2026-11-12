package operations_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/externalanalysis"
	"tei/internal/operations"
)

// The pin identifies one previously inspected producer delivery, not a
// current Source read, an accepted semantic model, or a disclosure grant.
func TestExternalCandidatesRefusesStalePinnedDelivery(t *testing.T) {
	base := operations.ExternalCandidateRequest{
		EvidencePath:  candidateFixture(t),
		ToolVersion:   "6.8.0",
		ObservationID: candidateNodeID,
		RelationID:    candidateRelationID,
		Tool:          externalanalysis.Tool{Name: "rea", Present: true},
	}
	received, err := operations.ExternalCandidates(base, candidateReadAt)
	if err != nil {
		t.Fatalf("read baseline receipt: %v", err)
	}
	if received.Attempt.EvidenceID == "" ||
		received.Attempt.ReceivedDigest == "" ||
		received.Attempt.SourceRevision == "" {
		t.Fatalf("fixture needs all three pin facts: %+v", received.Attempt)
	}
	pinned := base
	pinned.ExpectedEvidenceID = received.Attempt.EvidenceID
	pinned.ExpectedReceivedDigest = received.Attempt.ReceivedDigest
	pinned.ExpectedSourceRevision = received.Attempt.SourceRevision
	if view, err := operations.ExternalCandidates(pinned, candidateReadAt); err != nil || view.Observation == nil || view.Relation == nil {
		t.Fatalf("unchanged pinned delivery must remain selectable: %+v err=%v", view, err)
	}
	cases := []struct {
		name string
		edit func(*operations.ExternalCandidateRequest)
	}{
		{"different producer Evidence ID", func(r *operations.ExternalCandidateRequest) {
			r.ExpectedEvidenceID = "ev_previous"
		}},
		{"different raw byte digest", func(r *operations.ExternalCandidateRequest) {
			r.ExpectedReceivedDigest = "sha256:previous"
		}},
		{"different producer-declared Source revision", func(r *operations.ExternalCandidateRequest) {
			r.ExpectedSourceRevision = "sha256:previous"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := pinned
			tc.edit(&request)
			view, err := operations.ExternalCandidates(request, candidateReadAt)
			assertStaleCandidatePin(t, view, err)
		})
	}

	// The same producer's semantic graph can be serialized into different
	// bytes. Re-reading the current EvidencePath must reject the old raw-byte
	// pin even if the producer's target revision still looks unchanged.
	raw := []byte(candidateText(t) + "\n")
	changedPath := filepath.Join(t.TempDir(), "same-semantic-evidence-new-bytes.json")
	if err := os.WriteFile(changedPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	changed := pinned
	changed.EvidencePath = changedPath
	view, err := operations.ExternalCandidates(changed, candidateReadAt)
	assertStaleCandidatePin(t, view, err)
	if view.Attempt.EvidenceID != received.Attempt.EvidenceID ||
		view.Attempt.SourceRevision != received.Attempt.SourceRevision ||
		view.Attempt.ReceivedDigest == received.Attempt.ReceivedDigest {
		t.Fatalf("mutation must vary received bytes only: old=%+v changed=%+v", received.Attempt, view.Attempt)
	}
}

func assertStaleCandidatePin(t *testing.T, view operations.ExternalCandidateView, err error) {
	t.Helper()
	var candidate *operations.ExternalCandidateError
	if !errors.As(err, &candidate) || candidate.Code != operations.ExternalCandidateStale {
		t.Fatalf("pin mismatch must be typed stale, got view=%+v err=%v", view, err)
	}
	if view.Observation != nil || view.Relation != nil {
		t.Fatalf("stale pin returned a candidate: %+v", view)
	}
	if view.Attempt.ReceivedDigest == "" || view.Attempt.EvidenceID == "" {
		t.Fatalf("refused pin must retain the measured received attempt: %+v", view.Attempt)
	}
	if view.Unselected.Observations == 0 || view.Unselected.Relations == 0 {
		t.Fatalf("refused pin must not imply a selected subset: %+v", view.Unselected)
	}
}

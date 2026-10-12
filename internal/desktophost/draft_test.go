package desktophost

import (
	"encoding/json"
	"errors"
	"testing"
)

func baseline() SourceObservation {
	return SourceObservation{WorkspaceRoot: "/tmp/workspace", Artifacts: []ObservedArtifact{{Path: "semantic/a.tei", Fingerprint: "sha256:one"}}}
}

func TestCompareObservations(t *testing.T) {
	old := baseline()
	if err := CompareObservations(old, old); err != nil {
		t.Fatal(err)
	}
	changed := baseline()
	changed.Artifacts[0].Fingerprint = "sha256:two"
	if !errors.Is(CompareObservations(old, changed), ErrSourceChanged) {
		t.Fatal("changed source was not detected")
	}
	changed = baseline()
	changed.Artifacts = append(changed.Artifacts, ObservedArtifact{Path: "semantic/new.tei", Fingerprint: "sha256:three"})
	if !errors.Is(CompareObservations(old, changed), ErrSourceChanged) {
		t.Fatal("new source was not detected")
	}
	changed = baseline()
	changed.WorkspaceRoot = "/tmp/other"
	if !errors.Is(CompareObservations(old, changed), ErrSourceChanged) {
		t.Fatal("another Workspace was not detected")
	}
}

func TestDraftRecoveryNeverAccepts(t *testing.T) {
	marker := DraftMarker{Version: MarkerVersion, CandidateRef: "candidate-7", BaseModelRevision: "rev-42", Baseline: baseline()}
	encoded, err := EncodeDraftMarker(marker)
	if err != nil {
		t.Fatal(err)
	}
	var stored map[string]any
	if err := json.Unmarshal(encoded, &stored); err != nil {
		t.Fatal(err)
	}
	if _, ok := stored["candidateBytes"]; ok {
		t.Fatal("candidate data must not be serialized")
	}
	readCount := 0
	restore, err := RecoverDraftMarker(encoded, "/tmp/workspace", func(string) (SourceObservation, error) {
		readCount++
		return baseline(), nil
	})
	if err != nil || readCount != 1 || restore.Status != RecoveryNeedsReview {
		t.Fatalf("recovery should only suggest review: %+v %v", restore, err)
	}
	changed, err := RecoverDraftMarker(encoded, "/tmp/workspace", func(string) (SourceObservation, error) {
		newObs := baseline()
		newObs.Artifacts[0].Fingerprint = "sha256:changed"
		return newObs, nil
	})
	if err != nil || changed.Status != RecoverySourceChanged {
		t.Fatalf("stale draft must need reconciliation: %+v %v", changed, err)
	}
	unavailable, err := RecoverDraftMarker(encoded, "/tmp/workspace", func(string) (SourceObservation, error) {
		return SourceObservation{}, errors.New("offline")
	})
	if err != nil || unavailable.Status != RecoverySourceUnavailable {
		t.Fatalf("unavailable source must not be accepted: %+v %v", unavailable, err)
	}
}

func TestInvalidDraftMarker(t *testing.T) {
	m := DraftMarker{Version: MarkerVersion, CandidateRef: "candidate-7", BaseModelRevision: "rev-42", Baseline: baseline()}
	encoded, err := EncodeDraftMarker(m)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverDraftMarker(encoded, "/tmp/other", func(string) (SourceObservation, error) {
		t.Fatal("must not probe unauthorized Workspace")
		return baseline(), nil
	}); err == nil {
		t.Fatal("untrusted marker selected another Workspace")
	}
	if _, err := RecoverDraftMarker([]byte("{}"), "/tmp/workspace", func(string) (SourceObservation, error) { return baseline(), nil }); err == nil {
		t.Fatal("invalid marker accepted")
	}
	if _, err := RecoverDraftMarker([]byte("{} {}"), "/tmp/workspace", func(string) (SourceObservation, error) { return baseline(), nil }); err == nil {
		t.Fatal("invalid marker accepted")
	}
	if _, err := EncodeDraftMarker(DraftMarker{}); err == nil {
		t.Fatal("invalid marker serialized")
	}
}

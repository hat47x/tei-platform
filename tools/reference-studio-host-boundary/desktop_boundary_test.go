package hostboundary

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"tei/internal/desktophost"
)

// This exercises the existing #1529 Draft-marker API directly, without
// implementing a second Acceptance or resource authorization mechanism.
func TestDraftStoreRestartRejectsOtherWorkspaceAndNeverAutoAccepts(t *testing.T) {
	stateDir := t.TempDir()
	if runtime.GOOS != "windows" {
		if err := os.Chmod(stateDir, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	makeMarker := func(root string) desktophost.DraftMarker {
		return desktophost.DraftMarker{
			Version: desktophost.MarkerVersion, CandidateRef: "same-candidate-name",
			BaseModelRevision: "baseline-revision",
			Baseline: desktophost.SourceObservation{
				WorkspaceRoot: root,
				Artifacts: []desktophost.ObservedArtifact{
					{Path: "tei.workspace.yaml", Fingerprint: "sha256:baseline"},
				},
			},
		}
	}
	// Observations here are synthetic and do not prove that an actual
	// Workspace path was contained by a Desktop OS resource adapter.
	workspaceA := filepath.Join(t.TempDir(), "workspace-a")
	workspaceB := filepath.Join(t.TempDir(), "workspace-b")
	markerA, markerB := makeMarker(workspaceA), makeMarker(workspaceB)
	store, err := desktophost.OpenDraftMarkerStore(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	keyA, err := store.Save(markerA)
	if err != nil {
		t.Fatal(err)
	}
	keyB, err := store.Save(markerB)
	if err != nil {
		t.Fatal(err)
	}
	if keyA == keyB {
		t.Fatal("two Workspaces shared a Draft marker key")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	// Process restart is represented by reopening the same private store
	// without carrying any in-memory key or review/approval state.
	reopened, err := desktophost.OpenDraftMarkerStore(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	discovered, err := reopened.DiscoverKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(discovered) != 2 || !slices.Contains(discovered, keyA) || !slices.Contains(discovered, keyB) {
		t.Fatalf("missing recovered marker inventory: %v", discovered)
	}
	// DiscoverKeys lists the host-private store, NOT a Workspace-scoped UI
	// view. The embedding Host must not disclose this complete list to
	// an arbitrary WebView/client.
	probes := 0
	observe := func(root string) (desktophost.SourceObservation, error) {
		probes++
		switch root {
		case workspaceA:
			return markerA.Baseline, nil
		case workspaceB:
			return markerB.Baseline, nil
		default:
			t.Fatalf("unexpected Workspace source probe: %q", root)
			return desktophost.SourceObservation{}, nil
		}
	}
	if _, err := reopened.Recover(keyA, workspaceB, observe); !errors.Is(err, desktophost.ErrInvalidDraftMarker) || probes != 0 {
		t.Fatalf("cross-Workspace marker observed Source: err=%v probes=%d", err, probes)
	}
	if _, err := reopened.Recover(keyB, workspaceA, observe); !errors.Is(err, desktophost.ErrInvalidDraftMarker) || probes != 0 {
		t.Fatalf("reverse cross-Workspace marker observed Source: err=%v probes=%d", err, probes)
	}

	for _, tc := range []struct {
		key, root string
	}{
		{keyA, workspaceA},
		{keyB, workspaceB},
	} {
		recovery, err := reopened.Recover(tc.key, tc.root, observe)
		if err != nil || recovery.Status != desktophost.RecoveryNeedsReview {
			t.Fatalf("restarted Draft cannot become Accepted: recovery=%+v err=%v", recovery, err)
		}
	}
	if probes != 2 {
		t.Fatalf("expected one fresh source observation per authorized Workspace, got %d", probes)
	}
	recovery, err := reopened.Recover(keyA, workspaceA, func(root string) (desktophost.SourceObservation, error) {
		changed := markerA.Baseline
		changed.Artifacts = []desktophost.ObservedArtifact{{Path: "tei.workspace.yaml", Fingerprint: "sha256:changed"}}
		return changed, nil
	})
	if err != nil || recovery.Status != desktophost.RecoverySourceChanged {
		t.Fatalf("stale Draft must not become Accepted: recovery=%+v err=%v", recovery, err)
	}
	recovery, err = reopened.Recover(keyA, workspaceA, func(string) (desktophost.SourceObservation, error) {
		return desktophost.SourceObservation{}, errors.New("source unavailable")
	})
	if err != nil || recovery.Status != desktophost.RecoverySourceUnavailable {
		t.Fatalf("unavailable Source must not become Accepted: recovery=%+v err=%v", recovery, err)
	}
}

// os.Root here restricts ONLY the immutable private Draft-marker store.
// It is not the missing Wails Desktop Host's general Workspace-file guard.
func TestDraftStoreRejectsRootAliasAndUntrustedLocator(t *testing.T) {
	private := t.TempDir()
	if runtime.GOOS != "windows" {
		if err := os.Chmod(private, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := desktophost.OpenDraftMarkerStore("."); !errors.Is(err, desktophost.ErrDraftStoreRefused) {
		t.Fatalf("relative store path was accepted: %v", err)
	}
	link := filepath.Join(t.TempDir(), "fake-private-store")
	if err := os.Symlink(private, link); err == nil {
		if _, err := desktophost.OpenDraftMarkerStore(link); !errors.Is(err, desktophost.ErrDraftStoreRefused) {
			t.Fatalf("symlink store root was accepted: %v", err)
		}
	} else {
		t.Logf("symlink creation unavailable on this OS: %v", err)
	}
	store, err := desktophost.OpenDraftMarkerStore(private)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	probes := 0
	_, err = store.Recover("../../other-workspace", "/authorized/workspace", func(string) (desktophost.SourceObservation, error) {
		probes++
		return desktophost.SourceObservation{}, nil
	})
	if !errors.Is(err, desktophost.ErrInvalidDraftMarker) || probes != 0 {
		t.Fatalf("untrusted marker locator caused source access: err=%v calls=%d", err, probes)
	}
}

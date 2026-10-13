package desktophost

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// The Host must explicitly provision a private state directory; generic
// test temporary directories are not an authority or a permission guarantee.
func privateDraftStateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func storageMarker(t *testing.T) (DraftMarker, string) {
	t.Helper()
	root, source := fixtureWorkspace(t)
	baseline, err := ObserveSource(root)
	if err != nil {
		t.Fatal(err)
	}
	return DraftMarker{
		Version:           MarkerVersion,
		CandidateRef:      "draft-immutable-1",
		BaseModelRevision: "model-revision-observed-separately",
		Baseline:          baseline,
	}, source
}

func TestDraftMarkerStoreSurvivesReopenWithoutAccept(t *testing.T) {
	marker, source := storageMarker(t)
	state := privateDraftStateDir(t)
	store, err := OpenDraftMarkerStore(state)
	if err != nil {
		t.Fatal(err)
	}
	key, err := store.Save(marker)
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != 64 {
		t.Fatalf("invalid saved marker key: %s", key)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenDraftMarkerStore(state)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	recovered, err := store.Recover(key, marker.Baseline.WorkspaceRoot, ObserveSource)
	if err != nil || recovered.Status != RecoveryNeedsReview {
		t.Fatalf("reopened Marker must need review: %+v %v", recovered, err)
	}
	if recovered.Marker.CandidateRef != marker.CandidateRef {
		t.Fatalf("candidate ref changed: %+v", recovered)
	}
	if err := os.WriteFile(source, []byte("external source change"), 0600); err != nil {
		t.Fatal(err)
	}
	recovered, err = store.Recover(key, marker.Baseline.WorkspaceRoot, ObserveSource)
	if err != nil || recovered.Status != RecoverySourceChanged {
		t.Fatalf("external change must invalidate stored Marker: %+v %v", recovered, err)
	}
	content, err := os.ReadFile(source)
	if err != nil || string(content) != "external source change" {
		t.Fatalf("Draft recovery wrote Source: %q %v", content, err)
	}
	info, err := os.Stat(filepath.Join(state, "draft-"+key+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("marker permissions are not private: %v", info.Mode())
	}
}

func TestDraftMarkerStoreRejectsSecondWindowOverwrite(t *testing.T) {
	marker, _ := storageMarker(t)
	state := privateDraftStateDir(t)
	store, err := OpenDraftMarkerStore(state)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.Save(marker)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	ok, refused := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrDraftMarkerExists):
			refused++
		default:
			t.Fatalf("unexpected concurrent save result: %v", err)
		}
	}
	if ok != 1 || refused != 1 {
		t.Fatalf("create-only storage allowed overwrite: ok=%d rejected=%d", ok, refused)
	}
}

func TestDraftMarkerStoreRejectsUntrustedLocatorAndWorkspace(t *testing.T) {
	marker, _ := storageMarker(t)
	store, err := OpenDraftMarkerStore(privateDraftStateDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key, err := store.Save(marker)
	if err != nil {
		t.Fatal(err)
	}
	for _, badKey := range []string{"../draft", "/tmp/other", strings.ToUpper(key), key + ".json", ""} {
		if _, err := store.Recover(badKey, marker.Baseline.WorkspaceRoot, ObserveSource); !errors.Is(err, ErrInvalidDraftMarker) {
			t.Fatalf("invalid key %q was accepted: %v", badKey, err)
		}
	}
	if _, err := store.Recover(key, t.TempDir(), func(string) (SourceObservation, error) {
		t.Fatal("untrusted Marker attempted to open a different Workspace")
		return SourceObservation{}, nil
	}); !errors.Is(err, ErrInvalidDraftMarker) {
		t.Fatalf("unauthorized Workspace was accepted: %v", err)
	}
}

func TestDraftMarkerStoreRejectsSymlinksAndUnsafePermissions(t *testing.T) {
	marker, _ := storageMarker(t)
	state := privateDraftStateDir(t)
	store, err := OpenDraftMarkerStore(state)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := draftMarkerKey(marker)
	target := filepath.Join(state, "draft-"+key+".json")
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, target); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := store.Recover(key, marker.Baseline.WorkspaceRoot, ObserveSource); !errors.Is(err, ErrDraftStoreRefused) {
		t.Fatalf("symlink was followed: %v", err)
	}
	if _, err := store.Save(marker); !errors.Is(err, ErrDraftMarkerExists) {
		t.Fatalf("symlink was overwritten: %v", err)
	}

	if _, err := OpenDraftMarkerStore(target); !errors.Is(err, ErrDraftStoreRefused) {
		t.Fatalf("symlink directory accepted: %v", err)
	}
	if runtime.GOOS != "windows" {
		unprotected := t.TempDir()
		if err := os.Chmod(unprotected, 0755); err != nil {
			t.Fatal(err)
		}
		if _, err := OpenDraftMarkerStore(unprotected); !errors.Is(err, ErrDraftStoreRefused) {
			t.Fatalf("world-readable state directory accepted: %v", err)
		}
	}
}

func TestDraftMarkerStoreNeverPersistsCandidatePayload(t *testing.T) {
	marker, _ := storageMarker(t)
	state := privateDraftStateDir(t)
	store, err := OpenDraftMarkerStore(state)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key, err := store.Save(marker)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(state, "draft-"+key+".json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"candidateBytes", "authorization", "permission", "lease", "command"} {
		if strings.Contains(string(raw), "\""+forbidden+"\"") {
			t.Fatalf("Marker unexpectedly persisted %s", forbidden)
		}
	}
}

func TestDraftMarkerStoreRejectsTamperedIdentityBeforeWorkspaceRead(t *testing.T) {
	marker, _ := storageMarker(t)
	state := privateDraftStateDir(t)
	store, err := OpenDraftMarkerStore(state)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key, err := store.Save(marker)
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(state, "draft-"+key+".json")
	original, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	tampered := marker
	tampered.CandidateRef = "other-candidate-with-valid-metadata"
	wrongIdentity, err := EncodeDraftMarker(tampered)
	if err != nil {
		t.Fatal(err)
	}
	observations := 0
	observe := func(root string) (SourceObservation, error) {
		observations++
		if root != marker.Baseline.WorkspaceRoot {
			t.Fatalf("unexpected Workspace observed: %s", root)
		}
		return marker.Baseline, nil
	}
	for _, tc := range []struct {
		name string
		raw  []byte
	}{
		{name: "valid-json-wrong-key", raw: wrongIdentity},
		{name: "malformed-json", raw: []byte("{")},
		{name: "unknown-field", raw: []byte(`{"elevated":true}`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(filename, tc.raw, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := store.Recover(key, marker.Baseline.WorkspaceRoot, observe)
			if !errors.Is(err, ErrInvalidDraftMarker) {
				t.Fatalf("corrupted immutable marker was not rejected: %v", err)
			}
			if observations != 0 {
				t.Fatalf("invalid persisted marker caused %d Workspace observations", observations)
			}
		})
	}
	if err := os.WriteFile(filename, original, 0o600); err != nil {
		t.Fatal(err)
	}
	recovered, err := store.Recover(key, marker.Baseline.WorkspaceRoot, observe)
	if err != nil || recovered.Status != RecoveryNeedsReview || observations != 1 {
		t.Fatalf("valid marker did not observe once and require review: %+v, %v, calls=%d", recovered, err, observations)
	}
}

func TestDraftStorePermissionRevocationAfterOpen(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Host must verify Windows ACLs; Unix mode bits do not establish privacy")
	}
	marker, _ := storageMarker(t)
	state := privateDraftStateDir(t)
	store, err := OpenDraftMarkerStore(state)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key, err := store.Save(marker)
	if err != nil {
		t.Fatal(err)
	}
	next := marker
	next.CandidateRef = "candidate-after-permission-check"
	observations := 0
	observe := func(root string) (SourceObservation, error) {
		observations++
		return marker.Baseline, nil
	}

	// A long-lived Host does not retain private-directory authority from
	// OpenDraftMarkerStore forever. Even an execute-only group bit is unsafe.
	for _, mode := range []os.FileMode{0o755, 0o710} {
		if err := os.Chmod(state, mode); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Save(next); !errors.Is(err, ErrDraftStoreRefused) {
			t.Fatalf("Save allowed mode %04o: %v", mode, err)
		}
		if _, err := store.DiscoverKeys(); !errors.Is(err, ErrDraftStoreRefused) {
			t.Fatalf("Discover allowed mode %04o: %v", mode, err)
		}
		if _, err := store.Recover(key, marker.Baseline.WorkspaceRoot, observe); !errors.Is(err, ErrDraftStoreRefused) {
			t.Fatalf("Recover allowed mode %04o: %v", mode, err)
		}
		if observations != 0 {
			t.Fatalf("revoked directory caused %d Workspace observations", observations)
		}
		if _, err := os.Stat(filepath.Join(state, "draft-"+draftMarkerKey(next)+".json")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Save exposed an extra marker after permission revocation: %v", err)
		}
	}
	if err := os.Chmod(state, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save(next); err != nil {
		t.Fatalf("Restored private directory should permit Save: %v", err)
	}
	keys, err := store.DiscoverKeys()
	if err != nil || len(keys) != 2 {
		t.Fatalf("Restored directory discovery failed: %v, %v", keys, err)
	}
	recovered, err := store.Recover(key, marker.Baseline.WorkspaceRoot, observe)
	if err != nil || recovered.Status != RecoveryNeedsReview || observations != 1 {
		t.Fatalf("Restored privacy failed read-only recovery: %+v, %v, calls=%d", recovered, err, observations)
	}
}

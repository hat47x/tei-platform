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

func storageMarker(t *testing.T) (DraftMarker, string) {
	t.Helper()
	root, source := fixtureWorkspace(t)
	baseline, err := ObserveSource(root)
	if err != nil {
		t.Fatal(err)
	}
	return DraftMarker{
		Version: MarkerVersion, CandidateRef: "draft-immutable-1",
		BaseModelRevision: "model-revision-observed-separately",
		Baseline: baseline,
	}, source
}

func TestDraftMarkerStoreSurvivesReopenWithoutAccept(t *testing.T) {
	marker, source := storageMarker(t)
	state := t.TempDir()
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
	state := t.TempDir()
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
	store, err := OpenDraftMarkerStore(t.TempDir())
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
	state := t.TempDir()
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
	state := t.TempDir()
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
		if strings.Contains(string(raw), "\"" + forbidden + "\"") {
			t.Fatalf("Marker unexpectedly persisted %s", forbidden)
		}
	}
}

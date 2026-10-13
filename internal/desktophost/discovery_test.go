package desktophost

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The child has a fresh process and receives NO previously saved Draft key.
// The Host supplies only its authorized private directory and Workspace.
// No UI state, candidate payload, or approval is reconstructed.
func TestDraftDiscoveryWithFreshHostProcess(t *testing.T) {
	if os.Getenv("TEI_DESKTOP_DISCOVERY_CHILD") == "yes" {
		dir := os.Getenv("TEI_DESKTOP_DISCOVERY_STORE")
		authorized := os.Getenv("TEI_DESKTOP_DISCOVERY_WORKSPACE")
		expected := RecoveryStatus(os.Getenv("TEI_DESKTOP_DISCOVERY_EXPECT"))
		store, err := OpenDraftMarkerStore(dir)
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		keys, err := store.DiscoverKeys()
		if err != nil || len(keys) != 1 {
			t.Fatalf("restarted Host could not discover exactly one Marker: %v, keys=%v", err, keys)
		}
		recovered, err := store.Recover(keys[0], authorized, ObserveSource)
		if err != nil || recovered.Status != expected {
			t.Fatalf("restarted Host must classify, not accept: %+v, %v, expected %s", recovered, err, expected)
		}
		fmt.Println("TEI_DESKTOP_DISCOVERY_CHILD_OK:" + string(recovered.Status))
		return
	}
	marker, source := storageMarker(t)
	stateDir := privateDraftStateDir(t)
	store, err := OpenDraftMarkerStore(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save(marker); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	restart := func(want RecoveryStatus) {
		t.Helper()
		cmd := exec.Command(os.Args[0], "-test.run=^TestDraftDiscoveryWithFreshHostProcess$", "-test.v")
		cmd.Env = append(os.Environ(),
			"TEI_DESKTOP_DISCOVERY_CHILD=yes",
			"TEI_DESKTOP_DISCOVERY_STORE="+stateDir,
			"TEI_DESKTOP_DISCOVERY_WORKSPACE="+marker.Baseline.WorkspaceRoot,
			"TEI_DESKTOP_DISCOVERY_EXPECT="+string(want))
		output, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(output), "TEI_DESKTOP_DISCOVERY_CHILD_OK:"+string(want)) {
			t.Fatalf("fresh process failed (expected %s): %v\n%s", want, err, output)
		}
	}
	restart(RecoveryNeedsReview)
	if err := os.WriteFile(source, []byte("external modification after Host crash"), 0o600); err != nil {
		t.Fatal(err)
	}
	restart(RecoverySourceChanged)
	got, err := os.ReadFile(source)
	if err != nil || string(got) != "external modification after Host crash" {
		t.Fatalf("recovery overwrote accepted Source: %q, %v", got, err)
	}
}

func TestDraftDiscoverySortedAndIgnoresInterruptedSave(t *testing.T) {
	first, _ := storageMarker(t)
	second := first
	second.CandidateRef = "draft-immutable-2"
	stateDir := privateDraftStateDir(t)
	store, err := OpenDraftMarkerStore(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	keys, err := store.DiscoverKeys()
	if err != nil || len(keys) != 0 {
		t.Fatalf("empty store unexpectedly contained drafts: %v %v", keys, err)
	}
	key1, err := store.Save(first)
	if err != nil {
		t.Fatal(err)
	}
	key2, err := store.Save(second)
	if err != nil {
		t.Fatal(err)
	}
	// Simulates a process that exited before publishing its private temp.
	partial := filepath.Join(stateDir, ".draft-write-"+strings.Repeat("a", 32))
	if err := os.WriteFile(partial, []byte("{unfinished"), 0o600); err != nil {
		t.Fatal(err)
	}
	keys, err = store.DiscoverKeys()
	if err != nil || len(keys) != 2 {
		t.Fatalf("only published Markers should be discoverable: %v %v", keys, err)
	}
	if keys[0] >= keys[1] || (keys[0] != key1 && keys[0] != key2) || (keys[1] != key1 && keys[1] != key2) {
		t.Fatalf("Draft inventory is not sorted and complete: %v, want %s and %s", keys, key1, key2)
	}
}

func TestDraftDiscoveryFailsClosedOnMalformedOrUnsafeStoreEntries(t *testing.T) {
	marker, _ := storageMarker(t)
	for _, tc := range []struct {
		name string
		add  func(t *testing.T, state string, key string)
	}{
		{name: "unrelated", add: func(t *testing.T, state, _ string) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(state, "unexpected.txt"), []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "invalid-marker-name", add: func(t *testing.T, state, _ string) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(state, "draft-not-a-sha256.json"), []byte("{}"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "symlink", add: func(t *testing.T, state, key string) {
			t.Helper()
			outside := filepath.Join(t.TempDir(), "outside.json")
			if err := os.WriteFile(outside, []byte("{}"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(state, "draft-"+key+".json")); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
		}},
		{name: "readable-by-others", add: func(t *testing.T, state, key string) {
			t.Helper()
			if runtime.GOOS == "windows" {
				t.Skip("Windows private directory ACL must be checked by the embedding Host")
			}
			path := filepath.Join(state, "draft-"+key+".json")
			if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "invalid-temporary-name", add: func(t *testing.T, state, _ string) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(state, ".draft-write-invalid"), []byte("{}"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := privateDraftStateDir(t)
			store, err := OpenDraftMarkerStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			tc.add(t, dir, draftMarkerKey(marker))
			if _, err := store.DiscoverKeys(); !errors.Is(err, ErrDraftStoreRefused) {
				t.Fatalf("unsafe entry should fail closed, got %v", err)
			}
		})
	}
}

func TestDraftDiscoveryRefusesClosedStore(t *testing.T) {
	store, err := OpenDraftMarkerStore(privateDraftStateDir(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DiscoverKeys(); !errors.Is(err, ErrDraftStoreRefused) {
		t.Fatalf("closed store was used: %v", err)
	}
}

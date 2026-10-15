package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/tools/internal/gobuilder"
)

// This uses the existing hermetic Go Builder on a real TEI plugin package,
// rather than directly invoking a toy compiler in the Artifact Store test.
// The test's BuildPlan is a local fixture, not independently authorized Host
// Evidence, and neither success nor rejection proves real Host acceptance.
func TestPublishLockedRealBuilderOutputUsesPinnedDigest(t *testing.T) {
	goExecutable, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	goExecutable, err = filepath.Abs(goExecutable)
	if err != nil {
		t.Fatal(err)
	}
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	buildDir := filepath.Join(t.TempDir(), "real-builder-outputs")
	if err := os.MkdirAll(buildDir, 0o700); err != nil {
		t.Fatal(err)
	}
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	firstPath := filepath.Join(buildDir, "first"+ext)
	secondPath := filepath.Join(buildDir, "second"+ext)
	if err := gobuilder.BuildPackage(repo, goExecutable, "./tools/reference-process-plugin",
		firstPath, filepath.Join(buildDir, "state-first")); err != nil {
		t.Fatalf("first real Builder execution failed: %v", err)
	}
	firstBytes, err := os.ReadFile(firstPath)
	if err != nil || len(firstBytes) == 0 {
		t.Fatalf("first real Builder output unavailable: %v", err)
	}

	// Pin the expected SHA-256 in a real LockedBuild value BEFORE running
	// the second Builder. Never derive or replace expected from the bytes
	// subsequently passed to publishLockedBuilderOutput.
	plan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"},
		PluginDefinitionFingerprint: strings.Repeat("a", 64),
		Sources: []plugin.BuildSource{{
			Kind: "reference-source", ID: "process-plugin-fixture",
			Locator: "repo://tools/reference-process-plugin",
			Digest:  digestBytes([]byte("test-only source closure marker")),
		}},
		Builder: plugin.BuilderIdentity{
			ID: core.MustCanonicalID("tei.builder.go.seed"), Version: "1.0.0",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  digestBytes([]byte("test-only builder identity marker")),
		},
		Recipe:     plugin.BuildRecipe{Kind: "go-package", Entrypoint: "./tools/reference-process-plugin"},
		OutputKind: plugin.BuildOutputRuntimeArtifact,
		Target:     "host-process/" + runtime.GOOS + "/" + runtime.GOARCH,
		Profile:    "reference",
	}
	locked, err := plugin.LockBuildOutput(plan, digestBytes(firstBytes))
	if err != nil {
		t.Fatalf("pin first Builder output in LockedBuild: %v", err)
	}
	expected := locked.ArtifactDigest

	if err := gobuilder.BuildPackage(repo, goExecutable, "./tools/reference-process-plugin",
		secondPath, filepath.Join(buildDir, "state-second")); err != nil {
		t.Fatalf("second real Builder execution failed: %v", err)
	}
	comparison, err := gobuilder.CompareArtifactFiles(firstPath, secondPath)
	if err != nil || !comparison.Reproducible ||
		comparison.FirstDigest != expected || comparison.SecondDigest != expected {
		t.Fatalf("real Builder outputs differ from the pinned digest: %+v err=%v", comparison, err)
	}
	secondBytes, err := os.ReadFile(secondPath)
	if err != nil || !bytes.Equal(secondBytes, firstBytes) {
		t.Fatalf("second real Builder output differs from pinned output: %v", err)
	}

	store := openTestStore(t, 64<<20)

	// Negative: deliberately alter only the publication input. The fixed
	// LockedBuild and valid second Builder output must remain untouched.
	modified := bytes.Clone(secondBytes)
	modified[0] ^= 1
	badPath := filepath.Join(buildDir, "deliberately-modified"+ext)
	if err := os.WriteFile(badPath, modified, 0o600); err != nil {
		t.Fatal(err)
	}
	rejected, err := store.publishLockedBuilderOutput(locked, badPath)
	if err == nil || len(rejected.Members) != 0 || locked.ArtifactDigest != expected {
		t.Fatalf("modified real Builder bytes were published or expected digest changed: %+v err=%v", rejected, err)
	}
	for _, forbiddenDigest := range []string{expected, digestBytes(modified)} {
		if _, _, err := store.GetBytes(forbiddenDigest); !errors.Is(err, errArtifactNotFound) {
			t.Fatalf("failed publication wrote unexpected CAS object %s: %v", forbiddenDigest, err)
		}
	}

	// Positive: read the actual second Builder output while it still exists,
	// publish under the preexisting lock, and verify both CAS receipts.
	published, err := store.publishLockedBuilderOutput(locked, secondPath)
	if err != nil || len(published.Members) != 1 {
		t.Fatalf("real Builder output failed pinned CAS publication: %+v err=%v", published, err)
	}
	member := published.Members[0]
	if member.Role != "executable" || member.Digest != expected ||
		member.Published.Digest != expected || member.Published.Operation != "put" ||
		member.Reverified.Digest != expected || member.Reverified.Operation != "get" {
		t.Fatalf("real Builder output has incorrect publication evidence: %+v", member)
	}

	// The temporary Builder output is gone, but its original exact bytes
	// remain available by the unchanged lock digest after reopening the CAS.
	if err := os.RemoveAll(buildDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(secondPath); !os.IsNotExist(err) {
		t.Fatalf("Builder temp output still exists: %v", err)
	}
	reopened, err := openLocalArtifactStore(store.root, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	got, receipt, err := reopened.GetBytes(locked.ArtifactDigest)
	if err != nil || receipt.Operation != "get" || receipt.Digest != expected ||
		!bytes.Equal(got, secondBytes) {
		t.Fatalf("reopened CAS did not retain pinned real Builder bytes: %+v err=%v", receipt, err)
	}
}

// TestPublishLockedBuilderOutputReadBudget exercises the bridge's own byte
// limit, separately from publishExactMembers's whole-set preflight. The
// after-Lstat growth case must not cause an unbounded ReadFile allocation.
func TestPublishLockedBuilderOutputReadBudget(t *testing.T) {
	const maxBytes int64 = 5
	t.Run("exact limit remains publishable", func(t *testing.T) {
		store := openTestStore(t, maxBytes)
		value := []byte("12345")
		path := filepath.Join(t.TempDir(), "builder-output")
		if err := os.WriteFile(path, value, 0o600); err != nil {
			t.Fatal(err)
		}
		locked := plugin.LockedBuild{ArtifactDigest: digestBytes(value)}
		published, err := store.publishLockedBuilderOutput(locked, path)
		if err != nil || len(published.Members) != 1 {
			t.Fatalf("output at exact maxBytes was not published: %+v err=%v", published, err)
		}
		member := published.Members[0]
		if member.Digest != locked.ArtifactDigest ||
			member.Published.Digest != locked.ArtifactDigest ||
			member.Reverified.Digest != locked.ArtifactDigest ||
			member.Reverified.Operation != "get" {
			t.Fatalf("exact-boundary output lost fixed Lock identity: %+v", member)
		}
		got, _, err := store.GetBytes(locked.ArtifactDigest)
		if err != nil || !bytes.Equal(got, value) {
			t.Fatalf("exact-boundary CAS bytes mismatch: %v", err)
		}
	})

	t.Run("growth after Lstat rejected before publication", func(t *testing.T) {
		store := openTestStore(t, maxBytes)
		initial := []byte("abcde")
		path := filepath.Join(t.TempDir(), "builder-output")
		if err := os.WriteFile(path, initial, 0o600); err != nil {
			t.Fatal(err)
		}
		locked := plugin.LockedBuild{ArtifactDigest: digestBytes(initial)}
		var appended bool
		setTestHook(t, func(stage string) {
			if stage != hookAfterLstat || appended {
				return
			}
			appended = true
			file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.Write([]byte("!")); err != nil {
				file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
		})
		published, err := store.publishLockedBuilderOutput(locked, path)
		if !appended {
			t.Fatal("did not enlarge Builder output after the bridge Lstat")
		}
		if err == nil || !strings.Contains(err.Error(), "exceeds store maxBytes") || len(published.Members) != 0 {
			t.Fatalf("post-Lstat growth was accepted: %+v err=%v", published, err)
		}
		info, err := os.Stat(path)
		if err != nil || info.Size() != maxBytes+1 {
			t.Fatalf("negative fixture is not exactly maxBytes+1: %v info=%v", err, info)
		}
		for _, digest := range []string{locked.ArtifactDigest, digestBytes([]byte("abcde!"))} {
			if _, _, err := store.GetBytes(digest); !errors.Is(err, errArtifactNotFound) {
				t.Fatalf("rejected output appeared in CAS (%s): %v", digest, err)
			}
		}
		if locked.ArtifactDigest != digestBytes(initial) {
			t.Fatal("failure rewrote the expected LockedBuild digest")
		}
	})

	t.Run("nonregular input never published", func(t *testing.T) {
		store := openTestStore(t, maxBytes)
		locked := plugin.LockedBuild{ArtifactDigest: digestBytes([]byte("abcde"))}
		directory := filepath.Join(t.TempDir(), "not-a-regular-output")
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		published, err := store.publishLockedBuilderOutput(locked, directory)
		if err == nil || !strings.Contains(err.Error(), "must be a regular file") || len(published.Members) != 0 {
			t.Fatalf("nonregular Builder output accepted: %+v err=%v", published, err)
		}
		if _, _, err := store.GetBytes(locked.ArtifactDigest); !errors.Is(err, errArtifactNotFound) {
			t.Fatalf("nonregular output was published: %v", err)
		}
	})
}

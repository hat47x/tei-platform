package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"tei/tools/internal/gobuilder"
)

// TestRealGoBuildPreExtractionRecipeParity builds the two real self-build
// subjects from ONE checkout using the Go invocation frozen from before the
// shared BuildPackage extraction and the current adapter invocation.
//
// This is an actual byte-for-byte artifact comparison, not a comparison of
// two hashes recorded by the same implementation. Both builds use the same
// exact Go binary and the same selected source closure; their state dirs and
// output paths are separate. It is NOT a run of a historical repository
// executable, and does not attest that different source revisions are equal.
func TestRealGoBuildPreExtractionRecipeParity(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	goPath, err = filepath.EvalSymlinks(goPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("Git needed to verify the actual tracked source closure: %v", err)
	}
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	repo, err = filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	const digestPrefix = "sha256:"

	for _, target := range []struct {
		name    string
		pkgPath string
	}{
		{name: "exact-runtime-engine", pkgPath: "./tools/reference-transform-runtime-engine"},
		{name: "process-plugin", pkgPath: "./tools/reference-process-plugin"},
	} {
		t.Run(target.name, func(t *testing.T) {
			tmp := t.TempDir()
			baselineState := filepath.Join(tmp, "old-state")
			adapterState := filepath.Join(tmp, "adapter-state")
			// Snapshot actual Go-selected inputs before building. The repo
			// checkout is the same for both invocations and output is
			// intentionally kept outside it.
			sourceBefore, fpBefore, err := gobuilder.ResolveSourceClosure(
				repo, goPath, target.pkgPath, filepath.Join(tmp, "list-before"),
			)
			if err != nil {
				t.Fatalf("resolve actual pre-build source closure: %v", err)
			}
			if len(sourceBefore) == 0 || !strings.HasPrefix(fpBefore, digestPrefix) {
				t.Fatalf("missing compiler-selected input Evidence: members=%d fingerprint=%s",
					len(sourceBefore), fpBefore)
			}

			ext := ""
			if runtime.GOOS == "windows" {
				ext = ".exe"
			}
			baselineOutput := filepath.Join(tmp, "pre-extraction"+ext)
			adapterOutput := filepath.Join(tmp, "shared-adapter"+ext)
			// Exact historical invocation shape, deliberately independent
			// of BuildPackage. Environment() already had been shared by
			// both old References before build execution was extracted.
			env, err := gobuilder.Environment(os.Environ(), baselineState, runtime.GOOS, runtime.GOARCH)
			if err != nil {
				t.Fatal(err)
			}
			legacy := exec.Command(goPath, "build", "-trimpath", "-buildvcs=false",
				"-mod=readonly", "-o", baselineOutput, target.pkgPath)
			legacy.Dir = repo
			legacy.Env = env
			if out, err := legacy.CombinedOutput(); err != nil {
				t.Fatalf("pre-extraction Go invocation failed: %v: %s", err, out)
			}
			if err := gobuilder.BuildPackage(repo, goPath, target.pkgPath, adapterOutput, adapterState); err != nil {
				t.Fatalf("shared Go adapter failed: %v", err)
			}

			sourceAfter, fpAfter, err := gobuilder.ResolveSourceClosure(
				repo, goPath, target.pkgPath, filepath.Join(tmp, "list-after"),
			)
			if err != nil {
				t.Fatalf("resolve actual post-build source closure: %v", err)
			}
			if fpBefore != fpAfter || !reflect.DeepEqual(sourceBefore, sourceAfter) {
				t.Fatalf("source changed while comparing builders: before=%s after=%s", fpBefore, fpAfter)
			}
			if baselineState == adapterState {
				t.Fatal("two real builds shared a state directory")
			}
			beforeBytes, err := os.ReadFile(baselineOutput)
			if err != nil {
				t.Fatal(err)
			}
			afterBytes, err := os.ReadFile(adapterOutput)
			if err != nil {
				t.Fatal(err)
			}
			before := sha256.Sum256(beforeBytes)
			after := sha256.Sum256(afterBytes)
			beforeDigest := digestPrefix + hex.EncodeToString(before[:])
			afterDigest := digestPrefix + hex.EncodeToString(after[:])
			if len(beforeBytes) == 0 || len(afterBytes) == 0 || !bytes.Equal(beforeBytes, afterBytes) {
				t.Fatalf("real Go binary changed between legacy invocation and common adapter: old=%s new=%s source=%s",
					beforeDigest, afterDigest, fpBefore)
			}
			t.Logf("same exact Go binary: package=%s source=%s old=%s new=%s",
				target.pkgPath, fpBefore, beforeDigest, afterDigest)
		})
	}
}

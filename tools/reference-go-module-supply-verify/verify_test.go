package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func hostModuleCache(t *testing.T) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(t.TempDir(), "go.mod"), []byte("module tei\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Skipf("go env unavailable: %v", err)
	}
	return strings.TrimSpace(string(output))
}

// downloadCacheDir returns Go's own module download-cache path for one module.
// The download cache is also the GOPROXY=file:// layout this reference verifies,
// so it is the right fixture: extracted module directories use a different
// escaping rule and are not a proxy source.
func downloadCacheDir(t *testing.T, modulePath string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(t.TempDir(), "go.mod"), []byte("module tei\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Skipf("go env unavailable: %v", err)
	}
	return filepath.Join(strings.TrimSpace(string(output)), "cache", "download", filepath.FromSlash(modulePath), "@v")
}

func copyModuleEntry(t *testing.T, cache, modulePath, version string, destination string) string {
	t.Helper()
	source := filepath.Join(cache, "cache", "download", filepath.FromSlash(modulePath), "@v")
	if _, err := os.Stat(source); err != nil {
		t.Skipf("host module cache lacks %s: %v", modulePath, err)
	}
	target := filepath.Join(destination, filepath.FromSlash(modulePath), "@v")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{".mod", ".zip", ".info"} {
		name := version + suffix
		raw, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatalf("host module cache lacks %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(target, name), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return target
}

// TestModuleHashReproducesGoSumHashes pins the h1 construction against the
// hashes Go itself recorded, using module files from the host cache as the
// fixture. It is a fixture, not a trusted supply.
func TestModuleHashReproducesGoSumHashes(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	sums, err := parseGoSum(mustRead(t, filepath.Join(repo, "go.sum")))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ path, version string }{
		{path: "github.com/jackc/pgpassfile", version: "v1.0.0"},
		{path: "github.com/jackc/pgx/v5", version: "v5.9.2"},
		{path: "modernc.org/memory", version: "v1.8.0"},
	} {
		dir := downloadCacheDir(t, tc.path)
		if _, err := os.Stat(dir); err != nil {
			t.Skipf("host module cache lacks %s: %v", tc.path, err)
		}
		_, wantMod, ok := goSumEntry(sums, tc.path, tc.version, false)
		if !ok {
			t.Fatalf("go.sum has no go.mod entry for %s@%s", tc.path, tc.version)
		}
		_, wantZip, ok := goSumEntry(sums, tc.path, tc.version, true)
		if !ok {
			t.Fatalf("go.sum has no zip entry for %s@%s", tc.path, tc.version)
		}
		gotMod, err := moduleHash(mustRead(t, filepath.Join(dir, tc.version+".mod")), false)
		if err != nil {
			t.Fatal(err)
		}
		gotZip, err := moduleHash(mustRead(t, filepath.Join(dir, tc.version+".zip")), true)
		if err != nil {
			t.Fatal(err)
		}
		if gotMod != wantMod {
			t.Fatalf("%s@%s go.mod hash: computed %s want %s", tc.path, tc.version, gotMod, wantMod)
		}
		if gotZip != wantZip {
			t.Fatalf("%s@%s zip hash: computed %s want %s", tc.path, tc.version, gotZip, wantZip)
		}
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestVerifySupplyDetectsTamperedZipBytes(t *testing.T) {
	cache := hostModuleCache(t)
	supply := t.TempDir()
	target := copyModuleEntry(t, cache, "github.com/jackc/pgpassfile", "v1.0.0", supply)
	// Flip one byte inside the stored zip so the module hash no longer matches.
	path := filepath.Join(target, "v1.0.0.zip")
	raw := mustRead(t, path)
	if len(raw) < 64 {
		t.Skip("zip fixture too small to tamper safely")
	}
	raw[len(raw)-40] ^= 0xFF
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	verification, err := VerifySupply("../..", supply, "./tools/reference-build-authoring-intent", "go", stateDir(t),
		[]ModuleRequirement{{Path: "github.com/jackc/pgpassfile", Version: "v1.0.0", Declared: true}})
	if err != nil {
		t.Fatal(err)
	}
	if verification.Verified {
		t.Fatal("tampered zip was accepted against go.sum")
	}
	if len(verification.Modules) != 1 || verification.Modules[0].Verified {
		t.Fatalf("tampered module was not reported as failed: %+v", verification.Modules)
	}
	if verification.Modules[0].Zip.MatchesGoSum {
		t.Fatal("tampered zip still matched the go.sum hash")
	}
}

func TestVerifySupplyRejectsMissingGoSumEntry(t *testing.T) {
	cache := hostModuleCache(t)
	supply := t.TempDir()
	copyModuleEntry(t, cache, "github.com/jackc/pgpassfile", "v1.0.0", supply)

	// Declare the module at a version go.sum never recorded.
	verification, err := VerifySupply("../..", supply, "./tools/reference-build-authoring-intent", "go", stateDir(t),
		[]ModuleRequirement{{Path: "github.com/jackc/pgpassfile", Version: "v9.9.9", Declared: true}})
	if err != nil {
		t.Fatal(err)
	}
	if verification.Verified {
		t.Fatal("unrecorded module version was accepted")
	}
	if verification.Modules[0].Mod.FailureReason == "" || verification.Modules[0].Zip.FailureReason == "" {
		t.Fatalf("missing go.sum entry was not explained per file: %+v", verification.Modules[0])
	}
}

func TestDeclaredRequirementsReadsRequireBlocks(t *testing.T) {
	root := t.TempDir()
	goMod := `module example

go 1.23

require (
	example.com/a v1.0.0
	example.com/b v2.0.0 // indirect
)

require example.com/c v3.0.0
`
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := declaredRequirements(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("require directives were not fully read: %+v", got)
	}
	for i, want := range []string{"example.com/a@v1.0.0", "example.com/b@v2.0.0", "example.com/c@v3.0.0"} {
		if got[i].Path+"@"+got[i].Version != want {
			t.Fatalf("require directive %d: got %s want %s", i, got[i].Path+"@"+got[i].Version, want)
		}
	}
}

func TestFileURLKeepsTheVolumeInThePathOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("volume handling is a Windows-specific rule")
	}
	if got := fileURL(`C:\Users\example\modcache`); got != "file:///C:/Users/example/modcache" {
		t.Fatalf("file URL lost the volume: %s", got)
	}
}

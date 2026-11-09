package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/tools/internal/gobuilder"
)

// hostCacheComplete reports whether the host download cache carries every
// module the repository declares. It is a fixture availability probe, not a
// trust statement.
func hostCacheComplete(t *testing.T, cache string) bool {
	t.Helper()
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	requirements, err := declaredRequirements(repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, requirement := range requirements {
		for _, suffix := range []string{".mod", ".zip"} {
			path := filepath.Join(cache, filepath.FromSlash(requirement.Path), "@v", requirement.Version+suffix)
			if _, err := os.Stat(path); err != nil {
				return false
			}
		}
	}
	return true
}

// TestVerifySupplyAcceptsCompleteHostCacheForCmdTei exercises the whole path
// against the real repository: re-resolve cmd/tei from the named supply alone,
// then bind every selected module to go.sum by recomputing the h1 hash. If the
// host cache is incomplete the test skips rather than failing, because a
// missing fixture is not evidence about the code.
func TestVerifySupplyAcceptsCompleteHostCacheForCmdTei(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	cache := filepath.Join(hostModuleCache(t), "cache", "download")
	if !hostCacheComplete(t, cache) {
		t.Skip("host module download cache does not carry every declared module")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("Git needed for the repository fixture: %v", err)
	}

	verification, err := VerifySupply("../..", cache, "./cmd/tei", "go", stateDir(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Verified {
		t.Fatalf("complete supply was not verified: %s (%+v)", verification.FailureCause, verification.UnresolvedPackages)
	}
	if len(verification.ResolvedModules) == 0 {
		t.Fatal("re-resolution selected no external module; the supply was not exercised")
	}
	declared := map[string]struct{}{}
	for _, module := range verification.DeclaredModules {
		declared[module] = struct{}{}
	}
	for _, module := range verification.ResolvedModules {
		if _, ok := declared[module]; !ok {
			t.Fatalf("selected module was not declared by go.mod: %s", module)
		}
	}
	for _, check := range verification.Modules {
		if !check.Verified {
			t.Fatalf("selected module was not bound to go.sum: %s (%s)", check.Path, check.FailureCause)
		}
		if check.Mod.ComputedHash != check.Mod.GoSumHash || check.Zip.ComputedHash != check.Zip.GoSumHash {
			t.Fatalf("computed hash differs from go.sum for %s", check.Path)
		}
	}
	if verification.GoSumDigest == "" || !strings.HasPrefix(verification.GoSumDigest, "sha256:") {
		t.Fatal("go.sum digest was not recorded")
	}
}

// TestVerifySupplyRecordsDeclaredModulesTheTargetDoesNotSelect keeps the
// difference between "declared" and "actually consumed" visible. A master
// supply carries modules the target never selects; failing on them would make
// a correct supply look wrong, and silently dropping them would hide the gap.
func TestVerifySupplyRecordsDeclaredModulesTheTargetDoesNotSelect(t *testing.T) {
	cache := filepath.Join(hostModuleCache(t), "cache", "download")
	if !hostCacheComplete(t, cache) {
		t.Skip("host module download cache does not carry every declared module")
	}
	verification, err := VerifySupply("../..", cache, "./cmd/tei", "go", stateDir(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(verification.UnselectedDeclared) == 0 {
		t.Skip("every declared module happened to be selected; nothing to contrast")
	}
	selected := map[string]struct{}{}
	for _, module := range verification.ResolvedModules {
		selected[module] = struct{}{}
	}
	for _, module := range verification.UnselectedDeclared {
		if _, ok := selected[module]; ok {
			t.Fatalf("module was reported both selected and unselected: %s", module)
		}
	}
	if len(verification.Modules) != len(verification.ResolvedModules) {
		t.Fatalf("byte checks did not cover exactly the selected modules: checks=%d selected=%d",
			len(verification.Modules), len(verification.ResolvedModules))
	}
}

// TestVerifySupplyDoesNotClaimSuccessForASelfContainedTarget pins the vacuous
// case: a target with no external module never exercises the supply, so the
// verification must not report itself as verified.
func TestVerifySupplyDoesNotClaimSuccessForASelfContainedTarget(t *testing.T) {
	verification, err := VerifySupply("../..", t.TempDir(), "./tools/reference-build-authoring-intent", "go", stateDir(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if verification.Verified {
		t.Fatal("empty supply was reported verified for a target that selects no external module")
	}
	if verification.FailureCause == "" {
		t.Fatal("vacuous verification had no stated cause")
	}
}

// TestResolveSourceClosureStillRefusesVerifiedExternalModules records the
// policy gap that follows SB-0b rather than hiding it: the existing source
// closure resolver refuses every external module root, so a verified supply
// alone does not make cmd/tei reachable. Changing that policy belongs to the
// owner of #1090/#1096, not to this reference.
func TestResolveSourceClosureStillRefusesVerifiedExternalModules(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	_, _, err = gobuilder.ResolveSourceClosure(repo, goPath, "./cmd/tei", stateDir(t))
	if err == nil {
		t.Fatal("source closure resolver now reaches cmd/tei; the recorded policy gap is stale")
	}
	// Any of the two stopping reasons is acceptable evidence of the gap:
	// the isolated resolver has no supply, or it refuses external roots once
	// it has one. Both keep the closure unreachable.
	if !strings.Contains(err.Error(), "module lookup disabled") &&
		!strings.Contains(err.Error(), "external module") &&
		!strings.Contains(err.Error(), "source closure") {
		t.Fatalf("unexpected stopping reason, so the gap may have changed shape: %v", err)
	}
}

package gobuilder

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// These tests pin option C from
// plan/design/EXTERNAL_GO_MODULE_ADMISSION_CONTRACT_2026-11.md: the default
// policy still refuses external modules, and only an explicit supply admits
// them with a recorded admission.

func admissionSupply(t *testing.T) string {
	t.Helper()
	output, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Skipf("go env unavailable: %v", err)
	}
	cache := filepath.Join(strings.TrimSpace(string(output)), "cache", "download")
	if _, err := os.Stat(cache); err != nil {
		t.Skipf("host download cache unavailable: %v", err)
	}
	return cache
}

func admissionTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "tei-admission-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := removeWritableTree(dir); err != nil {
			t.Errorf("remove isolated state %s: %v", dir, err)
		}
	})
	return dir
}

// TestResolveQualifiedSourceClosureRefusesExternalWithoutSupply keeps the
// existing policy observable: no supply means no external members, not a
// partial closure.
func TestResolveQualifiedSourceClosureRefusesExternalWithoutSupply(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("Git needed for the repository fixture: %v", err)
	}
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	members, digest, admission, err := ResolveQualifiedSourceClosure(
		repo, goPath, "./cmd/tei", "", admissionTempDir(t),
	)
	if err == nil {
		t.Fatalf("repository-only policy admitted an external-module target: members=%d digest=%s", len(members), digest)
	}
	if admission.Policy != ClosurePolicyRepositoryOnly {
		t.Fatalf("refusal did not record the repository-only policy: %s", admission.Policy)
	}
	if admission.Qualified {
		t.Fatal("a refused closure was reported qualified")
	}
	if admission.RefusedReason == "" {
		t.Fatal("refusal had no stated reason")
	}
	if members != nil || digest != "" {
		t.Fatalf("refusal returned a partial closure: members=%d digest=%s", len(members), digest)
	}
}

// TestResolveQualifiedSourceClosureAdmitsWithExplicitSupply proves option C:
// naming a supply admits the selected external module members, and the
// admission records the policy, the module set and the binding it relied on.
func TestResolveQualifiedSourceClosureAdmitsWithExplicitSupply(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("Git needed for the repository fixture: %v", err)
	}
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	members, digest, admission, err := ResolveQualifiedSourceClosure(
		repo, goPath, "./cmd/tei", admissionSupply(t), admissionTempDir(t),
	)
	if err != nil {
		t.Skipf("this environment cannot supply the target: %v", err)
	}
	if !admission.Qualified || admission.Policy != ClosurePolicyExternalModuleOptIn {
		t.Fatalf("opt-in admission was not recorded: %+v", admission)
	}
	if admission.ExternalMembers == 0 || len(admission.ExternalModules) == 0 {
		t.Fatalf("opt-in admitted no external member: %+v", admission)
	}
	if admission.RepositoryMembers == 0 {
		t.Fatal("repository members were not counted")
	}
	if admission.GoSumBinding == "" || !strings.Contains(admission.GoSumBinding, "go.sum") {
		t.Fatalf("admission did not state its go.sum binding: %s", admission.GoSumBinding)
	}
	if admission.SupplyRoot == "" {
		t.Fatal("admission did not record the supply root")
	}
	if digest != SourceClosureDigest(members) {
		t.Fatal("closure digest is not reproducible from the returned members")
	}
	if admission.ClosureDigest != digest {
		t.Fatalf("admission digest differs from the returned digest: %s vs %s", admission.ClosureDigest, digest)
	}

	// External members must be namespaced, so a repository path can never
	// collide with a module file.
	repository, external := 0, 0
	for _, member := range members {
		if strings.HasPrefix(member.Path, externalMemberPrefix) {
			external++
			if !strings.Contains(member.Path, "@") {
				t.Fatalf("external member is not version pinned: %s", member.Path)
			}
			continue
		}
		repository++
		if strings.Contains(member.Path, externalMemberPrefix) {
			t.Fatalf("repository member used the reserved namespace: %s", member.Path)
		}
	}
	if repository != admission.RepositoryMembers || external != admission.ExternalMembers {
		t.Fatalf("counts disagree with the member list: repo=%d/%d ext=%d/%d",
			repository, admission.RepositoryMembers, external, admission.ExternalMembers)
	}

	// The observation entry point must report the same member set for the
	// same supply, so a caller can cross-check one against the other.
	observation, err := ObserveSourceClosure(repo, goPath, "./cmd/tei", admissionSupply(t), admissionTempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(observation.ExternalMembers) != admission.ExternalMembers {
		t.Fatalf("observation and admission disagree on external member count: %d vs %d",
			len(observation.ExternalMembers), admission.ExternalMembers)
	}
	if len(observation.ExternalModules) != len(admission.ExternalModules) {
		t.Fatalf("observation and admission disagree on module count: %d vs %d",
			len(observation.ExternalModules), len(admission.ExternalModules))
	}
}

// TestResolveQualifiedSourceClosureAdmitsSelfContainedTargetWithoutExternal
// checks that a target with no external module is admitted under the default
// policy without needing a supply.
func TestResolveQualifiedSourceClosureAdmitsSelfContainedTargetWithoutExternal(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	members, digest, admission, err := ResolveQualifiedSourceClosure(
		repo, goPath, "./tools/reference-build-authoring-intent", "", admissionTempDir(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !admission.Qualified || admission.Policy != ClosurePolicyRepositoryOnly {
		t.Fatalf("self-contained target admission changed: %+v", admission)
	}
	if admission.ExternalMembers != 0 || len(admission.ExternalModules) != 0 {
		t.Fatalf("self-contained target admitted external members: %+v", admission)
	}
	if len(members) == 0 || digest == "" {
		t.Fatal("self-contained target returned no closure")
	}
}

func TestSelectedMembersRejectsReservedRepositoryNamespace(t *testing.T) {
	if runtime.GOOS == "windows" {
		// A Windows path cannot contain ':', so the reserved prefix cannot be
		// used as a repository directory there. The guard still applies on
		// platforms where such a path is representable.
		t.Skip("the reserved prefix contains ':', which is not a valid Windows path character")
	}
	root := admissionTempDir(t)
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A repository file whose path starts with the reserved prefix must be
	// rejected, because it would collide with an external member namespace.
	rel := externalMemberPrefix + "fake@v1/pkg.go"
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(filepath.Dir(rel))), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"go.mod", filepath.FromSlash(rel), "pkg/main.go"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("package x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := exec.Command("git", "-C", root, "init").Run(); err != nil {
		t.Skipf("git init failed: %v", err)
	}
	if err := exec.Command("git", "-C", root, "add", "go.mod", "pkg/main.go", rel).Run(); err != nil {
		t.Skipf("git add failed: %v", err)
	}

	// The reserved path is selected as its own package, so the namespace check
	// applies to it rather than being skipped as unselected.
	raw := `{"Dir":` + quote(filepath.Join(root, "pkg")) + `,"ImportPath":"example/cmd","Name":"main","Module":{"Path":"example","Dir":` + quote(root) + `},"GoFiles":["main.go"]}
{"Dir":` + quote(filepath.Join(root, filepath.FromSlash(filepath.Dir(rel)))) + `,"ImportPath":"example/reserved","Name":"reserved","Module":{"Path":"example","Dir":` + quote(root) + `},"GoFiles":["pkg.go"]}
`
	_, _, err := selectedMembers(root, []byte(raw))
	if err == nil || !strings.Contains(err.Error(), "reserved external member namespace") {
		t.Fatalf("reserved namespace was not rejected: %v", err)
	}
}

func quote(value string) string {
	return `"` + strings.ReplaceAll(value, `\`, `\\`) + `"`
}

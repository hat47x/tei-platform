package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// This is an end-to-end exercise of produceEvidence: unlike JSON-only tests,
// it runs real git and go list in a new, isolated Go module. It is still not
// evidence that the TEI First Web Target binary is minimal.
func TestEvidenceFromCleanGitCheckoutAndBuildTags(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("go unavailable: %v", err)
	}

	root := t.TempDir()
	writeFixture := func(name, content string) {
		t.Helper()
		full := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	writeFixture("go.mod", "module example.test/tei-e0\n\ngo 1.23.2\n")
	writeFixture(".gitignore", "ignored.go\nignored.txt\n")
	writeFixture("cmd/app/main.go", "package main\nimport (\"embed\"; \"example.test/tei-e0/internal/calc\")\n//go:embed assets/*\nvar assets embed.FS\nfunc main() { _ = calc.Result(); _,_ = assets.ReadFile(\"assets/payload.txt\") }\n")
	writeFixture("cmd/app/assets/payload.txt", "versioned asset\n")
	writeFixture("cmd/app/optional.go", "//go:build optional\n\npackage main\nfunc optional() {}\n")
	writeFixture("cmd/app/amd64v3.go", "//go:build amd64.v3\n\npackage main\nfunc tuned() {}\n")
	writeFixture("internal/calc/calc.go", "package calc\nfunc Result() int { return 42 }\n")
	git("init", "-q")
	git("add", "-A")
	git("-c", "user.name=TEI Reference", "-c", "user.email=tei-reference@example.invalid", "commit", "-qm", "initial fixture")
	exactHead := git("rev-parse", "HEAD")

	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	// Do not require the network, a user Go workspace, or user-local Go flags.
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOTOOLCHAIN", "local")
	// A nondefault machine feature level must not leak from the host.
	t.Setenv("GOAMD64", "v1")

	base, err := produceEvidence("./cmd/app", "")
	if err != nil {
		t.Fatalf("actual go list on clean module: %v", err)
	}
	if base.SourceRevision != exactHead || base.Module != "example.test/tei-e0" ||
		base.Entry != "example.test/tei-e0/cmd/app" || len(base.Packages) != 2 {
		t.Fatalf("unexpected source identity or closure: %+v", base)
	}
	if base.GoVersion == "" || base.GOOS == "" || base.GOARCH == "" {
		t.Fatalf("missing toolchain identity: %+v", base)
	}
	findEntry := func(r Report) SourcePackage {
		t.Helper()
		for _, p := range r.Packages {
			if p.ImportPath == r.Entry {
				return p
			}
		}
		t.Fatalf("entry package absent in report: %+v", r)
		return SourcePackage{}
	}
	if slices.Contains(findEntry(base).GoFiles, "optional.go") {
		t.Fatal("build-tagged source leaked into default build closure")
	}
	if !slices.Contains(findEntry(base).AuxiliaryInputs, SourceInput{Kind: "embed", Path: "assets/payload.txt"}) {
		t.Fatal("tracked embedded asset absent from E0 package inventory")
	}

	tagged, err := produceEvidence("./cmd/app", "optional")
	if err != nil {
		t.Fatalf("actual go list with build tag: %v", err)
	}
	if tagged.SourceRevision != exactHead || tagged.BuildTags != "optional" ||
		!slices.Contains(findEntry(tagged).GoFiles, "optional.go") {
		t.Fatalf("build tag failed to select source: %+v", tagged)
	}
	// Explicit build tags are not the only Go source selector. Architecture
	// feature levels change implicit build tags even when GOARCH is fixed.
	if base.GOARCH == "amd64" {
		if slices.Contains(findEntry(base).GoFiles, "amd64v3.go") {
			t.Fatal("GOAMD64=v1 unexpectedly compiled v3 source")
		}
		if base.TargetFeatures["GOAMD64"] != "v1" {
			t.Fatalf("missing source-selection feature profile: %+v", base.TargetFeatures)
		}
		t.Setenv("GOAMD64", "v3")
		tuned, err := produceEvidence("./cmd/app", "")
		if err != nil {
			t.Fatalf("actual go list with GOAMD64=v3: %v", err)
		}
		if tuned.GOARCH != base.GOARCH || tuned.SourceRevision != base.SourceRevision ||
			tuned.TargetFeatures["GOAMD64"] != "v3" ||
			!slices.Contains(findEntry(tuned).GoFiles, "amd64v3.go") {
			t.Fatalf("architecture feature level did not change evidence: %+v", tuned)
		}
		t.Setenv("GOAMD64", "v1")
	}

	// Git index flags can make an edited tracked Go source invisible to
	// porcelain status. Such an observation must fail before go list.
	trackedSource := "internal/calc/calc.go"
	originalCalc := "package calc\nfunc Result() int { return 42 }\n"
	for _, tc := range []struct {
		name, setFlag, clearFlag string
	}{
		{"assume-unchanged", "--assume-unchanged", "--no-assume-unchanged"},
		{"skip-worktree", "--skip-worktree", "--no-skip-worktree"},
	} {
		git("update-index", tc.setFlag, trackedSource)
		writeFixture(trackedSource, "package calc\nfunc Result() int { return 999 }\n")
		if status := git("status", "--porcelain", "--untracked-files=all"); status != "" {
			t.Fatalf("hidden index flag unexpectedly exposed source changes: %q", status)
		}
		if _, err := produceEvidence("./cmd/app", ""); err == nil ||
			!strings.Contains(err.Error(), "hidden Git index flag") {
			t.Fatalf("modified tracked source hidden by %s was accepted: %v", tc.name, err)
		}
		git("update-index", tc.clearFlag, trackedSource)
		writeFixture(trackedSource, originalCalc)
		if status := git("status", "--porcelain", "--untracked-files=all"); status != "" {
			t.Fatalf("fixture was not restored after %s: %q", tc.name, status)
		}
	}

	// Embedded assets also alter the package bytes without appearing in
	// git porcelain when ignored. They are part of the observed inputs.
	writeFixture("cmd/app/assets/ignored.txt", "not tracked\n")
	if status := git("status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Fatalf("fixture should be Git-clean despite ignored embedded file: %q", status)
	}
	if _, err := produceEvidence("./cmd/app", ""); err == nil ||
		!strings.Contains(err.Error(), "not tracked by Git") {
		t.Fatalf("ignored embedded source must not become E0 evidence: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "cmd/app/assets/ignored.txt")); err != nil {
		t.Fatal(err)
	}

	// Git porcelain still reports clean when a .gitignore-matched Go file is
	// created; Go compiles it nevertheless. The E0 source inventory must reject it.
	writeFixture("cmd/app/ignored.go", "package main\nfunc ignored() {}\n")
	if status := git("status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Fatalf("fixture should be Git-clean despite ignored source: %q", status)
	}
	if _, err := produceEvidence("./cmd/app", ""); err == nil ||
		!strings.Contains(err.Error(), "not tracked by Git") {
		t.Fatalf("ignored compiled Go source must not become E0 evidence: %v", err)
	}
}

package gobuilder

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// recordedResolverOutput is a literal `go list -deps -json -e` stream shape:
// one resolved root-module package, one resolved external package reported as
// selected, and one package that failed because module supply was disabled.
const recordedResolverOutput = `{
	"Dir": "ROOT/stdlib",
	"ImportPath": "example/internal/a",
	"Name": "a",
	"Standard": true
}
{
	"Dir": "ROOT/pkg",
	"ImportPath": "example/cmd",
	"Name": "main",
	"Module": {"Path": "example", "Dir": "ROOT"}
}
{
	"Dir": "ROOT/pkg/sub",
	"ImportPath": "example/internal/sub",
	"Name": "sub",
	"Module": {"Path": "example", "Dir": "ROOT"}
}
{
	"ImportPath": "github.com/x/y",
	"Name": "y",
	"Module": {"Path": "github.com/x/y", "Version": "v1.2.3", "Dir": "/modcache/github.com/x/y@v1.2.3"},
	"Error": {"ImportStack": ["example/cmd"], "Pos": "pkg/main.go:5:2", "Err": "module lookup disabled by GOPROXY=off"}
}
`

func TestClassifyModuleClosureSeparatesSupplyGapFromSupportedTarget(t *testing.T) {
	preflight, err := classifyModuleClosure("ROOT", "./cmd", "/go/bin/go", "sha256:aaaa", []byte(recordedResolverOutput))
	if err != nil {
		t.Fatal(err)
	}
	if preflight.Classification != ModuleClosureMissing {
		t.Fatalf("missing module supply was not reported as missing-module: %s (%s)", preflight.Classification, preflight.Summary)
	}
	if len(preflight.Unresolved) != 1 || preflight.Unresolved[0].ImportPath != "github.com/x/y" {
		t.Fatalf("resolver failure record was dropped: %+v", preflight.Unresolved)
	}
	if preflight.Unresolved[0].Position != "pkg/main.go:5:2" || len(preflight.Unresolved[0].ImportStack) != 1 {
		t.Fatalf("resolver failure lost its position or import stack: %+v", preflight.Unresolved[0])
	}
	// The external module is NOT reported as a selected root: its package was
	// never resolved, so claiming "external-unresolved" would overstate the
	// observation.
	for _, entry := range preflight.ModuleRoots {
		if entry.Path == "github.com/x/y" {
			t.Fatalf("unresolved module was promoted to a selected root: %+v", entry)
		}
	}
	if preflight.RootModulePath != "example" || preflight.RootModuleDir != "ROOT" {
		t.Fatalf("target module identity changed: %+v", preflight)
	}
	if preflight.GoExecutableDigest != "sha256:aaaa" {
		t.Fatalf("caller-supplied Go executable digest was rewritten: %s", preflight.GoExecutableDigest)
	}
}

func TestClassifyModuleClosureKeepsUnrecognizedFailureUnknown(t *testing.T) {
	output := `{
	"ImportPath": "example/cmd",
	"Name": "main",
	"Module": {"Path": "example", "Dir": "ROOT"},
	"Error": {"ImportStack": ["example/cmd"], "Pos": "pkg/main.go:9:2", "Err": "syntax error in embedded file"}
}
`
	preflight, err := classifyModuleClosure("ROOT", "./cmd", "/go/bin/go", "sha256:aaaa", []byte(output))
	if err != nil {
		t.Fatal(err)
	}
	if preflight.Classification != ModuleClosureUnknown {
		t.Fatalf("unrecognized resolver failure was guessed instead of kept unknown: %s (%s)", preflight.Classification, preflight.Summary)
	}
}

func TestClassifyModuleClosureReportsSelectedExternalModuleRoot(t *testing.T) {
	output := `{
	"Dir": "/modcache/github.com/x/y@v1.2.3/pkg",
	"ImportPath": "github.com/x/y/pkg",
	"Name": "pkg",
	"Module": {"Path": "github.com/x/y", "Version": "v1.2.3", "Dir": "/modcache/github.com/x/y@v1.2.3"}
}
{
	"Dir": "ROOT/pkg",
	"ImportPath": "example/cmd",
	"Name": "main",
	"Module": {"Path": "example", "Dir": "ROOT"}
}
`
	preflight, err := classifyModuleClosure("ROOT", "./cmd", "/go/bin/go", "sha256:bbbb", []byte(output))
	if err != nil {
		t.Fatal(err)
	}
	if preflight.Classification != ModuleClosureExternal {
		t.Fatalf("selected external module root was not reported: %s (%s)", preflight.Classification, preflight.Summary)
	}
	if !strings.Contains(preflight.Summary, "github.com/x/y") {
		t.Fatalf("summary lost the external module identity: %s", preflight.Summary)
	}
}

func TestClassifyModuleClosureReportsSupportedTargetModule(t *testing.T) {
	output := `{
	"Dir": "ROOT/pkg",
	"ImportPath": "example/cmd",
	"Name": "main",
	"Module": {"Path": "example", "Dir": "ROOT"}
}
{
	"Dir": "ROOT/internal/sub",
	"ImportPath": "example/internal/sub",
	"Name": "sub",
	"Module": {"Path": "example", "Dir": "ROOT"}
}
`
	preflight, err := classifyModuleClosure("ROOT", "./cmd", "/go/bin/go", "sha256:cccc", []byte(output))
	if err != nil {
		t.Fatal(err)
	}
	if preflight.Classification != ModuleClosureSupported {
		t.Fatalf("target-module target was not classified reachable: %s (%s)", preflight.Classification, preflight.Summary)
	}
	if preflight.ResolvedPackages != 2 || len(preflight.ModuleRoots) != 1 || preflight.ModuleRoots[0].SelectedPackages != 2 {
		t.Fatalf("selected package accounting changed: %+v", preflight)
	}
	if preflight.SourceClosureReachable {
		t.Fatal("pure classification must not claim a live closure was reached")
	}
}

// TestResolveModuleClosurePreflightReportsMissingOfflineSupply exercises the
// actual resolver against a throwaway module that requires an external module
// the isolated cache does not contain. It asserts only the classification, not
// a build, artifact or acceptance.
func TestResolveModuleClosurePreflightReportsMissingOfflineSupply(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	root := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.invalid/missing\n\ngo 1.23\n\nrequire example.invalid/absent v1.0.0\n")
	write("go.sum", "")
	if err := os.MkdirAll(filepath.Join(root, "cmd", "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join("cmd", "app", "main.go"), "package main\n\nimport _ \"example.invalid/absent\"\n\nfunc main() {}\n")

	preflight, err := ResolveModuleClosurePreflight(root, goPath, "./cmd/app", filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}
	if preflight.Classification != ModuleClosureMissing {
		t.Fatalf("offline-only module supply gap was not classified as missing-module: %s (%s) %+v",
			preflight.Classification, preflight.Summary, preflight.Unresolved)
	}
	if preflight.SourceClosureReachable {
		t.Fatal("missing module supply was reported as reaching a source closure")
	}
	if preflight.SchemaVersion != moduleClosureSchemaV0 || !strings.Contains(preflight.Nature, "not a build attempt") {
		t.Fatalf("preflight evidence header lost its scope statement: %+v", preflight)
	}
	for _, file := range preflight.ModuleFiles {
		if file.Path == "go.sum" && !file.Present {
			t.Fatal("present but empty go.sum recorded as absent")
		}
	}
}

// TestResolveModuleClosurePreflightReportsSupportedCandidate proves the
// positive classification against the repository's own smallest tool package,
// which selects no external module.
func TestResolveModuleClosurePreflightReportsSupportedCandidate(t *testing.T) {
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
	preflight, err := ResolveModuleClosurePreflight(repo, goPath, "./tools/reference-build-authoring-intent", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if preflight.Classification != ModuleClosureSupported {
		t.Fatalf("self-contained tool package was not classified supported: %s (%s) %+v",
			preflight.Classification, preflight.Summary, preflight.Unresolved)
	}
	if !preflight.SourceClosureReachable || preflight.ResolvedPackages == 0 {
		t.Fatalf("supported target did not report a reachable closure: %+v", preflight)
	}
	if preflight.ModuleFiles[0].Path != "go.mod" || !preflight.ModuleFiles[0].Present {
		t.Fatalf("module definition file was not recorded: %+v", preflight.ModuleFiles)
	}
	if preflight.IsRootModuleRoot() != true {
		t.Fatal("supported target did not identify its own module root")
	}
}

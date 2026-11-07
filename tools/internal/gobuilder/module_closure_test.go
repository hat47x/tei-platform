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

// TestClassifyModuleClosureSeparatesCorruptSupplyFromMissingModule pins the
// distinction the caller acts on: content that was supplied but does not match
// its recorded digest is replaced, while absent content is supplied. Collapsing
// both into `unknown` would leave the failure pointing at the Source.
func TestClassifyModuleClosureSeparatesCorruptSupplyFromMissingModule(t *testing.T) {
	output := `{
	"ImportPath": "example/cmd",
	"Name": "main",
	"Module": {"Path": "example", "Dir": "ROOT"}
}
{
	"ImportPath": "golang.org/x/text/unicode/norm",
	"Name": "norm",
	"Module": {"Path": "golang.org/x/text", "Version": "v0.41.0", "Dir": "/modcache/golang.org/x/text@v0.41.0"},
	"Error": {"ImportStack": ["example/cmd"], "Pos": "pkg/main.go:9:2", "Err": "unzip /modcache/cache/download/golang.org/x/text/@v/v0.41.0.zip: flate: corrupt input before offset 44"}
}
`
	preflight, err := classifyModuleClosure("ROOT", "./cmd", "/go/bin/go", "sha256:aaaa", []byte(output))
	if err != nil {
		t.Fatal(err)
	}
	if preflight.Classification != ModuleClosureCorrupt {
		t.Fatalf("corrupt module supply was not classified as corrupt-supply: %s (%s)",
			preflight.Classification, preflight.Summary)
	}
	if !strings.Contains(preflight.Summary, "golang.org/x/text/unicode/norm") {
		t.Fatalf("corrupt-supply summary lost the failing import: %s", preflight.Summary)
	}
	if preflight.SourceClosureReachable {
		t.Fatal("corrupt supply was reported as reaching a source closure")
	}
}

// TestClassifyModuleClosureReportsVerifierMismatchAsCorruptSupply covers the
// message the Go verifier emits for an extracted tree that no longer matches
// its recorded hash. An artifact digest match cannot stand in for this check.
func TestClassifyModuleClosureReportsVerifierMismatchAsCorruptSupply(t *testing.T) {
	output := `{
	"ImportPath": "example/cmd",
	"Name": "main",
	"Module": {"Path": "example", "Dir": "ROOT"}
}
{
	"ImportPath": "golang.org/x/text",
	"Name": "text",
	"Module": {"Path": "golang.org/x/text", "Version": "v0.41.0", "Dir": "/modcache/golang.org/x/text@v0.41.0"},
	"Error": {"ImportStack": ["example/cmd"], "Pos": "pkg/main.go:9:2", "Err": "golang.org/x/text v0.41.0: dir has been modified (/modcache/golang.org/x/text@v0.41.0)"}
}
`
	preflight, err := classifyModuleClosure("ROOT", "./cmd", "/go/bin/go", "sha256:aaaa", []byte(output))
	if err != nil {
		t.Fatal(err)
	}
	if preflight.Classification != ModuleClosureCorrupt {
		t.Fatalf("verifier mismatch was not classified as corrupt-supply: %s (%s)",
			preflight.Classification, preflight.Summary)
	}
}

// TestClassifyModuleClosurePrefersUnknownOverCorruptSupply keeps an
// unrecognized failure above a recognized one, so a mixed stream is not read as
// a supply problem that a replacement would fix.
func TestClassifyModuleClosurePrefersUnknownOverCorruptSupply(t *testing.T) {
	output := `{
	"ImportPath": "example/cmd",
	"Name": "main",
	"Module": {"Path": "example", "Dir": "ROOT"}
}
{
	"ImportPath": "golang.org/x/text/unicode/norm",
	"Name": "norm",
	"Module": {"Path": "golang.org/x/text", "Version": "v0.41.0", "Dir": "/modcache/golang.org/x/text@v0.41.0"},
	"Error": {"ImportStack": ["example/cmd"], "Pos": "pkg/main.go:9:2", "Err": "flate: corrupt input before offset 44"}
}
{
	"ImportPath": "example/internal/broken",
	"Name": "broken",
	"Module": {"Path": "example", "Dir": "ROOT"},
	"Error": {"ImportStack": ["example/cmd"], "Pos": "internal/broken/x.go:3:1", "Err": "syntax error in embedded file"}
}
`
	preflight, err := classifyModuleClosure("ROOT", "./cmd", "/go/bin/go", "sha256:aaaa", []byte(output))
	if err != nil {
		t.Fatal(err)
	}
	if preflight.Classification != ModuleClosureUnknown {
		t.Fatalf("mixed unrecognized failure was folded into a recognized class: %s (%s)",
			preflight.Classification, preflight.Summary)
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

func TestClassifyModuleClosureDoesNotQualifyEmptyOrStandardOnlyResolverOutput(t *testing.T) {
	for _, tc := range []struct {
		name   string
		output string
	}{
		{name: "empty-stream"},
		{name: "standard-library-only", output: "{\"ImportPath\":\"fmt\",\"Standard\":true}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			preflight, err := classifyModuleClosure("ROOT", "./cmd", "/go/bin/go", "sha256:aaaa", []byte(tc.output))
			if err != nil {
				t.Fatal(err)
			}
			if preflight.Classification != ModuleClosureUnknown {
				t.Fatalf("target module not observed: classification=%s, want unknown", preflight.Classification)
			}
			if preflight.ResolvedPackages != 0 || len(preflight.ModuleRoots) != 0 || preflight.SourceClosureReachable {
				t.Fatalf("unobserved target was promoted to a reachable closure: %+v", preflight)
			}
		})
	}
}

package architecturecheck

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// These checks pin one narrow claim for the RMT Source Adapter (#950): the
// Canonical product packages neither need nor are changed by the adapter, and
// the adapter does not reach into the Canonical writer packages. They use the
// Go toolchain's resolved package graph (build constraints and transitive
// dependencies included), not only direct import text. They do not show that
// the adapter is correct, that it has no Runtime Authority, or that a future
// Plugin registration keeps this property.

const (
	xeadSourceDir = "internal/xeadsource"
	// Packages the product graph must be able to build without.
	canonicalProductPatterns = "./cmd/... ./core/... ./internal/... ./plugins/..."
)

type listedPackage struct {
	ImportPath string
	Deps       []string
}

// adapterOnlyPackages are the RMT-side packages whose presence must not be a
// build requirement of any Canonical product package.
func adapterOnlyPackages(module string) map[string]bool {
	return map[string]bool{
		module + "/" + xeadSourceDir:           true,
		module + "/tools/internal/rmtevidence": true,
		module + "/tools/internal/rmtlink":     true,
	}
}

// canonicalWriterPackages are packages that own Canonical source/World meaning
// or Workspace loading. The adapter must not depend on them, otherwise reading
// a source could become a second Canonical model path.
func canonicalWriterPackages(module string) map[string]bool {
	out := map[string]bool{}
	for _, name := range []string{"ir", "syntax", "validation", "workspace", "plugin", "sourceindex"} {
		out[module+"/internal/"+name] = true
	}
	return out
}

// productDependsOnAdapter reports "package -> dependency" for each package that
// is not itself the adapter yet transitively depends on an adapter-only package.
func productDependsOnAdapter(pkgs []listedPackage, forbidden map[string]bool, adapter string) []string {
	var out []string
	for _, pkg := range pkgs {
		if pkg.ImportPath == adapter {
			continue
		}
		deps := append([]string{pkg.ImportPath}, pkg.Deps...)
		for _, dep := range deps {
			if forbidden[dep] {
				out = append(out, pkg.ImportPath+" -> "+dep)
			}
		}
	}
	sort.Strings(out)
	return out
}

// adapterDependsOnWriters reports each Canonical writer package in the
// transitive dependencies of the adapter package.
func adapterDependsOnWriters(pkgs []listedPackage, writers map[string]bool, adapter string) []string {
	var out []string
	for _, pkg := range pkgs {
		if pkg.ImportPath != adapter {
			continue
		}
		for _, dep := range pkg.Deps {
			if writers[dep] {
				out = append(out, adapter+" -> "+dep)
			}
		}
	}
	sort.Strings(out)
	return out
}

func TestSourceAdapterIsolationClassification(t *testing.T) {
	forbidden := adapterOnlyPackages("tei")
	adapter := "tei/" + xeadSourceDir
	clean := []listedPackage{
		{ImportPath: "tei/internal/workspace", Deps: []string{"tei/internal/ir", "fmt"}},
		{ImportPath: adapter, Deps: []string{"encoding/xml"}},
	}
	if got := productDependsOnAdapter(clean, forbidden, adapter); len(got) != 0 {
		t.Fatalf("clean graph rejected: %v", got)
	}
	transitive := append(clean[:1:1], listedPackage{ImportPath: "tei/cmd/tei", Deps: []string{"tei/internal/workspace", adapter}})
	if got := productDependsOnAdapter(transitive, forbidden, adapter); len(got) != 1 || got[0] != "tei/cmd/tei -> "+adapter {
		t.Fatalf("transitive adapter dependency not reported: %v", got)
	}
	evidence := []listedPackage{{ImportPath: "tei/internal/query", Deps: []string{"tei/tools/internal/rmtevidence"}}}
	if got := productDependsOnAdapter(evidence, forbidden, adapter); len(got) != 1 {
		t.Fatalf("RMT Evidence dependency not reported: %v", got)
	}
	// A product package that is itself named like a forbidden package must also be reported.
	self := []listedPackage{{ImportPath: "tei/tools/internal/rmtlink"}}
	if got := productDependsOnAdapter(self, forbidden, adapter); len(got) != 1 {
		t.Fatalf("forbidden package as product root not reported: %v", got)
	}

	writers := canonicalWriterPackages("tei")
	leaky := []listedPackage{{ImportPath: adapter, Deps: []string{"tei/internal/ir", "tei/internal/core"}}}
	if got := adapterDependsOnWriters(leaky, writers, adapter); len(got) != 1 || got[0] != adapter+" -> tei/internal/ir" {
		t.Fatalf("adapter dependency on Canonical writer not reported: %v", got)
	}
	if got := adapterDependsOnWriters(clean, writers, adapter); len(got) != 0 {
		t.Fatalf("clean adapter rejected: %v", got)
	}
	if got := adapterDependsOnWriters([]listedPackage{{ImportPath: "tei/other", Deps: []string{"tei/internal/ir"}}}, writers, adapter); len(got) != 0 {
		t.Fatalf("non-adapter package must not be judged by the adapter rule: %v", got)
	}
}

func repoRootAndModule(t *testing.T) (string, string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	module, err := readModulePath(root)
	if err != nil {
		t.Fatal(err)
	}
	return root, module
}

func goCommand(root string, env []string, args ...string) *exec.Cmd {
	cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	return cmd
}

// listPackages resolves the Go build graph. Errors for unrelated packages are
// tolerated by -e, but an empty result is rejected by callers.
func listPackages(t *testing.T, root string, env []string, patterns string) []listedPackage {
	t.Helper()
	args := append([]string{"list", "-e", "-json=ImportPath,Deps"}, strings.Fields(patterns)...)
	cmd := goCommand(root, env, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go list failed: %v\n%s", err, stderr.String())
	}
	var pkgs []listedPackage
	dec := json.NewDecoder(&stdout)
	for {
		var pkg listedPackage
		if err := dec.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("decode go list output: %v", err)
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}

func TestCanonicalProductGraphDoesNotRequireSourceAdapter(t *testing.T) {
	root, module := repoRootAndModule(t)
	forbidden := adapterOnlyPackages(module)
	adapter := module + "/" + xeadSourceDir
	targets := map[string][]string{
		"native":  nil,
		"js/wasm": {"GOOS=js", "GOARCH=wasm", "CGO_ENABLED=0"},
	}
	for name, env := range targets {
		t.Run(name, func(t *testing.T) {
			pkgs := listPackages(t, root, env, canonicalProductPatterns)
			seen := map[string]bool{}
			for _, pkg := range pkgs {
				seen[pkg.ImportPath] = true
			}
			// The inspected set must be real: it contains the adapter (exempt) and the
			// Workspace loader that the adapter must not be needed by.
			for _, want := range []string{adapter, module + "/internal/workspace", module + "/cmd/tei"} {
				if !seen[want] {
					t.Fatalf("package %s missing from the inspected graph; the check would be vacuous", want)
				}
			}
			for _, leak := range productDependsOnAdapter(pkgs, forbidden, adapter) {
				t.Errorf("Canonical product graph depends on the RMT adapter: %s", leak)
			}
		})
	}
}

func TestSourceAdapterDetectsRealConsumers(t *testing.T) {
	// Positive control against the real graph: the Reference consumers do import
	// the adapter, so the detector above is able to see a real edge.
	root, module := repoRootAndModule(t)
	adapter := module + "/" + xeadSourceDir
	pkgs := listPackages(t, root, nil, "./tools/reference-xead-source-inspect ./tools/reference-plugin-capability-contract")
	if len(pkgs) != 2 {
		t.Fatalf("expected two Reference consumer packages, got %d", len(pkgs))
	}
	if got := productDependsOnAdapter(pkgs, adapterOnlyPackages(module), adapter); len(got) == 0 {
		t.Fatal("Reference consumers no longer reach the adapter in the resolved graph; the positive control is stale")
	}
}

func TestSourceAdapterDoesNotDependOnCanonicalWriters(t *testing.T) {
	root, module := repoRootAndModule(t)
	adapter := module + "/" + xeadSourceDir
	pkgs := listPackages(t, root, nil, "./"+xeadSourceDir)
	if len(pkgs) != 1 || pkgs[0].ImportPath != adapter {
		t.Fatalf("adapter package not resolved: %+v", pkgs)
	}
	for _, leak := range adapterDependsOnWriters(pkgs, canonicalWriterPackages(module), adapter) {
		t.Errorf("RMT adapter reaches a Canonical writer package: %s", leak)
	}
}

// TestCanonicalProductBuildsWithAdapterRemoved removes every Go file of the
// adapter from the build through an overlay, then builds the product packages.
func TestCanonicalProductBuildsWithAdapterRemoved(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the product packages")
	}
	root, _ := repoRootAndModule(t)
	files, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(xeadSourceDir), "*.go"))
	if err != nil || len(files) == 0 {
		t.Fatalf("adapter Go files not found: %v (%d)", err, len(files))
	}
	replace := map[string]string{}
	for _, file := range files {
		replace[file] = ""
	}
	encoded, err := json.Marshal(map[string]map[string]string{"Replace": replace})
	if err != nil {
		t.Fatal(err)
	}
	overlay := filepath.Join(t.TempDir(), "remove-adapter.json")
	if err := os.WriteFile(overlay, encoded, 0o600); err != nil {
		t.Fatal(err)
	}

	// Control: a package that really imports the adapter must fail to build, so
	// the removal is effective and a passing product build is not vacuous.
	// Establish that the consumer builds normally, then require the failed
	// overlay build to identify the removed adapter rather than an unrelated error.
	baselineOutput := filepath.Join(t.TempDir(), "reference-consumer")
	baseline := goCommand(root, nil, "build", "-o", baselineOutput, "./tools/reference-xead-source-inspect")
	if out, err := baseline.CombinedOutput(); err != nil {
		t.Fatalf("known Reference consumer does not build without the overlay: %v\n%s", err, out)
	}
	control := goCommand(root, nil, "build", "-overlay", overlay, "./tools/reference-xead-source-inspect")
	out, err := control.CombinedOutput()
	if err == nil {
		t.Fatalf("adapter removal had no effect on a known consumer:\n%s", out)
	}
	if !strings.Contains(string(out), "/"+xeadSourceDir) {
		t.Fatalf("overlay control failed without identifying the removed adapter: %v\n%s", err, out)
	}

	args := append([]string{"build", "-overlay", overlay}, strings.Fields(canonicalProductPatterns)...)
	cmd := goCommand(root, nil, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Canonical product packages do not build without the RMT adapter: %v\n%s", err, out)
	}
}

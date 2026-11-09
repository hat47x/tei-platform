package operations

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// forbiddenDependencyPrefixes lists transports and presentation hosts that a
// transport-neutral Operation must not reach, even transitively. CLI argv/exit
// handling, HTTP, RPC and Desktop WebView frameworks translate to and from the
// typed operations; the operations never import them (#1158).
//
// os/exec is intentionally not listed: it is currently reachable through the
// deployment executor that validation depends on. This test pins what is
// true today and does not claim the transitive closure is process-free.
var forbiddenDependencyPrefixes = []string{
	"tei/cmd/",
	"tei/tools/",
	"tei/reference/",
	"flag",
	"net",
	"syscall/js",
	"html/template",
	"github.com/wailsapp",
	"golang.org/x/net",
}

// forbiddenDependencies returns every dependency that is the forbidden path
// itself or lies beneath it. A forbidden prefix matches whole path elements,
// so "net" rejects "net/http" but not "network/other".
func forbiddenDependencies(deps []string) []string {
	var found []string
	for _, dep := range deps {
		for _, prefix := range forbiddenDependencyPrefixes {
			trimmed := strings.TrimSuffix(prefix, "/")
			if dep == trimmed || strings.HasPrefix(dep, trimmed+"/") {
				found = append(found, dep)
				break
			}
		}
	}
	return found
}

func TestForbiddenDependenciesMatchesWholePathElements(t *testing.T) {
	deps := []string{
		"fmt", "tei/internal/workspace", "tei/internal/network",
		"network", "net", "net/http", "flag", "tei/cmd/tei",
		"github.com/wailsapp/wails/v2", "syscall/js", "syscall",
		"golang.org/x/net/http2", "tei/reference/web-composition/go-host",
	}
	want := []string{
		"net", "net/http", "flag", "tei/cmd/tei",
		"github.com/wailsapp/wails/v2", "syscall/js",
		"golang.org/x/net/http2", "tei/reference/web-composition/go-host",
	}
	if got := forbiddenDependencies(deps); !reflect.DeepEqual(got, want) {
		t.Fatalf("forbidden dependency detection drifted:\n got=%v\nwant=%v", got, want)
	}
	if got := forbiddenDependencies([]string{"fmt", "tei/internal/workspace", "syscall"}); len(got) != 0 {
		t.Fatalf("allowed dependencies were rejected: %v", got)
	}
}

// TestOperationsDoNotDependOnTransportsOrHosts inspects the real transitive
// package graph of this package. It fails if a CLI, HTTP, RPC or Desktop host
// dependency is introduced, so each adapter stays a translation layer.
func TestOperationsDoNotDependOnTransportsOrHosts(t *testing.T) {
	goTool := filepath.Join(runtime.GOROOT(), "bin", "go")
	cmd := exec.Command(goTool, "list", "-deps", "-f", "{{.ImportPath}}", ".")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go list -deps failed: %v: %s", err, stderr.String())
	}
	deps := strings.Fields(stdout.String())
	// Guard against a vacuous pass: the graph must contain the shared
	// operation package itself and the model it projects.
	seen := map[string]bool{}
	for _, dep := range deps {
		seen[dep] = true
	}
	for _, required := range []string{"tei/internal/operations", "tei/internal/workspace", "tei/internal/validation"} {
		if !seen[required] {
			t.Fatalf("dependency listing is incomplete, missing %s (%d packages)", required, len(deps))
		}
	}
	if found := forbiddenDependencies(deps); len(found) != 0 {
		t.Fatalf("internal/operations must stay transport-neutral; forbidden dependencies: %v", found)
	}
	// Every non-standard dependency must belong to this module.
	for _, dep := range deps {
		if first, _, _ := strings.Cut(dep, "/"); strings.Contains(first, ".") {
			t.Fatalf("external module dependency %s is not allowed in the operation boundary", dep)
		}
	}
}

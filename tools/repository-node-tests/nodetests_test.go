// Package nodetests runs Node.js tests that no workflow, shell script or other
// Go test invoked (#2346).
//
// Each file below passed on its own when it was found unreferenced; this test
// makes `go test ./...` (and therefore tools/repository-health-check.sh and the
// existing Go CI) notice a regression. It does not make the tests a required
// check for every pull request, and it does not prove anything about a real
// browser or a real SUI host: the tests use Node's own fakes.
package nodetests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// unreferencedNodeTests are the *.test.mjs files that #2346 found were not
// executed from anywhere. Paths are relative to the repository root.
var unreferencedNodeTests = []string{
	"tools/reference-sui-interaction/bridge.test.mjs",
	"tools/reference-sui-interaction/native-edit.test.mjs",
	"tools/reference-sui-interaction/sui-http-ports.test.mjs",
	"tools/reference-studio-host-qualification/client_transport_contract.test.mjs",
	"reference/business-spec-templates/template-reference.test.mjs",
}

func TestUnreferencedNodeTestsPass(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			t.Fatal("Node.js is required on GitHub Actions")
		}
		t.Skip("Node.js tests not executed: Node.js is unavailable")
	}
	root := filepath.Join("..", "..")
	for _, rel := range unreferencedNodeTests {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			path := filepath.Join(root, filepath.FromSlash(rel))
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("listed Node test is missing (remove it from the list if it was deleted): %v", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, node, "--test", path)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("node --test %s: %v\n%s", rel, err, out)
			}
		})
	}
}

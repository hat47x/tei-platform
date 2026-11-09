package sharedprojection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// This is the first real Go package boundary usable by either Application.
// The compiled package dependency closure must not acquire a dependency on
// the combined First Web Target, Workspace loader, or legacy login fixtures.
// Passing this test does not by itself prove app-specific binary isolation.
func TestSharedProjectionDependencyClosureExcludesOtherApplicationAndAuthoring(t *testing.T) {
	command := exec.Command("go", "list", "-deps", "-json", ".")
	command.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go list shared projection dependencies: %v\n%s", err, output)
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	found := false
	for {
		var p struct {
			ImportPath string
			Incomplete bool
			Error      *struct{ Err string }
			DepsErrors []struct{ Err string }
		}
		if err := decoder.Decode(&p); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("parse Go package closure: %v", err)
		}
		if p.ImportPath == "" || p.Incomplete || p.Error != nil || len(p.DepsErrors) != 0 {
			t.Fatalf("invalid Go dependency node: %+v", p)
		}
		if p.ImportPath == "tei/tools/reference-first-web-target/sharedprojection" {
			found = true
		}
		for _, prefix := range []string{
			"tei/tools/reference-first-web-target/",
			"tei/internal/workspace",
			"tei/internal/security/legacyauth",
			"tei/internal/security/legacyauthorization",
		} {
			if (p.ImportPath == strings.TrimSuffix(prefix, "/") ||
				strings.HasPrefix(p.ImportPath, prefix)) &&
				p.ImportPath != "tei/tools/reference-first-web-target/sharedprojection" {
				t.Error(fmt.Sprintf("shared projection unexpectedly imports %q", p.ImportPath))
			}
		}
	}
	if !found {
		t.Fatal("shared projection package missing from observed Go dependency graph")
	}
}

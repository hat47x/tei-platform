package commandbinding_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestAuthorizationSourceDoesNotImportTheCommandBoundary pins the direction of
// the dependency: the authorization source (legacyauthorization) must not know
// about the Command boundary. If it did, every host that only needs the rules
// would compile the Command boundary and, through it, the Workspace loader. The
// frozen API build checks exactly that closure, and it broke once already.
func TestAuthorizationSourceDoesNotImportTheCommandBoundary(t *testing.T) {
	deps := packageClosure(t, "tei/internal/security/legacyauthorization")
	for _, forbidden := range []string{
		"tei/internal/command",
		"tei/internal/workspace",
		"tei/internal/syntax",
		"tei/internal/security/legacyauthorization/commandbinding",
	} {
		if deps[forbidden] {
			t.Errorf("the authorization source must not reach %s; put the Command binding in commandbinding", forbidden)
		}
	}
	if !deps["tei/internal/security/legacyauth"] {
		t.Error("the closure looks wrong: legacyauthorization must still depend on legacyauth")
	}
}

// TestCommandBindingReachesBothSides keeps the other half honest: the binding
// package is the one place that may see both.
func TestCommandBindingReachesBothSides(t *testing.T) {
	deps := packageClosure(t, "tei/internal/security/legacyauthorization/commandbinding")
	for _, required := range []string{
		"tei/internal/command",
		"tei/internal/security/legacyauthorization",
	} {
		if !deps[required] {
			t.Errorf("the binding must depend on %s", required)
		}
	}
}

// packageClosure returns the import closure of one package, read from the Go
// toolchain rather than from the source text.
func packageClosure(t *testing.T, path string) map[string]bool {
	t.Helper()
	command := exec.Command("go", "list", "-deps", "-json", path)
	command.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go list %s: %v\n%s", path, err, output)
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	result := map[string]bool{}
	for {
		var pkg struct {
			ImportPath string
			Incomplete bool
			Error      *struct{ Err string }
		}
		if err := decoder.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("parse closure of %s: %v", path, err)
		}
		if pkg.ImportPath == "" || pkg.Incomplete || pkg.Error != nil {
			t.Fatalf("invalid closure node for %s: %+v", path, pkg)
		}
		if strings.HasPrefix(pkg.ImportPath, "tei/") {
			result[pkg.ImportPath] = true
		}
	}
	return result
}

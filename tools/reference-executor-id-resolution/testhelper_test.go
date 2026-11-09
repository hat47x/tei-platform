package executoridresolution

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/core"
)

// write creates a parent directory before writing, so the Workspace fixture does
// not need a separate setup step.
func write(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func coreID(s string) core.CanonicalID { return core.CanonicalID(s) }

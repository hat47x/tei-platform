//go:build windows

package workspace_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
)

// These tests record Windows-only behaviour that cannot be exercised on Linux
// and that does not need SeCreateSymbolicLinkPrivilege (so they run on a plain
// Windows host, unlike the symlink tests that skip without Developer Mode).
//
// The classification is recorded in
// plan/evidence/WORKSPACE_SOURCE_READER_ENV_EVIDENCE_2026-11.md.

func requireJunction(t *testing.T, link, target string) {
	t.Helper()
	out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
	if err != nil {
		t.Skipf("directory junction unavailable: %v %s", err, strings.TrimSpace(string(out)))
	}
}

func writeEvidenceWorkspace(t *testing.T, files map[string]string, sources ...string) string {
	t.Helper()
	root := t.TempDir()
	manifest := "workspace: windows_evidence\nformatVersion: 1\nsources:\n"
	for _, source := range sources {
		manifest += "  - " + source + "\n"
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// A directory junction leaving the Workspace is refused before any source is
// parsed, so an attacker-controlled reparse point cannot contribute bytes.
func TestWindowsJunctionEscapeIsRefused(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.tei"), []byte("namespace secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := writeEvidenceWorkspace(t, map[string]string{"semantic/.keep": ""}, "semantic")
	requireJunction(t, filepath.Join(root, "semantic", "escape"), outside)

	_, err := workspace.Load(root)
	if err == nil || !strings.Contains(err.Error(), "outside the Workspace") {
		t.Fatalf("escaping junction was not refused: %v", err)
	}
}

// Unlike an in-root relative symlink, an in-root junction is refused too.
// The reader is deliberately conservative for reparse points on Windows; this
// test locks that behaviour so a future relaxation is an explicit decision.
func TestWindowsInternalJunctionIsRefusedConservatively(t *testing.T) {
	root := writeEvidenceWorkspace(t, map[string]string{
		"internal-target/inside.tei": "namespace inside\n",
	}, "semantic")
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	requireJunction(t, filepath.Join(root, "semantic", "alias"), filepath.Join(root, "internal-target"))

	_, err := workspace.Load(root)
	if err == nil || !strings.Contains(err.Error(), "outside the Workspace") {
		t.Fatalf("in-root junction was followed: %v", err)
	}
}

// Windows file systems are case-insensitive: a manifest may name the source
// root in a different case from the directory. The same physical file is still
// indexed once.
func TestWindowsCaseVariantSourceRootResolves(t *testing.T) {
	root := writeEvidenceWorkspace(t, map[string]string{"semantic/f.tei": "namespace case_variant\n"}, "Semantic")

	if _, err := workspace.Load(root); err != nil {
		t.Fatalf("case-variant source root was rejected: %v", err)
	}
}

// Two source roots that differ only by case reach the same files. The physical
// identity check must reject the duplicate instead of indexing the file twice.
func TestWindowsCaseVariantRootsAreRejectedAsSamePhysicalFile(t *testing.T) {
	root := writeEvidenceWorkspace(t, map[string]string{"semantic/f.tei": "namespace case_duplicate\n"}, "semantic", "SEMANTIC")

	_, err := workspace.Load(root)
	if err == nil || !strings.Contains(err.Error(), "same physical file") {
		t.Fatalf("case-variant duplicate roots were accepted: %v", err)
	}
}

// A Workspace opened through an administrative UNC share loads normally. The
// Workspace boundary is still the pinned root; what protects the share itself
// is the share ACL, not TEI.
func TestWindowsUNCPathWorkspaceLoads(t *testing.T) {
	root := writeEvidenceWorkspace(t, map[string]string{"semantic/f.tei": "namespace unc\n"}, "semantic")
	volume := filepath.VolumeName(root)
	if volume == "" {
		t.Skip("workspace is not on a drive-letter path")
	}
	unc := `\\localhost\` + strings.TrimSuffix(volume, ":") + "$" + strings.TrimPrefix(root, volume)
	if _, err := os.Stat(unc); err != nil {
		t.Skipf("administrative share is not reachable: %v", err)
	}

	if _, err := workspace.Load(unc); err != nil {
		t.Fatalf("UNC workspace did not load: %v", err)
	}
}

// A junction crossing drives is refused like any other escaping reparse point.
func TestWindowsCrossDriveJunctionIsRefused(t *testing.T) {
	root := writeEvidenceWorkspace(t, map[string]string{"semantic/.keep": ""}, "semantic")
	volume := filepath.VolumeName(root)
	target := ""
	for _, letter := range []string{"C:", "D:", "E:", "F:", "G:"} {
		if strings.EqualFold(letter, volume) {
			continue
		}
		if _, err := os.Stat(letter + `\`); err == nil {
			target = letter + `\`
			break
		}
	}
	if target == "" {
		t.Skip("no second drive available")
	}
	requireJunction(t, filepath.Join(root, "semantic", "cross"), target)

	_, err := workspace.Load(root)
	if err == nil || !strings.Contains(err.Error(), "outside the Workspace") {
		t.Fatalf("cross-drive junction was followed: %v", err)
	}
}

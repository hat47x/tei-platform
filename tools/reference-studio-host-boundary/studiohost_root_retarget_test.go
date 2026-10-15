//go:build tei_studio_host_integration

package hostboundary

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/studiohost"
)

func createHostRetargetFixture(t *testing.T, dir, namespace string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "semantic"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: root_boundary_fixture\nsources:\n  - semantic\n"
	if err := os.WriteFile(filepath.Join(dir, "tei.workspace.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	source := "namespace " + namespace + "\n\ntype Marker from std.text\n"
	if err := os.WriteFile(filepath.Join(dir, "semantic", "marker.tei"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
}

func catalogJSON(t *testing.T, host *studiohost.ReadOnly) []byte {
	t.Helper()
	r := host.CatalogWorkspace("workspace-1", "")
	if !r.OK || r.Value == nil || r.Failure != nil {
		t.Fatalf("fixture CatalogWorkspace unexpectedly failed: %+v", r.Failure)
	}
	out, err := json.Marshal(r.Value)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// Positive control: a host-configured symlink alias is resolved at startup.
// Changing ONLY the alias afterward must not retarget the fixed allowlist.
func TestStudioHostConfiguredAliasReplacementKeepsPinnedRealRoot(t *testing.T) {
	parent := t.TempDir()
	allowed := filepath.Join(parent, "allowed")
	outside := filepath.Join(parent, "outside")
	alias := filepath.Join(parent, "configured-alias")
	createHostRetargetFixture(t, allowed, "allowed")
	createHostRetargetFixture(t, outside, "secret")
	if err := os.Symlink(allowed, alias); err != nil {
		t.Skipf("creating symlinks is unavailable on this host: %v", err)
	}
	h, err := studiohost.NewReadOnly([]string{alias})
	if err != nil {
		t.Fatal(err)
	}
	before := catalogJSON(t, h)
	if !bytes.Contains(before, []byte("allowed")) {
		t.Fatalf("positive fixture does not identify the allowed namespace: %s", before)
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, alias); err != nil {
		t.Fatal(err)
	}
	after := catalogJSON(t, h)
	if !bytes.Equal(before, after) {
		t.Fatalf("replacing an input alias redirected Host's fixed root: before=%s after=%s", before, after)
	}
}

// Negative control: replacing the RESOLVED on-disk root itself must not permit
// a caller holding the original Workspace ID to read a different Workspace.
// An operation failure is acceptable. Returning outside's catalog is not.
//
// This does not model an in-flight OS fd race; it probes Host lifetime after
// startup, which is distinct from the Loader's per-Load os.Root pinning.
func TestStudioHostResolvedRootReplacementNeverReadsOutsideWorkspace(t *testing.T) {
	parent := t.TempDir()
	allowed := filepath.Join(parent, "allowed")
	outside := filepath.Join(parent, "outside")
	moved := filepath.Join(parent, "allowed-renamed")
	createHostRetargetFixture(t, allowed, "allowed")
	createHostRetargetFixture(t, outside, "secret")

	h, err := studiohost.NewReadOnly([]string{allowed})
	if err != nil {
		t.Fatal(err)
	}
	original := catalogJSON(t, h)
	outsideHost, err := studiohost.NewReadOnly([]string{outside})
	if err != nil {
		t.Fatal(err)
	}
	foreign := catalogJSON(t, outsideHost)
	if bytes.Equal(original, foreign) || !bytes.Contains(foreign, []byte("secret")) {
		t.Fatalf("insufficiently distinct fixtures: original=%s outside=%s", original, foreign)
	}

	if err := os.Rename(allowed, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, allowed); err != nil {
		t.Skipf("creating symlinks is unavailable on this host: %v", err)
	}
	got := h.CatalogWorkspace("workspace-1", "")
	if !got.OK {
		// Fail-closed is acceptable: the former root is no longer available.
		return
	}
	if got.Value == nil {
		t.Fatal("Host returned success without a catalog")
	}
	raw, err := json.Marshal(got.Value)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(raw, foreign) || bytes.Contains(raw, []byte("secret")) {
		t.Fatalf("SECURITY: a previously allowlisted Workspace ID now reads the outside Workspace after root path replacement; original=%s foreign=%s actual=%s", original, foreign, raw)
	}
}

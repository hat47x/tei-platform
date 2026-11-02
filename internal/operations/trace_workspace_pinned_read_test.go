package operations

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/modelref"
	"tei/internal/modelref/sourcebridge"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// makeDirAlias makes link point at target. It uses a symlink, or on Windows a
// directory junction, which needs no elevation. It skips (never passes) when
// the platform cannot make either, so a skipped test is not evidence.
func makeDirAlias(t *testing.T, link, target string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
		if err != nil {
			t.Skipf("junction creation unavailable: %v %s", err, out)
		}
		return
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const pinnedReadTei = "namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n}\n"

// Red on the EvalSymlinks-then-Open reader: it read any size.
func TestReadWorkspaceSourceRefusesOversizedSourceInsteadOfReadingIt(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "small.tei"), "abcd")
	writeTestFile(t, filepath.Join(root, "big.tei"), "abcde")
	previous := maxWorkspaceSourceBytes
	maxWorkspaceSourceBytes = 4
	t.Cleanup(func() { maxWorkspaceSourceBytes = previous })
	reader, err := readWorkspaceSource(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := reader("small.tei"); err != nil || string(got) != "abcd" {
		t.Fatalf("file at the limit = %q, %v", got, err)
	}
	if got, err := reader("big.tei"); err == nil {
		t.Fatalf("oversized source was read (%d bytes)", len(got))
	}
}

func TestReadWorkspaceSourceDefaultLimitIsBounded(t *testing.T) {
	if maxWorkspaceSourceBytes <= 0 || maxWorkspaceSourceBytes > 1<<30 {
		t.Fatalf("unreasonable default Source read limit: %d", maxWorkspaceSourceBytes)
	}
}

// Boundary guard (passes on the old reader too): a directory inside the
// Workspace that points outside must not be readable through.
func TestReadWorkspaceSourceRefusesAliasDirectoryEscapingWorkspace(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeTestFile(t, filepath.Join(outside, "secret.tei"), "OUTSIDE")
	makeDirAlias(t, filepath.Join(root, "semantic"), outside)
	reader, err := readWorkspaceSource(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := reader("semantic/secret.tei"); err == nil {
		t.Fatalf("read through an escaping alias returned %q", got)
	}
}

// Regression guard (passes on the old reader too, which failed closed because
// its containment check no longer matched): retargeting the Workspace root
// alias after the reader exists must fail the read, not move it.
func TestReadWorkspaceSourceFailsWhenRootAliasIsRetargeted(t *testing.T) {
	base := t.TempDir()
	first := filepath.Join(base, "first")
	second := filepath.Join(base, "second")
	writeTestFile(t, filepath.Join(first, "a.tei"), "FIRST")
	writeTestFile(t, filepath.Join(second, "a.tei"), "SECOND")
	alias := filepath.Join(base, "alias")
	makeDirAlias(t, alias, first)
	reader, err := readWorkspaceSource(alias)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	makeDirAlias(t, alias, second)
	if got, err := reader("a.tei"); err == nil {
		t.Fatalf("retargeted root alias was read: %q", got)
	}
}

// The scenario of #1017: the Source Index is built, then a source directory is
// replaced by an alias to an outside directory holding a same-named file. The
// Resolver must fail explicitly and no outside byte may be returned.
func TestSourceResolverFailsClosedWhenSourceDirectoryIsRetargetedOutsideAfterIndex(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "tei.workspace.yaml"), "workspace: trace\nsources:\n  - semantic\n")
	writeTestFile(t, filepath.Join(root, "semantic", "one.tei"), pinnedReadTei)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := readWorkspaceSource(root)
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	writeTestFile(t, filepath.Join(outside, "one.tei"), "OUTSIDE-BYTES")
	if err := os.Rename(filepath.Join(root, "semantic"), filepath.Join(root, "semantic.orig")); err != nil {
		t.Fatal(err)
	}
	makeDirAlias(t, filepath.Join(root, "semantic"), outside)

	if got, err := reader("semantic/one.tei"); err == nil {
		t.Fatalf("reader returned %q after the source directory was retargeted outside", got)
	}
	var leaked []byte
	capture := func(p string) ([]byte, error) {
		data, err := reader(p)
		if err == nil {
			leaked = data
		}
		return data, err
	}
	if _, err := modelrefResolver(index, capture); err == nil {
		t.Fatal("Resolver accepted a Source Index whose source directory now resolves outside")
	}
	if strings.Contains(string(leaked), "OUTSIDE") {
		t.Fatalf("outside bytes reached the Resolver: %q", leaked)
	}
}

func TestTraceWorkspaceStillLocatesSourceThroughPinnedReader(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	reader, err := readWorkspaceSource(root)
	if err != nil {
		t.Fatal(err)
	}
	data, err := reader("semantic/sales/order.tei")
	if err != nil || len(data) == 0 {
		t.Fatalf("pinned read of a reference source: %d bytes, %v", len(data), err)
	}
	for _, bad := range []string{"../go.mod", "semantic/../../go.mod", "/etc/passwd", "semantic"} {
		if got, err := reader(bad); err == nil {
			t.Fatalf("unsafe or non-regular source %q returned %d bytes", bad, len(got))
		}
	}
}

func modelrefResolver(index sourceindex.Index, read func(string) ([]byte, error)) (*sourcebridge.Resolver, error) {
	return sourcebridge.NewResolver(modelref.Graph{Schema: "tei.modelref.v0"}, index, read)
}

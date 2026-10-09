package workspace_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/workspace"
)

func TestLoadedSemanticIndexRetainsParsedSourceAfterRewrite(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace:\n  sources:\n    - semantic\n"), 0600); err != nil {
		t.Fatal(err)
	}
	path := "semantic/order.tei"
	writeSource(t, root, path, []byte("namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n}\n"))
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	before, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}

	// This is no longer a valid semantic declaration. Source Index must
	// describe the bytes actually parsed, not the current filesystem.
	writeSource(t, root, path, []byte("replaced after Load"))
	after, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("source index reread modified semantic source: %v", err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("source index drifted from parsed bytes\nbefore=%#v\nafter=%#v", before, after)
	}
}

func TestLoadedNativeSQLIndexRetainsAuthoringBytesAfterRewrite(t *testing.T) {
	root := newQueryWorkspace(t, `query order_entry.recent_orders {
    datasource sales
    sql postgres file "recent_orders.sql"
}`, "recent_orders.sql", nativeSQLBody)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	before, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "applications", "order-entry", "queries", "recent_orders.sql")
	if err := os.WriteFile(path, []byte("SELECT something_else"), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("native SQL index differs from bytes used by Query\nbefore=%#v\nafter=%#v", before, after)
	}
	if got := loaded.World.Queries[0].Source; got != nativeSQLBody {
		t.Fatalf("parsed query source changed: %q", got)
	}
	fresh, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	freshIndex, err := workspace.BuildSourceIndex(root, fresh)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(before, freshIndex) {
		t.Fatal("fresh Load did not observe changed native SQL")
	}
}

func TestLoadedSourceSnapshotDoesNotFollowRetargetedSymlink(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace:\n  sources:\n    - applications\n"), 0600); err != nil {
		t.Fatal(err)
	}
	path := "applications/demo/application.tei"
	writeSource(t, root, path, []byte("application demo {}\n"))
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	before, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "external.tei")
	if err := os.WriteFile(outside, []byte("application external {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, filepath.FromSlash(path))
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, file); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	after, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("source index reread retargeted symlink: %v", err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("loaded Source Index followed a symlink changed after Load")
	}
}

func TestLoadedSnapshotRejectsDifferentWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace:\n  sources:\n    - applications\n"), 0600); err != nil {
		t.Fatal(err)
	}
	writeSource(t, root, "applications/demo/application.tei", []byte("application demo {}\n"))
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = workspace.BuildSourceIndex(t.TempDir(), loaded)
	if err == nil || !strings.Contains(err.Error(), "another Workspace") {
		t.Fatalf("snapshot was accepted for another Workspace: %v", err)
	}
}

func TestStandaloneResultContinuesToReadCurrentFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "native.txt")
	if err := os.WriteFile(path, []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	result := workspace.Result{ManagedFiles: []workspace.ManagedFile{{Path: "native.txt", SourceRoot: "custom"}}}
	before, err := workspace.BuildSourceIndex(root, result)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("second"), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := workspace.BuildSourceIndex(root, result)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Artifacts) != 1 || len(after.Artifacts) != 1 ||
		before.Artifacts[0].Fingerprint == after.Artifacts[0].Fingerprint {
		t.Fatal("standalone Result incorrectly reused cached bytes")
	}
}

func TestLoadedSnapshotAcceptsRelativeAbsoluteAndSymlinkedRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace:\\n  sources:\\n    - applications\\n"), 0600); err != nil {
		t.Fatal(err)
	}
	writeSource(t, root, "applications/demo/application.tei", []byte("application demo {}\\n"))

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(cwd, root)
	if err != nil {
		t.Skipf("no relative path to temporary Workspace: %v", err)
	}
	loaded, err := workspace.Load(relative)
	if err != nil {
		t.Fatalf("load with relative root: %v", err)
	}
	fromAbsolute, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("index with absolute root after relative Load: %v", err)
	}
	fromRelative, err := workspace.BuildSourceIndex(relative, loaded)
	if err != nil {
		t.Fatalf("index with relative root after relative Load: %v", err)
	}
	if !reflect.DeepEqual(fromAbsolute, fromRelative) {
		t.Fatal("relative and absolute root spellings produced different Source Index")
	}

	alias := filepath.Join(t.TempDir(), "workspace-alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("root symlink unavailable: %v", err)
	}
	fromAlias, err := workspace.BuildSourceIndex(alias, loaded)
	if err != nil {
		t.Fatalf("same physical root reached through symlink rejected: %v", err)
	}
	if !reflect.DeepEqual(fromAbsolute, fromAlias) {
		t.Fatal("Workspace root symlink changed Source Index identity")
	}
}

func TestStandaloneSourceIndexAcceptsRelativeRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "native.md"), []byte("portable native asset"), 0600); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(cwd, root)
	if err != nil {
		t.Skipf("no relative path to temporary Workspace: %v", err)
	}
	result := workspace.Result{ManagedFiles: []workspace.ManagedFile{{Path: "native.md", SourceRoot: "assets"}}}
	absoluteIndex, err := workspace.BuildSourceIndex(root, result)
	if err != nil {
		t.Fatal(err)
	}
	relativeIndex, err := workspace.BuildSourceIndex(relative, result)
	if err != nil {
		t.Fatalf("standalone Source Index rejected relative Workspace root: %v", err)
	}
	if !reflect.DeepEqual(absoluteIndex, relativeIndex) {
		t.Fatal("standalone source fingerprints differ by Workspace root spelling")
	}
}

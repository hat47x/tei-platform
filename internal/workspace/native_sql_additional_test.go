package workspace_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestNativeSQLRejectsAbsolutePath(t *testing.T) {
	absolute := filepath.ToSlash(filepath.Join(t.TempDir(), "outside.sql"))
	declaration := "query order_entry.recent_orders {\n    datasource sales\n    sql postgres file \"" + absolute + "\"\n}\n"
	root := newQueryWorkspace(t, declaration, "", "")
	_, err := workspace.Load(root)
	if err == nil || !strings.Contains(err.Error(), "relative to the Query declaration") {
		t.Fatalf("error=%v", err)
	}
}

func TestNativeSQLContentChangeChangesMeaningAndFingerprint(t *testing.T) {
	root := newQueryWorkspace(t, `query order_entry.recent_orders {
    datasource sales
    sql postgres file "recent_orders.sql"
}`, "recent_orders.sql", nativeSQLBody)

	before, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	beforeIndex, err := workspace.BuildSourceIndex(root, before)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(root, "applications", "order-entry", "queries", "recent_orders.sql")
	changedSQL := nativeSQLBody + "\nORDER BY ordered_at DESC"
	if err := os.WriteFile(path, []byte(changedSQL), 0o644); err != nil {
		t.Fatal(err)
	}

	after, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	afterIndex, err := workspace.BuildSourceIndex(root, after)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(before.World, after.World) {
		t.Fatal("changing authoritative SQL did not change normalized World")
	}
	if after.World.Queries[0].Source != changedSQL {
		t.Fatalf("query source=%q", after.World.Queries[0].Source)
	}

	subject := sourceindex.Subject{Kind: "query", ID: "order_entry.recent_orders"}
	beforeRef := referenceByRole(t, beforeIndex.ReferencesFor(subject), sourceindex.RoleAuthoring)
	afterRef := referenceByRole(t, afterIndex.ReferencesFor(subject), sourceindex.RoleAuthoring)
	beforeArtifact := artifactByPath(t, beforeIndex, beforeRef.ArtifactPath)
	afterArtifact := artifactByPath(t, afterIndex, afterRef.ArtifactPath)
	if beforeArtifact.Fingerprint == afterArtifact.Fingerprint {
		t.Fatal("changing authoritative SQL did not change source fingerprint")
	}
}

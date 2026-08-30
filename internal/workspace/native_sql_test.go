package workspace_test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

const nativeSQLBody = "SELECT order_id\nFROM public.orders\nWHERE ordered_at >= :from"

func TestNativeSQLLoadsToSameWorldAsEmbeddedSQL(t *testing.T) {
	embeddedRoot := newQueryWorkspace(t, `query order_entry.recent_orders {
    datasource sales
    sql postgres {
        SELECT order_id
FROM public.orders
WHERE ordered_at >= :from
    }
}`, "", "")
	nativeRoot := newQueryWorkspace(t, `query order_entry.recent_orders {
    datasource sales
    sql postgres file "recent_orders.sql"
}`, "recent_orders.sql", nativeSQLBody)

	embedded, err := workspace.Load(embeddedRoot)
	if err != nil {
		t.Fatal(err)
	}
	native, err := workspace.Load(nativeRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(embedded.World, native.World) {
		t.Fatalf("native SQL changed normalized World\nembedded=%#v\nnative=%#v", embedded.World, native.World)
	}
	if len(native.World.Queries) != 1 || native.World.Queries[0].Source != nativeSQLBody {
		t.Fatalf("native query=%#v", native.World.Queries)
	}
}

func TestNativeSQLAddsAuthoringSourceWithoutReplacingQueryDefinition(t *testing.T) {
	root := newQueryWorkspace(t, `query order_entry.recent_orders {
    datasource sales
    sql postgres file "recent_orders.sql"
}`, "recent_orders.sql", nativeSQLBody)

	result, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := workspace.BuildSourceIndex(root, result)
	if err != nil {
		t.Fatal(err)
	}
	refs := idx.ReferencesFor(sourceindex.Subject{Kind: "query", ID: "order_entry.recent_orders"})
	if len(refs) != 2 {
		t.Fatalf("query source references=%#v", refs)
	}
	roles := map[sourceindex.Role]sourceindex.Reference{}
	for _, ref := range refs {
		roles[ref.Role] = ref
	}
	definition, ok := roles[sourceindex.RoleDefinition]
	if !ok || definition.ArtifactPath != "applications/order-entry/queries/recent_orders.tei" {
		t.Fatalf("definition=%#v", definition)
	}
	authoring, ok := roles[sourceindex.RoleAuthoring]
	if !ok || authoring.ArtifactPath != "applications/order-entry/queries/recent_orders.sql" {
		t.Fatalf("authoring=%#v", authoring)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(authoring.ArtifactPath)))
	if err != nil {
		t.Fatal(err)
	}
	if authoring.Range.Start.Offset != 0 || authoring.Range.End.Offset != len(content) {
		t.Fatalf("SQL authoring range=%+v content length=%d", authoring.Range, len(content))
	}
}

func TestNativeSQLMoveChangesOnlyAuthoringLocator(t *testing.T) {
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

	queryDir := filepath.Join(root, "applications", "order-entry", "queries")
	if err := os.MkdirAll(filepath.Join(queryDir, "sql"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(queryDir, "recent_orders.sql"), filepath.Join(queryDir, "sql", "recent_orders.sql")); err != nil {
		t.Fatal(err)
	}
	declaration := `query order_entry.recent_orders {
    datasource sales
    sql postgres file "sql/recent_orders.sql"
}`
	if err := os.WriteFile(filepath.Join(queryDir, "recent_orders.tei"), []byte(declaration), 0o644); err != nil {
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
	if !reflect.DeepEqual(before.World, after.World) {
		t.Fatal("moving native SQL changed normalized World")
	}

	subject := sourceindex.Subject{Kind: "query", ID: "order_entry.recent_orders"}
	beforeAuthoring := referenceByRole(t, beforeIndex.ReferencesFor(subject), sourceindex.RoleAuthoring)
	afterAuthoring := referenceByRole(t, afterIndex.ReferencesFor(subject), sourceindex.RoleAuthoring)
	if beforeAuthoring.ArtifactPath != "applications/order-entry/queries/recent_orders.sql" {
		t.Fatalf("before authoring path=%q", beforeAuthoring.ArtifactPath)
	}
	if afterAuthoring.ArtifactPath != "applications/order-entry/queries/sql/recent_orders.sql" {
		t.Fatalf("after authoring path=%q", afterAuthoring.ArtifactPath)
	}
	beforeArtifact := nativeSQLArtifactByPath(t, beforeIndex, beforeAuthoring.ArtifactPath)
	afterArtifact := nativeSQLArtifactByPath(t, afterIndex, afterAuthoring.ArtifactPath)
	if beforeArtifact.Fingerprint != afterArtifact.Fingerprint {
		t.Fatalf("same SQL content changed fingerprint: %s != %s", beforeArtifact.Fingerprint, afterArtifact.Fingerprint)
	}
}

func TestNativeSQLRejectsUnsafeOrInvalidPaths(t *testing.T) {
	tests := []struct {
		name      string
		ref       string
		prepare   func(t *testing.T, root, queryDir string)
		wantError string
	}{
		{name: "workspace escape", ref: "../../../../outside.sql", wantError: "escapes the Workspace"},
		{name: "missing", ref: "missing.sql", wantError: "no such file"},
		{
			name: "directory",
			ref:  "directory.sql",
			prepare: func(t *testing.T, root, queryDir string) {
				t.Helper()
				if err := os.Mkdir(filepath.Join(queryDir, "directory.sql"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantError: "regular file",
		},
		{
			name: "non SQL extension",
			ref:  "query.txt",
			prepare: func(t *testing.T, root, queryDir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(queryDir, "query.txt"), []byte(nativeSQLBody), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantError: ".sql file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			declaration := "query order_entry.recent_orders {\n    datasource sales\n    sql postgres file \"" + tt.ref + "\"\n}\n"
			root := newQueryWorkspace(t, declaration, "", "")
			queryDir := filepath.Join(root, "applications", "order-entry", "queries")
			if tt.prepare != nil {
				tt.prepare(t, root, queryDir)
			}
			_, err := workspace.Load(root)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.wantError)) {
				t.Fatalf("error=%v want substring %q", err, tt.wantError)
			}
		})
	}
}

func TestNativeSQLRejectsSymlinkThatEscapesWorkspace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	root := newQueryWorkspace(t, `query order_entry.recent_orders {
    datasource sales
    sql postgres file "outside.sql"
}`, "", "")
	outsideDir := t.TempDir()
	outside := filepath.Join(outsideDir, "outside.sql")
	if err := os.WriteFile(outside, []byte(nativeSQLBody), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "applications", "order-entry", "queries", "outside.sql")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	_, err := workspace.Load(root)
	if err == nil || !strings.Contains(err.Error(), "resolves outside the Workspace") {
		t.Fatalf("error=%v", err)
	}
}

func newQueryWorkspace(t *testing.T, queryDeclaration, sqlFile, sqlContent string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace:\n  sources:\n    - applications\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	unit := filepath.Join(root, "applications", "order-entry")
	queryDir := filepath.Join(unit, "queries")
	if err := os.MkdirAll(queryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unit, "application.tei"), []byte("application order_entry {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(queryDir, "recent_orders.tei"), []byte(queryDeclaration), 0o644); err != nil {
		t.Fatal(err)
	}
	if sqlFile != "" {
		full := filepath.Join(queryDir, filepath.FromSlash(sqlFile))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(sqlContent), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func referenceByRole(t *testing.T, refs []sourceindex.Reference, role sourceindex.Role) sourceindex.Reference {
	t.Helper()
	for _, ref := range refs {
		if ref.Role == role {
			return ref
		}
	}
	t.Fatalf("reference role %s not found in %#v", role, refs)
	return sourceindex.Reference{}
}

func nativeSQLArtifactByPath(t *testing.T, idx sourceindex.Index, path string) sourceindex.Artifact {
	t.Helper()
	for _, artifact := range idx.Artifacts {
		if artifact.Path == path {
			return artifact
		}
	}
	t.Fatalf("artifact %s not found", path)
	return sourceindex.Artifact{}
}

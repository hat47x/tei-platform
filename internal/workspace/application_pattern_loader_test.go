package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

func TestWorkspaceLoadsApplicationPatternsFromIndependentFiles(t *testing.T) {
	root := t.TempDir()
	writePatternWorkspaceFile(t, root, "tei.workspace.yaml", `workspace: pattern_loader_test
sources:
  - applications
`)
	writePatternWorkspaceFile(t, root, "applications/sample/application.tei", `application sample_app {
}`)

	// ファイル名とCanonical IDの順序を意図的に逆転させる。
	// Normalize後の順序がfile pathではなくPattern IDで決まることを確認するためである。
	writePatternWorkspaceFile(t, root, "applications/sample/patterns/a-second.tei", `pattern sample_app.z_orders browse {
    data sales.order
    query sample_app.z_orders_query
}`)
	writePatternWorkspaceFile(t, root, "applications/sample/patterns/z-first.tei", `pattern sample_app.a_orders browse {
    data sales.order
    query sample_app.a_orders_query
    datasource sales
}`)

	got, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load workspace: %v", err)
	}
	if len(got.World.Applications) != 1 {
		t.Fatalf("applications=%#v", got.World.Applications)
	}
	app := got.World.Applications[0]
	if app.ID != "sample_app" {
		t.Fatalf("application id=%q", app.ID)
	}
	if len(app.Patterns) != 2 {
		t.Fatalf("patterns=%#v", app.Patterns)
	}
	if app.Patterns[0].ID != "sample_app.a_orders" || app.Patterns[1].ID != "sample_app.z_orders" {
		t.Fatalf("patterns are not normalized by Canonical ID: %#v", app.Patterns)
	}
	if app.Patterns[0].Datasource != "sales" {
		t.Fatalf("explicit datasource was not preserved: %#v", app.Patterns[0])
	}

	cases := map[string]string{
		"application-pattern:sample_app.a_orders": "applications/sample/patterns/z-first.tei",
		"application-pattern:sample_app.z_orders": "applications/sample/patterns/a-second.tei",
	}
	for key, wantFile := range cases {
		loc, ok := got.Sources[key]
		if !ok {
			t.Fatalf("missing source mapping %s", key)
		}
		if loc.File != wantFile {
			t.Fatalf("%s file=%q want=%q", key, loc.File, wantFile)
		}
	}
}

func writePatternWorkspaceFile(t *testing.T, root, relativePath, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

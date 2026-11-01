package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/validation"
)

func TestDuplicateSemanticSourceCandidates(t *testing.T) {
	for _, tc := range []struct {
		name, first, second string
		wantPaths           []string
		wantLines           []int
	}{
		{"same-file", "namespace intake\ndata Request {\n attribute status type std.code\n attribute status type std.code\n}\n", "", []string{"semantic/a.tei", "semantic/a.tei"}, []int{3, 4}},
		{"different-files", "namespace intake\ndata Request {\n attribute status type std.code\n}\n", "namespace intake\ndata Request {\n attribute status type std.code\n}\n", []string{"semantic/a.tei", "semantic/b.tei"}, []int{3, 3}},
		{"different-kinds", "namespace intake\ntype Request from std.code\n", "namespace intake\ndata Request {}\n", []string{"semantic/a.tei", "semantic/b.tei"}, []int{2, 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "semantic"), 0755); err != nil {
				t.Fatal(err)
			}
			for path, text := range map[string]string{"tei.workspace.yaml": "workspace: test\nsources:\n  - semantic\n", "semantic/a.tei": tc.first, "semantic/b.tei": tc.second} {
				if text == "" {
					continue
				}
				if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), []byte(text), 0644); err != nil {
					t.Fatal(err)
				}
			}
			loaded, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}
			index, err := BuildSourceIndex(root, loaded)
			if err != nil {
				t.Fatal(err)
			}
			before, err := json.Marshal(index)
			if err != nil {
				t.Fatal(err)
			}
			diagnostics := validation.Validate(loaded.World)
			result := ProjectDiagnosticsWithLoadedDuplicates(diagnostics, index, loaded)
			var found bool
			for _, item := range result {
				if item.Diagnostic.Code != "DUPLICATE_SEMANTIC_ELEMENT" || item.Diagnostic.Subject == nil {
					continue
				}
				if tc.name == "different-files" && item.Diagnostic.Subject.Kind != "semantic-attribute" {
					continue
				}
				found = true
				if len(item.Sources) != 2 {
					t.Fatalf("sources=%+v", item.Sources)
				}
				for i, ref := range item.Sources {
					if ref.ArtifactPath != tc.wantPaths[i] || ref.Range.Start.Line != tc.wantLines[i] || ref.Role != sourceindex.RoleDefinition {
						t.Fatalf("source[%d]=%+v", i, ref)
					}
				}
			}
			after, err := json.Marshal(index)
			if err != nil || !found || string(before) != string(after) {
				t.Fatal("missing duplicate or Source Index changed")
			}
			// Missing snapshots and imports must not produce partial candidates.
			loaded.Imports = append(loaded.Imports, ResolvedImport{})
			if got := ProjectDiagnosticsWithLoadedDuplicates(diagnostics, index, loaded); !reflect.DeepEqual(got, ProjectDiagnostics(diagnostics, index)) {
				t.Fatal("imported sources were treated as complete")
			}
			loaded.Imports = nil
			delete(loaded.sourceSnapshot, "semantic/a.tei")
			if got := ProjectDiagnosticsWithLoadedDuplicates(diagnostics, index, loaded); !reflect.DeepEqual(got, ProjectDiagnostics(diagnostics, index)) {
				t.Fatal("missing captured bytes were treated as complete")
			}
		})
	}
}

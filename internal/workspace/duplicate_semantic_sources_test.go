package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/subject"
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
			// An unrelated import must not erase local declaration candidates
			// and must not be presented as a physical position.
			loaded.Imports = append(loaded.Imports, ResolvedImport{Alias: "other", Path: "../other", Namespace: "other"})
			if got := ProjectDiagnosticsWithLoadedDuplicates(diagnostics, index, loaded); !reflect.DeepEqual(got, result) {
				t.Fatalf("unrelated import changed local candidates: %+v", got)
			}
			loaded.Imports = nil
			delete(loaded.sourceSnapshot, "semantic/a.tei")
			if got := ProjectDiagnosticsWithLoadedDuplicates(diagnostics, index, loaded); !reflect.DeepEqual(got, ProjectDiagnostics(diagnostics, index)) {
				t.Fatal("missing captured bytes were treated as complete")
			}
		})
	}
}

func writeDuplicateFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// An unrelated import must not erase local-local duplicate candidates. An
// import whose namespace is already owned is rejected by Load before
// validation, so no imported declaration position is invented here.
func TestDuplicateSemanticSourcesSurviveUnrelatedImport(t *testing.T) {
	base := t.TempDir()
	consumer := filepath.Join(base, "consumer")
	provider := filepath.Join(base, "provider")
	writeDuplicateFixture(t, provider, "tei.workspace.yaml", "workspace: provider\nsources:\n  - semantic\n")
	writeDuplicateFixture(t, provider, "semantic/external.tei", "namespace external\ntype ExternalId from std.identifier\n")
	writeDuplicateFixture(t, consumer, "tei.workspace.yaml", "workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: external\n    path: ../provider\n    namespace: external\n")
	writeDuplicateFixture(t, consumer, "semantic/request.tei", "namespace intake\ndata Request {\n    attribute status type std.code\n    attribute status type std.code\n}\n")

	loaded, err := Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	index, err := BuildSourceIndex(consumer, loaded)
	if err != nil {
		t.Fatal(err)
	}
	result := ProjectDiagnosticsWithLoadedDuplicates(validation.Validate(loaded.World), index, loaded)
	for _, item := range result {
		if item.Diagnostic.Code != "DUPLICATE_SEMANTIC_ELEMENT" || item.Diagnostic.Subject == nil || item.Diagnostic.Subject.ID != "intake.request.status" {
			continue
		}
		if len(item.Sources) != 2 || len(item.UnresolvedSourceOrigins) != 0 {
			t.Fatalf("unrelated import erased candidates: %+v", item)
		}
		if item.Sources[0].ArtifactPath != "semantic/request.tei" || item.Sources[0].Range.Start.Line != 3 ||
			item.Sources[1].Range.Start.Line != 4 {
			t.Fatalf("local candidate positions changed: %+v", item.Sources)
		}
		return
	}
	t.Fatal("duplicate diagnostic missing")
}

// The prelude is an in-process value, not a Source file. Its declaration
// position is reported as unresolved rather than filled with the local path.
func TestDuplicateSemanticSourcesMarkStandardPreludeUnknown(t *testing.T) {
	root := t.TempDir()
	writeDuplicateFixture(t, root, "tei.workspace.yaml", "workspace: prelude\nsources:\n  - semantic\n")
	writeDuplicateFixture(t, root, "semantic/std.tei", "namespace std\ntype code from std.text\n")

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	index, err := BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	result := ProjectDiagnosticsWithLoadedDuplicates(validation.Validate(loaded.World), index, loaded)
	for _, item := range result {
		if item.Diagnostic.Code != "DUPLICATE_SEMANTIC_ELEMENT" || item.Diagnostic.Subject == nil || item.Diagnostic.Subject.ID != "std.code" {
			continue
		}
		if len(item.Sources) != 1 || item.Sources[0].ArtifactPath != "semantic/std.tei" || item.Sources[0].Range.Start.Line != 2 {
			t.Fatalf("local prelude duplicate candidate: %+v", item.Sources)
		}
		if !reflect.DeepEqual(item.UnresolvedSourceOrigins, []string{"standard"}) {
			t.Fatalf("standard declaration was not marked unknown: %+v", item.UnresolvedSourceOrigins)
		}
		return
	}
	t.Fatal("prelude duplicate diagnostic missing")
}

// A diagnostic can outlive the snapshot it was computed from. When only one
// declaration can be located, the projection must not present it as the
// complete duplicate set.
func TestDuplicateSemanticSourcesMarkUnlocatableDeclarationUnknown(t *testing.T) {
	root := t.TempDir()
	writeDuplicateFixture(t, root, "tei.workspace.yaml", "workspace: single\nsources:\n  - semantic\n")
	writeDuplicateFixture(t, root, "semantic/request.tei", "namespace intake\ndata Request {\n    attribute status type std.code\n}\n")
	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	index, err := BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	target := subject.Subject{Kind: "semantic-attribute", ID: "intake.request.status"}
	diagnostics := []validation.Diagnostic{{
		Code: "DUPLICATE_SEMANTIC_ELEMENT", Severity: validation.Error,
		Message: "semantic element intake.request.status is declared more than once as attribute",
		Subject: &target,
	}}
	result := ProjectDiagnosticsWithLoadedDuplicates(diagnostics, index, loaded)
	if len(result) != 1 {
		t.Fatalf("diagnostics=%+v", result)
	}
	if len(result[0].Sources) != 1 || result[0].Sources[0].Range.Start.Line != 3 {
		t.Fatalf("located declaration missing: %+v", result[0].Sources)
	}
	if !reflect.DeepEqual(result[0].UnresolvedSourceOrigins, []string{"unknown"}) {
		t.Fatalf("unlocatable declaration was not marked: %+v", result[0].UnresolvedSourceOrigins)
	}
}

package operations_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

func writeValidationWorkspace(t *testing.T, extra string) string {
	t.Helper()
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", "workspace: scn\nformatVersion: 1\nsources:\n  - semantic\n")
	write("semantic/model.tei", "namespace scn\n\ndata RuleCandidate {\n    attribute sourceVersion type std.text\n}\n")
	if extra != "" {
		write("semantic/extra.tei", extra)
	}
	return root
}

// TestCatalogReportsTheValidationVerdict pins #3674: a caller that reads only
// the catalogue must be able to tell whether the Workspace passes validate. The
// catalogue still answers for an unsound Workspace (it is a navigation
// projection), so ok alone never carried that fact.
func TestCatalogReportsTheValidationVerdict(t *testing.T) {
	cases := []struct {
		name         string
		extra        string
		wantOK       bool
		wantErrors   int
		wantWarnings int
	}{
		{"sound workspace", "", true, 0, 0},
		{
			name:       "unknown attribute type",
			extra:      "namespace scn\n\ndata X {\n    attribute a type scn.nothere\n}\n",
			wantOK:     false,
			wantErrors: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := writeValidationWorkspace(t, tc.extra)
			catalog, err := operations.CatalogWorkspace(root, "")
			if err != nil {
				t.Fatalf("catalog: %v", err)
			}
			if catalog.Validation.OK != tc.wantOK {
				t.Fatalf("validation.ok = %v, want %v", catalog.Validation.OK, tc.wantOK)
			}
			if catalog.Validation.ErrorCount != tc.wantErrors {
				t.Fatalf("errorCount = %d, want %d", catalog.Validation.ErrorCount, tc.wantErrors)
			}
			if catalog.Validation.WarningCount != tc.wantWarnings {
				t.Fatalf("warningCount = %d, want %d", catalog.Validation.WarningCount, tc.wantWarnings)
			}
		})
	}
}

// The catalogue's verdict must agree with ValidateWorkspace on the same
// Workspace, because both apply the same rule. Two independent opinions about
// one model would be worse than none.
func TestCatalogValidationAgreesWithValidate(t *testing.T) {
	for _, extra := range []string{
		"",
		"namespace scn\n\ndata X {\n    attribute a type scn.nothere\n}\n",
	} {
		root := writeValidationWorkspace(t, extra)
		catalog, err := operations.CatalogWorkspace(root, "")
		if err != nil {
			t.Fatalf("catalog: %v", err)
		}
		validation, err := operations.ValidateWorkspace(root)
		if err != nil {
			t.Fatalf("validate: %v", err)
		}
		wantOK := !validation.HasErrors()
		if catalog.Validation.OK != wantOK {
			t.Fatalf("catalog ok=%v, validate errors=%v", catalog.Validation.OK, validation.HasErrors())
		}
		errors := 0
		warnings := 0
		for _, d := range validation.Diagnostics {
			if string(d.Diagnostic.Severity) == "error" {
				errors++
				continue
			}
			warnings++
		}
		for _, f := range validation.SourceRootFindings {
			switch f.Severity {
			case "error":
				errors++
			case "warning":
				warnings++
			}
		}
		if catalog.Validation.ErrorCount != errors || catalog.Validation.WarningCount != warnings {
			t.Fatalf("counts disagree: catalog=%+v validate errors=%d warnings=%d",
				catalog.Validation, errors, warnings)
		}
	}
}

// The catalogue still succeeds for an unsound Workspace: the verdict is a field,
// not a failure, so an editor can keep browsing. This fixes the behaviour the
// issue asked to make explicit.
func TestCatalogStillAnswersForAnUnsoundWorkspace(t *testing.T) {
	root := writeValidationWorkspace(t, "namespace scn\n\ndata X {\n    attribute a type scn.nothere\n}\n")
	catalog, err := operations.CatalogWorkspace(root, "")
	if err != nil {
		t.Fatalf("catalog refused an unsound Workspace: %v", err)
	}
	if len(catalog.Entries) == 0 {
		t.Fatal("catalog returned no entries for an unsound Workspace")
	}
	if catalog.Validation.OK {
		t.Fatal("an unsound Workspace was reported as sound")
	}
}

// TestUnsoundWorkspaceCatalogInspectAndTraceKeepValidationDistinct covers the
// cross-view gap in MV-01. Reading a canonical Source and its references must
// remain possible when semantic validation fails, but none of those successes
// changes the validation verdict. The three operations load independently:
// agreement on fingerprints does not prove an atomic multi-operation snapshot.
func TestUnsoundWorkspaceCatalogInspectAndTraceKeepValidationDistinct(t *testing.T) {
	root := writeValidationWorkspace(t, "namespace scn\n\ndata X {\n    attribute a type scn.nothere\n}\n")
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "scn.X"}

	catalog, err := operations.CatalogWorkspace(root, "")
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	validation, err := operations.ValidateWorkspace(root)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if catalog.Validation.OK || catalog.Validation.ErrorCount == 0 || !validation.HasErrors() {
		t.Fatalf("unsound Workspace must remain unsound: catalog=%+v validate=%+v", catalog.Validation, validation)
	}
	if catalog.Effect != "read-only" || catalog.WorldFingerprint == "" ||
		catalog.SourceIndexFingerprint == "" || catalog.Workspace == nil ||
		validation.Workspace == nil || catalog.WorldFingerprint != validation.WorldFingerprint ||
		catalog.SourceIndexFingerprint != validation.SourceIndexFingerprint ||
		!reflect.DeepEqual(*catalog.Workspace, *validation.Workspace) {
		t.Fatal("catalog and validate do not agree on their separately observed Workspace")
	}

	matches := 0
	for _, entry := range catalog.Entries {
		if entry.Subject != subject {
			continue
		}
		matches++
		if entry.Ambiguous || entry.Prelude || entry.SourceStatus != "indexed" {
			t.Fatalf("unsound canonical Subject must remain navigable, not accepted: %+v", entry)
		}
	}
	if matches != 1 {
		t.Fatalf("expected one exact canonical Subject, got %d", matches)
	}

	inspected, err := operations.InspectSubject(root, subject)
	if err != nil {
		t.Fatalf("inspect unsound canonical Subject: %v", err)
	}
	if inspected.Subject != subject || inspected.Effect != "read-only" ||
		inspected.SourceStatus != "indexed" || len(inspected.Sources.References) == 0 ||
		inspected.Workspace == nil || !reflect.DeepEqual(*catalog.Workspace, *inspected.Workspace) ||
		inspected.WorldFingerprint != catalog.WorldFingerprint ||
		inspected.SourceIndexFingerprint != catalog.SourceIndexFingerprint {
		t.Fatalf("inspection is not bound to the same observed canonical Subject: %+v", inspected)
	}

	traced, err := operations.TraceSubjectSources(root, subject)
	if err != nil {
		t.Fatalf("trace unsound canonical Subject: %v", err)
	}
	if traced.Subject != subject ||
		!reflect.DeepEqual(traced.References, inspected.Sources.References) ||
		!reflect.DeepEqual(traced.Artifacts, inspected.Sources.Artifacts) {
		t.Fatalf("trace/inspect Source references disagree for %v", subject)
	}
	// Inspect and Source Trace do not carry a validation grant. In particular,
	// successful Source navigation does not turn the earlier validation NG into OK.
	if catalog.Validation.OK {
		t.Fatal("Source navigation must not upgrade failed validation")
	}
}

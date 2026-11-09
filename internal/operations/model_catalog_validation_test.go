package operations_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/operations"
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

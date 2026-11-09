package operations_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/operations"
)

func writeCatalogValidationWorkspace(t *testing.T, model string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: catalog_validation\nformatVersion: 1\nsources:\n  - semantic\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "model.tei"), []byte(model), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// The catalog is a projection, so it still succeeds for a Workspace that does
// not validate. What a caller reading only the catalog must not do is mistake
// that success for a healthy declaration list, so the result carries the same
// validation verdict `validate` reports (#3674).
func TestCatalogReportsValidationErrorsWithoutFailing(t *testing.T) {
	root := writeCatalogValidationWorkspace(t, `namespace scn

data X {
    attribute a type scn.nothere
}
`)
	catalog, err := operations.CatalogWorkspace(root, "")
	if err != nil {
		t.Fatalf("the catalog refused a readable Workspace: %v", err)
	}
	if catalog.Validation.OK {
		t.Fatal("a Workspace with an unknown attribute type was reported as valid")
	}
	if catalog.Validation.ErrorCount == 0 {
		t.Fatalf("the error count is zero beside ok=false: %+v", catalog.Validation)
	}
	// The same Workspace must fail validate, otherwise this test would pin a
	// divergence instead of the agreement it is meant to prove.
	validation, err := operations.ValidateWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if !validation.HasErrors() {
		t.Fatal("validate accepted the Workspace this test expects to be invalid")
	}
}

// A valid Workspace reports ok with no error count, so a caller can tell the
// two states apart without guessing.
func TestCatalogReportsACleanWorkspaceAsValid(t *testing.T) {
	root := writeCatalogValidationWorkspace(t, `namespace scn

data RuleCandidate {
    attribute sourceVersion type std.text
}
`)
	catalog, err := operations.CatalogWorkspace(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.Validation.OK || catalog.Validation.ErrorCount != 0 {
		t.Fatalf("a valid Workspace was not reported as valid: %+v", catalog.Validation)
	}
	validation, err := operations.ValidateWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if validation.HasErrors() {
		t.Fatalf("validate and the catalog disagree for a valid Workspace: %v", validation.Diagnostics)
	}
}

// A filtered read keeps the same verdict: the namespace filter changes which
// entries are listed, not whether the Workspace validates.
func TestCatalogValidationIgnoresTheNamespaceFilter(t *testing.T) {
	root := writeCatalogValidationWorkspace(t, `namespace scn

data X {
    attribute a type scn.nothere
}
`)
	all, err := operations.CatalogWorkspace(root, "")
	if err != nil {
		t.Fatal(err)
	}
	filtered, err := operations.CatalogWorkspace(root, "scn")
	if err != nil {
		t.Fatal(err)
	}
	if filtered.NamespaceMatchKind != "namespace" {
		t.Fatalf("the filter did not select the namespace: %+v", filtered)
	}
	if filtered.Validation != all.Validation {
		t.Fatalf("the namespace filter changed the validation verdict: %+v vs %+v", filtered.Validation, all.Validation)
	}
}

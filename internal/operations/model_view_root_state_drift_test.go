package operations_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// A declared unknown root is a validation error while absent and a warning
// when present. Empty directories do not change the modeled Semantic Source.
// Fingerprint equality therefore cannot stand in for a fresh validation read.
func TestModelViewRootAvailabilityChangesValidationWithoutStructuralDrift(t *testing.T) {
	root := writeValidationWorkspace(t,
		"namespace scn\n\ndata X {\n    attribute a type std.text\n}\n")
	const manifest = "workspace: scn\nformatVersion: 1\nsources:\n  - semantic\n  - future\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "scn.x"}

	type snapshot struct {
		catalog   operations.ModelCatalog
		modelMap  operations.ModelMap
		location  operations.ModelLocation
		validated operations.WorkspaceValidation
	}
	read := func() snapshot {
		t.Helper()
		catalog, err := operations.CatalogWorkspace(root, "")
		if err != nil {
			t.Fatalf("catalog: %v", err)
		}
		modelMap, err := operations.MapWorkspace(root)
		if err != nil {
			t.Fatalf("map: %v", err)
		}
		location, err := operations.LocateWorkspace(root, subject, 1)
		if err != nil {
			t.Fatalf("locate: %v", err)
		}
		validated, err := operations.ValidateWorkspace(root)
		if err != nil {
			t.Fatalf("validate: %v", err)
		}
		if catalog.Validation != modelMap.Catalog.Validation ||
			catalog.Validation != location.Validation ||
			catalog.Validation.OK == validated.HasErrors() ||
			modelMap.Coverage != "partial-projection" ||
			location.Effect != "read-only" ||
			location.Subject.Subject != subject {
			t.Fatalf("cross-view qualification differs: catalog=%+v map=%+v locate=%+v typed=%+v",
				catalog.Validation, modelMap.Catalog.Validation, location.Validation, validated)
		}
		return snapshot{catalog, modelMap, location, validated}
	}

	missing := read()
	if missing.catalog.Validation.OK || missing.catalog.Validation.ErrorCount != 1 ||
		missing.catalog.Validation.WarningCount != 0 || !missing.validated.HasErrors() {
		t.Fatalf("missing declared root must invalidate the model: %+v", missing.catalog.Validation)
	}
	if err := os.Mkdir(filepath.Join(root, "future"), 0o755); err != nil {
		t.Fatal(err)
	}

	present := read()
	if !present.catalog.Validation.OK || present.catalog.Validation.ErrorCount != 0 ||
		present.catalog.Validation.WarningCount != 1 || present.validated.HasErrors() {
		t.Fatalf("present unknown root must be warning-only: %+v", present.catalog.Validation)
	}
	if missing.catalog.WorldFingerprint != present.catalog.WorldFingerprint ||
		missing.catalog.SourceIndexFingerprint != present.catalog.SourceIndexFingerprint ||
		missing.catalog.ModelRevision != present.catalog.ModelRevision ||
		!reflect.DeepEqual(missing.catalog.Workspace, present.catalog.Workspace) {
		t.Fatalf("empty unknown root changed the model identity instead of only the validation finding: before=%+v after=%+v",
			missing.catalog, present.catalog)
	}
	if !reflect.DeepEqual(missing.modelMap.Links, present.modelMap.Links) ||
		!reflect.DeepEqual(missing.location.Subject, present.location.Subject) {
		t.Fatal("read-only Map/Locate structure changed without a modeled Source change")
	}
	// The same fingerprint can accompany two different validation verdicts.
	// This is not a stale-DOM test or a current authority/permission grant.
}

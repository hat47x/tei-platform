package documentdraft

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// These expectations are written from the source model by hand. They are not
// derived from GenerateCRUDTable or ValidateModel, so an implementation that
// changes both sides agrees with nothing here.

// writeDiagramModel writes one raw three-element model and returns its path.
// The model carries no stored diagnostics, which is the normal case for a model
// file: any finding must be computed, not echoed.
func writeDiagramModel(t *testing.T, name string, model map[string]any) string {
	t.Helper()
	if _, ok := model["schemaVersion"]; !ok {
		model["schemaVersion"] = "tei.three-element/v0"
	}
	if _, ok := model["effect"]; !ok {
		model["effect"] = "read-only"
	}
	encoded, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// A CRUD contradiction must be visible in the diagram itself. The reader sees a
// table of declared operations; if a Function also declares different
// operations through Function.Operations, the drawn cell must say so instead of
// presenting the declaration as the settled set.
func TestThreeElementDiagramMarksContradictoryCRUDCell(t *testing.T) {
	// Function.Operations declares create; the explicit record declares delete.
	path := writeDiagramModel(t, "conflict.json", map[string]any{
		"modelId":      "conflict-model",
		"name":         "Conflict model",
		"dataElements": []map[string]any{{"id": "order", "name": "Order"}},
		"functionElements": []map[string]any{{
			"id": "submit", "name": "Submit",
			"operations": []map[string]any{{"dataId": "order", "operations": "C"}},
		}},
		"businessElements": []map[string]any{},
		"crudDefinitions": []map[string]any{{
			"id": "declared", "functionId": "submit", "dataId": "order",
			"delete": true, "source": "declared",
		}},
	})
	draft, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: path})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	diagram := draft.Occurrences[0].Content.Diagram
	if diagram == nil {
		t.Fatal("the diagram occurrence carries no content")
	}

	if len(diagram.CRUDs) != 1 {
		t.Fatalf("cruds=%+v want one declared cell", diagram.CRUDs)
	}
	cell := diagram.CRUDs[0]
	if cell.Operations != "D" {
		t.Fatalf("declared cell operations=%q want D", cell.Operations)
	}
	// The overridden derived intent must stay readable, and the cell must not
	// look settled.
	if cell.DerivedOperations != "C" {
		t.Fatalf("derivedOperations=%q want C (the intent the declaration overrode)",
			cell.DerivedOperations)
	}
	if !cell.ConflictingDeclarations {
		t.Fatalf("a contradiction was presented as a clean cell: %+v", cell)
	}

	if diagram.CRUDConflictStatus != "conflicts-require-review" {
		t.Fatalf("crudConflictStatus=%q want conflicts-require-review", diagram.CRUDConflictStatus)
	}
	if len(diagram.OpposingCRUDPairs) != 1 || diagram.OpposingCRUDPairs[0] != "submit/order" {
		t.Fatalf("opposingCrudPairs=%v want [submit/order]", diagram.OpposingCRUDPairs)
	}
}

// A model whose declarations agree must still be presented as usable. This is
// the positive control that keeps the new status from becoming always-on.
func TestThreeElementDiagramKeepsAgreeingCRUDClean(t *testing.T) {
	path := writeDiagramModel(t, "agree.json", map[string]any{
		"modelId":      "agree-model",
		"name":         "Agreeing model",
		"dataElements": []map[string]any{{"id": "order", "name": "Order"}},
		"functionElements": []map[string]any{{
			"id": "submit", "name": "Submit",
			"operations": []map[string]any{{"dataId": "order", "operations": "CR"}},
		}},
		"businessElements": []map[string]any{},
		"crudDefinitions": []map[string]any{{
			"id": "declared", "functionId": "submit", "dataId": "order",
			"create": true, "read": true, "source": "declared",
		}},
	})
	draft, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: path})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	diagram := draft.Occurrences[0].Content.Diagram
	if len(diagram.CRUDs) != 1 || diagram.CRUDs[0].Operations != "CR" {
		t.Fatalf("cruds=%+v want one CR cell", diagram.CRUDs)
	}
	if diagram.CRUDs[0].ConflictingDeclarations || diagram.CRUDs[0].DerivedOperations != "" {
		t.Fatalf("agreeing declarations were flagged: %+v", diagram.CRUDs[0])
	}
	if diagram.CRUDConflictStatus != "no-conflicts-detected" {
		t.Fatalf("crudConflictStatus=%q want no-conflicts-detected", diagram.CRUDConflictStatus)
	}
	if len(diagram.OpposingCRUDPairs) != 0 {
		t.Fatalf("opposingCrudPairs=%v want none", diagram.OpposingCRUDPairs)
	}
}

// A narrowing or widening of the same operations is not a contradiction, so the
// diagram must not force review. The overridden intent is still named, because
// a reader who sees only "R" would otherwise conclude the function cannot write.
func TestThreeElementDiagramNamesOverriddenIntentWithoutReview(t *testing.T) {
	path := writeDiagramModel(t, "narrow.json", map[string]any{
		"modelId":      "narrow-model",
		"name":         "Narrowing model",
		"dataElements": []map[string]any{{"id": "order", "name": "Order"}},
		"functionElements": []map[string]any{{
			"id": "submit", "name": "Submit",
			"operations": []map[string]any{{"dataId": "order", "operations": "CRUD"}},
		}},
		"businessElements": []map[string]any{},
		"crudDefinitions": []map[string]any{{
			"id": "declared", "functionId": "submit", "dataId": "order",
			"read": true, "source": "declared",
		}},
	})
	draft, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: path})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	diagram := draft.Occurrences[0].Content.Diagram
	cell := diagram.CRUDs[0]
	if cell.Operations != "R" || cell.DerivedOperations != "CRUD" {
		t.Fatalf("narrowing lost an intent: %+v", cell)
	}
	if cell.ConflictingDeclarations {
		t.Fatalf("a narrowing is not a contradiction: %+v", cell)
	}
	if diagram.CRUDConflictStatus != "no-conflicts-detected" {
		t.Fatalf("crudConflictStatus=%q want no-conflicts-detected", diagram.CRUDConflictStatus)
	}
}

// Diagnostics must be computed from the model, not echoed from a stored field.
// A model file that carries no diagnostics still has findings when its
// references cannot be resolved.
func TestThreeElementDiagramComputesDiagnosticsInsteadOfEchoingThem(t *testing.T) {
	path := writeDiagramModel(t, "unknown.json", map[string]any{
		"modelId":      "unknown-model",
		"name":         "Unknown reference model",
		"dataElements": []map[string]any{{"id": "order", "name": "Order"}},
		"functionElements": []map[string]any{{
			"id": "submit", "name": "Submit",
		}},
		"businessElements": []map[string]any{},
		// Names a Data the model does not declare: a structural error.
		"crudDefinitions": []map[string]any{{
			"id": "declared", "functionId": "submit", "dataId": "absent",
			"read": true, "source": "declared",
		}},
	})
	draft, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: path})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	diagram := draft.Occurrences[0].Content.Diagram
	found := false
	for _, finding := range diagram.Diagnostics {
		if finding.Code == "UNKNOWN_CRUD_DATA" && finding.Severity == "error" {
			found = true
		}
	}
	if !found {
		t.Fatalf("a model with an unresolvable reference reported no error: %+v", diagram.Diagnostics)
	}
}

// A pair declared only through Function.Operations has no CRUDDefinition. If it
// opposes a declaration, the diagram must still name the contradiction rather
// than hiding it behind an absent row.
func TestThreeElementDiagramReportsConflictWithoutADeclaredRow(t *testing.T) {
	path := writeDiagramModel(t, "derived-only.json", map[string]any{
		"modelId": "derived-only-model",
		"name":    "Derived only model",
		"dataElements": []map[string]any{
			{"id": "order", "name": "Order"},
			{"id": "invoice", "name": "Invoice"},
		},
		"functionElements": []map[string]any{
			// Set order: contradicts the declaration below.
			{"id": "submit", "name": "Submit",
				"operations": []map[string]any{{"dataId": "order", "operations": "U"}}},
			// Set invoice: declared only through the CRUD definition, no conflict.
			{"id": "bill", "name": "Bill"},
		},
		"businessElements": []map[string]any{},
		"crudDefinitions": []map[string]any{
			{"id": "order-declared", "functionId": "submit", "dataId": "order",
				"delete": true, "source": "declared"},
			{"id": "invoice-declared", "functionId": "bill", "dataId": "invoice",
				"read": true, "source": "declared"},
		},
	})
	draft, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: path})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	diagram := draft.Occurrences[0].Content.Diagram
	if len(diagram.CRUDs) != 2 {
		t.Fatalf("cruds=%+v want the two declared cells", diagram.CRUDs)
	}
	flagged := map[string]bool{}
	for _, cell := range diagram.CRUDs {
		flagged[cell.ID] = cell.ConflictingDeclarations
	}
	if !flagged["order-declared"] {
		t.Fatalf("the opposing pair was not flagged: %+v", diagram.CRUDs)
	}
	if flagged["invoice-declared"] {
		t.Fatalf("a distinct pair was flagged: %+v", diagram.CRUDs)
	}
	if len(diagram.OpposingCRUDPairs) != 1 || diagram.OpposingCRUDPairs[0] != "submit/order" {
		t.Fatalf("opposingCrudPairs=%v want [submit/order]", diagram.OpposingCRUDPairs)
	}
}

package documentdraft

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// diagramFixture writes one three-element analysis model and returns its path.
// The model is synthetic and self-contained: this kind reads a separate analysis
// artifact, not a TEI Workspace.
func diagramFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "model.json")
	model := map[string]any{
		"schemaVersion": "tei.three-element/v0",
		"effect":        "read-only",
		"modelId":       "sales-analysis",
		"name":          "Sales analysis",
		"dataElements": []map[string]any{
			{"id": "sales.order", "name": "Order", "attributes": []map[string]any{
				{"id": "sales.order.order_id", "name": "Order ID", "type": "std.identifier", "key": true},
				{"id": "sales.order.status", "name": "Status", "type": "std.text"},
			}},
			{"id": "customer.customer", "name": "Customer", "attributes": []map[string]any{
				{"id": "customer.customer.customer_id", "name": "Customer ID", "type": "std.identifier", "key": true},
			}},
		},
		"functionElements": []map[string]any{
			{"id": "sales.order.submit", "name": "Submit order", "type": "XF200", "operations": []map[string]any{
				{"dataId": "sales.order", "operations": "C"},
			}},
		},
		"businessElements": []map[string]any{
			{"id": "sales.checkout", "name": "Checkout", "role": "actor", "activities": []map[string]any{
				{"id": "sales.checkout.place", "name": "Place order", "functionIds": []string{"sales.order.submit"}},
			}},
		},
		"crossReferences": []map[string]any{
			{"id": "xref-business-function", "kind": "business-to-function",
				"fromId": "sales.checkout", "fromKind": "business",
				"toId": "sales.order.submit", "toKind": "function", "directed": true},
			{"id": "xref-function-data", "kind": "function-to-data",
				"fromId": "sales.order.submit", "fromKind": "function",
				"toId": "sales.order", "toKind": "data", "directed": true},
			// One edge whose target the model does not declare: it must survive.
			{"id": "xref-data-missing", "kind": "data-to-data",
				"fromId": "sales.order", "fromKind": "data",
				"toId": "sales.nothing", "toKind": "data", "directed": true},
		},
		"crudDefinitions": []map[string]any{
			{"id": "crud-submit-order", "functionId": "sales.order.submit", "dataId": "sales.order",
				"create": true, "read": true, "source": "declared"},
		},
	}
	encoded, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestThreeElementDiagramProjectsNodesAndEdges is the first-pass proof: the
// declared elements become nodes, the declared relations become edges, and the
// declared CRUD mapping is carried as declared.
func TestThreeElementDiagramProjectsNodesAndEdges(t *testing.T) {
	modelPath := diagramFixture(t)
	draft, err := Build(Request{
		DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if draft.DocumentKind != DocumentKindThreeElementDiagram {
		t.Fatalf("documentKind=%q", draft.DocumentKind)
	}
	if len(draft.Occurrences) != 1 || draft.Occurrences[0].Kind != OccurrenceDiagram {
		t.Fatalf("occurrences=%+v", draft.Occurrences)
	}
	diagram := draft.Occurrences[0].Content.Diagram
	if diagram == nil {
		t.Fatal("the diagram occurrence carries no diagram content")
	}
	if diagram.ModelID != "sales-analysis" {
		t.Fatalf("modelId=%q", diagram.ModelID)
	}
	// Nodes: three layers, five elements.
	layers := map[string]int{}
	for _, node := range diagram.Nodes {
		layers[node.Layer]++
	}
	if layers["data"] != 2 || layers["function"] != 1 || layers["business"] != 1 {
		t.Fatalf("layers=%v", layers)
	}
	// The attribute and operation counts name what the element declares.
	for _, node := range diagram.Nodes {
		if node.ID == "sales.order" && node.AttributeCount != 2 {
			t.Fatalf("sales.order attributeCount=%d want 2", node.AttributeCount)
		}
		if node.ID == "sales.order.submit" && node.OperationCount != 1 {
			t.Fatalf("submit operationCount=%d want 1", node.OperationCount)
		}
		if node.ID == "sales.checkout" && node.ActivityCount != 1 {
			t.Fatalf("checkout activityCount=%d want 1", node.ActivityCount)
		}
	}
	if len(diagram.Edges) != 3 {
		t.Fatalf("edges=%+v", diagram.Edges)
	}
	// The CRUD mapping names the declared letters in a fixed order.
	if len(diagram.CRUDs) != 1 || diagram.CRUDs[0].Operations != "CR" {
		t.Fatalf("cruds=%+v", diagram.CRUDs)
	}
	// The authority limits name the element mapping as model-declared, because
	// that mapping is the Owner decision this projection must not appear to have
	// made.
	found := false
	for _, limit := range draft.AuthorityLimits {
		if limit == "diagram-element-mapping-is-model-declared-not-tei-decided" {
			found = true
		}
	}
	if !found {
		t.Fatalf("authorityLimits=%v", draft.AuthorityLimits)
	}
}

// TestThreeElementDiagramKeepsAnUnresolvedEdge pins that an edge whose endpoint
// the model does not declare survives, named as unresolved. Dropping it would
// present a complete-looking diagram of a model that is not complete.
func TestThreeElementDiagramKeepsAnUnresolvedEdge(t *testing.T) {
	modelPath := diagramFixture(t)
	draft, err := Build(Request{
		DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	diagram := draft.Occurrences[0].Content.Diagram
	var unresolved []DiagramEdge
	for _, edge := range diagram.Edges {
		if edge.Resolution == ResolutionUnresolved {
			unresolved = append(unresolved, edge)
		}
	}
	if len(unresolved) != 1 || unresolved[0].ID != "xref-data-missing" {
		t.Fatalf("unresolved=%+v", unresolved)
	}
	if unresolved[0].ToID != "sales.nothing" {
		t.Fatalf("edge endpoint=%q", unresolved[0].ToID)
	}
	if len(diagram.UnresolvedEdges) != 1 || diagram.UnresolvedEdges[0] != "xref-data-missing" {
		t.Fatalf("unresolvedEdges=%v", diagram.UnresolvedEdges)
	}
	// The endpoint is not a node: an undeclared ID never becomes one.
	for _, node := range diagram.Nodes {
		if node.ID == "sales.nothing" {
			t.Fatal("an undeclared ID became a node")
		}
	}
}

// TestThreeElementDiagramIsDeterministicForTheSameModel pins that the same model
// file always produces the same diagram, node and edge order included.
func TestThreeElementDiagramIsDeterministicForTheSameModel(t *testing.T) {
	modelPath := diagramFixture(t)
	first, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath})
	if err != nil {
		t.Fatalf("first Build: %v", err)
	}
	second, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath})
	if err != nil {
		t.Fatalf("second Build: %v", err)
	}
	if first.DraftID != second.DraftID {
		t.Fatalf("draftId changed: %q vs %q", first.DraftID, second.DraftID)
	}
	if first.Occurrences[0].ContentDigest != second.Occurrences[0].ContentDigest {
		t.Fatal("the diagram digest changed between identical reads")
	}
}

// TestThreeElementDiagramReconcileReportsAMovedModel pins that a changed model
// moves the diagram, and that an unchanged one does not.
func TestThreeElementDiagramReconcileReportsAMovedModel(t *testing.T) {
	modelPath := diagramFixture(t)
	draft, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	report, err := Reconcile(draft, "")
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if report.ModelReference.Status != ModelReferenceCurrent {
		t.Fatalf("model reference=%+v", report.ModelReference)
	}
	if len(report.NeedsReconfirmation) != 0 {
		t.Fatalf("an unchanged model produced reconfirmation items: %v", report.NeedsReconfirmation)
	}

	// One attribute is added to the model: the diagram's nodes move.
	raw, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	var model map[string]any
	if err := json.Unmarshal(raw, &model); err != nil {
		t.Fatal(err)
	}
	data := model["dataElements"].([]any)
	first := data[0].(map[string]any)
	attributes := first["attributes"].([]any)
	attributes = append(attributes, map[string]any{
		"id": "sales.order.placed_at", "name": "Placed at", "type": "std.datetime",
	})
	first["attributes"] = attributes
	model["dataElements"] = data
	changed, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := Reconcile(draft, "")
	if err != nil {
		t.Fatalf("Reconcile after change: %v", err)
	}
	if after.ModelReference.Status != ModelReferenceStale {
		t.Fatalf("model reference=%+v", after.ModelReference)
	}
	if len(after.NeedsReconfirmation) != 1 {
		t.Fatalf("needsReconfirmation=%v want the one diagram occurrence", after.NeedsReconfirmation)
	}
	if after.Occurrences[0].Status != OccurrenceContentChanged {
		t.Fatalf("status=%q want %q", after.Occurrences[0].Status, OccurrenceContentChanged)
	}
}

// TestThreeElementDiagramRefusesMixedInputs keeps the four input shapes apart at
// the diagram boundary too.
func TestThreeElementDiagramRefusesMixedInputs(t *testing.T) {
	modelPath := diagramFixture(t)
	for name, request := range map[string]Request{
		"no model path":    {DocumentKind: DocumentKindThreeElementDiagram},
		"relative path":    {DocumentKind: DocumentKindThreeElementDiagram, ModelPath: "model.json"},
		"application":      {DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath, Application: "order_entry"},
		"sections":         {DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath, Sections: []SectionSpec{{Subject: orderSubject()}}},
		"element selector": {DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath, Subject: orderSubject()},
		"top relation":     {DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath, RelationID: "sales.order.customer"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Build(request)
			if err == nil {
				t.Fatalf("Build accepted %+v", request)
			}
			var draftErr *DraftError
			if !errors.As(err, &draftErr) || draftErr.Code != CodeInvalidArgument {
				t.Fatalf("code=%v want %q", err, CodeInvalidArgument)
			}
		})
	}
}

// TestOtherKindsRefuseAModelPath pins the other direction: a kind that does not
// read an analysis model refuses the flag rather than ignoring it.
func TestOtherKindsRefuseAModelPath(t *testing.T) {
	modelPath := diagramFixture(t)
	fixture := newFixture(t)
	_, err := Build(Request{Root: fixture.root, Subject: orderSubject(), ModelPath: modelPath})
	if err == nil {
		t.Fatal("Build accepted a model path for an element draft")
	}
	var draftErr *DraftError
	if !errors.As(err, &draftErr) || draftErr.Code != CodeInvalidArgument {
		t.Fatalf("code=%v want %q", err, CodeInvalidArgument)
	}
}

// TestThreeElementDiagramRefusesAnUnreadableModel pins that a model that cannot
// be read is a refusal, not an empty diagram.
func TestThreeElementDiagramRefusesAnUnreadableModel(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "absent.json")
	_, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: missing})
	if err == nil {
		t.Fatal("Build accepted a missing model")
	}
	var draftErr *DraftError
	if !errors.As(err, &draftErr) || draftErr.Code != CodeInvalidArgument {
		t.Fatalf("code=%v want %q", err, CodeInvalidArgument)
	}

	// A file that is not JSON is the same refusal, not a partial diagram.
	notJSON := filepath.Join(dir, "model.json")
	if err := os.WriteFile(notJSON, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: notJSON}); err == nil {
		t.Fatal("Build accepted an unreadable model")
	}

	// A model without an ID has no identity to project, so it is refused.
	noID := filepath.Join(dir, "no-id.json")
	if err := os.WriteFile(noID, []byte(`{"name":"nameless"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: noID}); err == nil {
		t.Fatal("Build accepted a model with no ID")
	}
}

// TestThreeElementDiagramNeverReadsAWorkspace pins the boundary between the two
// models: the diagram kind ignores the Workspace root entirely, so a caller
// cannot mix a TEI Workspace with an analysis model and have the document layer
// decide which one wins.
func TestThreeElementDiagramNeverReadsAWorkspace(t *testing.T) {
	modelPath := diagramFixture(t)
	// A root that is not a Workspace at all: the diagram still builds, because
	// this kind never reads it.
	draft, err := Build(Request{
		Root:         filepath.Join(t.TempDir(), "not-a-workspace"),
		DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	// No Workspace identity is recorded at all: this kind never read one.
	if draft.Model.Workspace != nil {
		t.Fatalf("model=%+v carries a Workspace read", draft.Model)
	}
	if draft.Model.WorldFingerprint != "" || draft.Model.SourceIndexFingerprint != "" {
		t.Fatalf("model=%+v carries Workspace fingerprints", draft.Model)
	}
	if draft.Model.ModelRevision != draft.Occurrences[0].Content.Diagram.ModelFingerprint {
		t.Fatal("the model revision is not the analysis model's fingerprint")
	}
}

// TestThreeElementDiagramToleratesABomInTheModel pins that a model written by a
// Windows editor still projects. A BOM is an encoding detail, not a broken model,
// and refusing it would report a failure the reader cannot act on.
func TestThreeElementDiagramToleratesABomInTheModel(t *testing.T) {
	modelPath := diagramFixture(t)
	raw, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	withBOM := append([]byte{0xEF, 0xBB, 0xBF}, raw...)
	if err := os.WriteFile(modelPath, withBOM, 0o600); err != nil {
		t.Fatal(err)
	}
	draft, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(draft.Occurrences) != 1 || draft.Occurrences[0].Content.Diagram == nil {
		t.Fatalf("occurrences=%+v", draft.Occurrences)
	}
	if len(draft.Occurrences[0].Content.Diagram.Nodes) != 4 {
		t.Fatalf("nodes=%+v", draft.Occurrences[0].Content.Diagram.Nodes)
	}
}

// TestThreeElementDiagramReconcileRefusesADraftWithoutAModelPath pins that a
// hand-edited diagram draft with no model path is refused rather than compared
// against nothing.
func TestThreeElementDiagramReconcileRefusesADraftWithoutAModelPath(t *testing.T) {
	modelPath := diagramFixture(t)
	draft, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	tampered := draft
	tampered.Subjects = nil
	if _, err := Reconcile(tampered, ""); err == nil {
		t.Fatal("Reconcile accepted a diagram draft with no model path")
	}
}

// TestThreeElementDiagramPreservesDeclaredElementAuthorityLimits is an
// independent consumer regression: the generated diagram must still carry
// limits declared in the source JSON after a structural fingerprint is computed.
func TestThreeElementDiagramPreservesDeclaredElementAuthorityLimits(t *testing.T) {
	modelPath := diagramFixture(t)
	raw, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	var source map[string]any
	if err := json.Unmarshal(raw, &source); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"data:sales.order":            "data-owner-approval",
		"function:sales.order.submit": "function-review-required",
		"business:sales.checkout":     "business-scope-limited",
	}
	setLimit := func(field, id, limit string) {
		t.Helper()
		elements, ok := source[field].([]any)
		if !ok {
			t.Fatalf("%s is not an element array", field)
		}
		for _, item := range elements {
			element, ok := item.(map[string]any)
			if !ok {
				t.Fatalf("%s has non-object element", field)
			}
			if element["id"] == id {
				element["authorityLimits"] = []string{limit}
				return
			}
		}
		t.Fatalf("missing %s/%s", field, id)
	}
	setLimit("dataElements", "sales.order", want["data:sales.order"])
	setLimit("functionElements", "sales.order.submit", want["function:sales.order.submit"])
	setLimit("businessElements", "sales.checkout", want["business:sales.checkout"])

	updated, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, updated, 0o600); err != nil {
		t.Fatal(err)
	}
	draft, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	diagram := draft.Occurrences[0].Content.Diagram
	if diagram == nil {
		t.Fatal("diagram content is missing")
	}
	seen := map[string]bool{}
	for _, node := range diagram.Nodes {
		key := node.Layer + ":" + node.ID
		limit, expected := want[key]
		if !expected {
			continue
		}
		seen[key] = true
		if len(node.AuthorityLimits) != 1 || node.AuthorityLimits[0] != limit {
			t.Fatalf("node %s lost source authority limits: got %v, want [%s]",
				key, node.AuthorityLimits, limit)
		}
	}
	for key := range want {
		if !seen[key] {
			t.Fatalf("source node %s was omitted from the diagram", key)
		}
	}
}

// TestThreeElementDiagramReconcileDetectsAuthorityOnlySourceDrift proves that
// the current diagram Source must be requalified even when its STRUCTURAL
// fingerprint is unchanged. The source JSON, not the generated report, supplies
// the independent expectation for the authority difference.
func TestThreeElementDiagramReconcileDetectsAuthorityOnlySourceDrift(t *testing.T) {
	modelPath := diagramFixture(t)
	raw, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	var source map[string]any
	if err := json.Unmarshal(raw, &source); err != nil {
		t.Fatal(err)
	}
	elements := source["dataElements"].([]any)
	order := elements[0].(map[string]any)
	order["authorityLimits"] = []string{"owner-review-required"}
	withLimits, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, withLimits, 0o600); err != nil {
		t.Fatal(err)
	}

	recorded, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath})
	if err != nil {
		t.Fatalf("first Build: %v", err)
	}
	before, err := Reconcile(recorded, "")
	if err != nil {
		t.Fatalf("unchanged Reconcile: %v", err)
	}
	if before.ModelReference.Status != ModelReferenceCurrent {
		t.Fatalf("unchanged model reference=%+v", before.ModelReference)
	}
	if len(before.NeedsReconfirmation) != 0 {
		t.Fatalf("unchanged source requires reconfirmation: %v", before.NeedsReconfirmation)
	}

	order["authorityLimits"] = []string{"owner-approval-revoked"}
	changed, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	updated, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath})
	if err != nil {
		t.Fatalf("second Build: %v", err)
	}
	if updated.Model.ModelRevision != recorded.Model.ModelRevision {
		t.Fatalf("authority-only edit changed structural fingerprint: %q != %q",
			updated.Model.ModelRevision, recorded.Model.ModelRevision)
	}
	after, err := Reconcile(recorded, "")
	if err != nil {
		t.Fatalf("authority-only Reconcile: %v", err)
	}
	if after.ModelReference.Status != ModelReferenceStale ||
		!containsString(after.ModelReference.Diffs, "sourceFingerprint") {
		t.Fatalf("authority-only source change was not requalified: %+v", after.ModelReference)
	}
	if len(after.NeedsReconfirmation) != 1 ||
		len(after.Occurrences) != 1 ||
		after.Occurrences[0].Status != OccurrenceContentChanged {
		t.Fatalf("diagram authority change not reported per occurrence: %+v", after)
	}
}

// TestThreeElementDiagramReconcileDetectsByteOnlyAndMissingSourceDigest
// distinguishes a source-only change from a missing historic observation.
// Neither may silently make a recorded draft current.
func TestThreeElementDiagramReconcileDetectsByteOnlyAndMissingSourceDigest(t *testing.T) {
	modelPath := diagramFixture(t)
	recorded, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	raw, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	var source map[string]any
	if err := json.Unmarshal(raw, &source); err != nil {
		t.Fatal(err)
	}
	// Reformat only: decoded structure and diagram remain identical.
	pretty, err := json.MarshalIndent(source, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if string(pretty) == string(raw) {
		t.Fatal("expected a distinct raw representation")
	}
	if err := os.WriteFile(modelPath, pretty, 0o600); err != nil {
		t.Fatal(err)
	}
	reformatted, err := Build(Request{DocumentKind: DocumentKindThreeElementDiagram, ModelPath: modelPath})
	if err != nil {
		t.Fatalf("reformatted Build: %v", err)
	}
	if recorded.Model.ModelRevision != reformatted.Model.ModelRevision ||
		recorded.Occurrences[0].ContentDigest != reformatted.Occurrences[0].ContentDigest {
		t.Fatal("source-only formatting altered the structural model/diagram")
	}
	diff, err := Reconcile(recorded, "")
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if diff.ModelReference.Status != ModelReferenceStale ||
		!containsString(diff.ModelReference.Diffs, "sourceFingerprint") ||
		len(diff.Occurrences) != 1 ||
		diff.Occurrences[0].Status != OccurrenceConditionsChanged {
		t.Fatalf("source formatting drift was not distinguished: %+v", diff)
	}

	// An old or incomplete draft without the source digest must not be
	// accepted as current merely because the structural digest is identical.
	missing := reformatted
	missing.Occurrences = append([]Occurrence(nil), reformatted.Occurrences...)
	missing.Occurrences[0].GenerationConditions.InputDigests = nil
	unknown, err := Reconcile(missing, "")
	if err != nil {
		t.Fatalf("missing digest Reconcile: %v", err)
	}
	if unknown.ModelReference.Status != ModelReferenceStale ||
		!containsString(unknown.ModelReference.Diffs, "sourceFingerprint") {
		t.Fatalf("missing source digest silently qualified: %+v", unknown.ModelReference)
	}
}

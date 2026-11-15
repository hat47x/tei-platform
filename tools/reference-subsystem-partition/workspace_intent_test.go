package partition

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/application"
	"tei/internal/operations"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// TestReferenceWorkspaceIntentStopsBeforeDirectCUD is a real Workspace read,
// not an Accepted Owner/Function I/O producer or a runtime Effect test.
// It freezes an intentionally incomplete, but navigable, business declaration:
// Pattern -> Capability -> Data is explicit; direct C/U/D and Owner are not.
func TestReferenceWorkspaceIntentStopsBeforeDirectCUD(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load reference Workspace: %v", err)
	}

	var patterns []application.Pattern
	for _, app := range loaded.World.Applications {
		for _, pattern := range app.Patterns {
			if string(pattern.ID) == "order_entry.submit_order" {
				patterns = append(patterns, pattern)
			}
		}
	}
	if len(patterns) != 1 {
		t.Fatalf("exactly one submit Pattern required, got %d", len(patterns))
	}
	pattern := patterns[0]
	if pattern.Kind != application.PatternCommand ||
		string(pattern.Capability) != "sales.order.submit" {
		t.Fatalf("unexpected declared Pattern intent: %+v", pattern)
	}

	var capabilities []semantic.Capability
	var data []semantic.Data
	for _, ns := range loaded.World.Semantics {
		for _, capability := range ns.Capabilities {
			if string(capability.ID) == "sales.order.submit" {
				capabilities = append(capabilities, capability)
			}
		}
		for _, item := range ns.Data {
			if string(item.ID) == "sales.order" {
				data = append(data, item)
			}
		}
	}
	if len(capabilities) != 1 || len(data) != 1 {
		t.Fatalf("not a unique Capability/Data link: capabilities=%d data=%d", len(capabilities), len(data))
	}
	capability := capabilities[0]
	if len(capability.Inputs) != 1 ||
		string(capability.Inputs[0].Type) != "sales.order" ||
		string(capability.Output) != "sales.order" {
		t.Fatalf("unexpected declared Capability input/output: %+v", capability)
	}
	if capability.Effect != nil {
		t.Fatalf("this fixed reference does not declare a C/U/D effect: %+v", capability.Effect)
	}

	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("build Load-byte-bound Source Index: %v", err)
	}
	subjects := []sourceindex.Subject{
		{Kind: "application-pattern", ID: "order_entry.submit_order"},
		{Kind: "semantic-capability", ID: "sales.order.submit"},
		{Kind: "semantic-data", ID: "sales.order"},
	}
	worldFingerprint, sourceFingerprint, err := operations.WorkspaceProjectionFingerprints(loaded.World, index)
	if err != nil {
		t.Fatalf("fingerprint captured Workspace: %v", err)
	}
	for _, subject := range subjects {
		if len(index.ReferencesFor(subject)) == 0 {
			t.Fatalf("no indexed Source location for %v", subject)
		}
		// This public Inspector makes its own Load attempt. Consistent
		// fingerprints do not prove a shared atomic Read Attempt.
		inspected, err := operations.InspectSubject(root, subject)
		if err != nil {
			t.Fatalf("inspect real Workspace subject %v: %v", subject, err)
		}
		if inspected.Subject != subject || inspected.Effect != "read-only" ||
			inspected.SourceStatus != "indexed" || inspected.Workspace == nil ||
			inspected.WorldFingerprint == "" || inspected.SourceIndexFingerprint == "" ||
			len(inspected.Sources.References) == 0 {
			t.Fatalf("unqualified read-only inspection for %v: %+v", subject, inspected)
		}
		if !reflect.DeepEqual(*inspected.Workspace, loaded.Identity()) {
			t.Fatalf("inspection Workspace differs from loaded Workspace for %v", subject)
		}
		if inspected.WorldFingerprint != worldFingerprint || inspected.SourceIndexFingerprint != sourceFingerprint {
			t.Fatalf("Workspace revision drift across subject inspections for %v", subject)
		}
		var canonical struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(inspected.Canonical, &canonical); err != nil ||
			canonical.ID != subject.ID {
			t.Fatalf("inspect returned wrong canonical for %v: %s (%v)", subject, inspected.Canonical, err)
		}
	}

	// No partition.Problem is constructed from these observations:
	// no accepted Primary Subsystem Owner or direct Function C/U/D
	// occurrence was established by any of the checks above.
}

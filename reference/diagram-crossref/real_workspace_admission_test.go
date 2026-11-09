package diagramcrossref

// The fixture contract tests above deliberately use synthetic JSON. This
// narrow test additionally enters the actual TEI Workspace through the
// existing application-facing InspectSubject operation. It is not a
// Data/Function/Business diagram or a CRUD projector conformance test.
import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

func TestRealWorkspaceQueryAndRelationSourceAdmission(t *testing.T) {
	root := filepath.Join("..", "workspace")
	cases := []sourceindex.Subject{
		{Kind: "query", ID: "order_entry.recent_orders"},
		{Kind: "semantic-relation", ID: "sales.order.lines"},
	}
	var worldFP, indexFP string
	var workspaceIdentity any
	for _, selected := range cases {
		t.Run(selected.Kind+"/"+selected.ID, func(t *testing.T) {
			got, err := operations.InspectSubject(root, selected)
			if err != nil {
				t.Fatalf("actual workspace inspect %s:%s: %v", selected.Kind, selected.ID, err)
			}
			if got.Subject != selected || got.Effect != "read-only" {
				t.Fatalf("unexpected subject/effect: %+v", got)
			}
			if got.SchemaVersion != "tei.operations.inspect/v0" {
				t.Fatalf("unrecognized operation contract: %q", got.SchemaVersion)
			}
			if !json.Valid(got.Canonical) || string(got.Canonical) == "null" {
				t.Fatalf("source not resolved into canonical inspection: %s", got.Canonical)
			}
			if got.SourceStatus != "indexed" || len(got.Sources.References) == 0 {
				t.Fatalf("actual source provenance unavailable: status=%q references=%d", got.SourceStatus, len(got.Sources.References))
			}
			if got.Workspace == nil || got.WorldFingerprint == "" || got.SourceIndexFingerprint == "" {
				t.Fatalf("missing source/workspace binding: %+v", got)
			}
			if worldFP != "" && (got.WorldFingerprint != worldFP || got.SourceIndexFingerprint != indexFP) {
				t.Fatalf("different snapshots/World revisions in same exact workspace read")
			}
			if workspaceIdentity != nil && !reflect.DeepEqual(workspaceIdentity, *got.Workspace) {
				t.Fatalf("different workspace identity for two reads")
			}
			worldFP, indexFP = got.WorldFingerprint, got.SourceIndexFingerprint
			workspaceIdentity = *got.Workspace
		})
	}
	// This establishes only two explicitly named, source-indexed subjects.
	// It does not establish complete Relation/Function/Business closure,
	// that this query uses the selected relation, an atomic multi-read
	// snapshot, accepted meaning, or any Provider / Graph / CRUD operation.
}

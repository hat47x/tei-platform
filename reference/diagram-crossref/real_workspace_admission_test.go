package diagramcrossref

// The fixture contract tests above deliberately use synthetic JSON. This
// narrow test additionally enters the actual TEI Workspace through the
// existing application-facing InspectSubject operation. It is not a
// Data/Function/Business diagram or a CRUD projector conformance test.
import (
	"encoding/json"
	"errors"
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
			var canonical struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(got.Canonical, &canonical); err != nil || canonical.ID != selected.ID {
				t.Fatalf("unexpected canonical identity for kind=%q: id=%q err=%v", selected.Kind, canonical.ID, err)
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

func TestRealWorkspaceSourceAdmissionRejectsMismatchedSubjectKind(t *testing.T) {
	root := filepath.Join("..", "workspace")
	for _, invalid := range []sourceindex.Subject{
		{Kind: "query", ID: "sales.order.lines"},
		{Kind: "semantic-relation", ID: "order_entry.recent_orders"},
	} {
		t.Run(invalid.Kind+"/"+invalid.ID, func(t *testing.T) {
			_, err := operations.InspectSubject(root, invalid)
			var inspection *operations.InspectionError
			if !errors.As(err, &inspection) || inspection.Code != operations.InspectionNotFound {
				t.Fatalf("role/ID mismatch must not resolve another Subject: %v", err)
			}
		})
	}
	// False-positive name matches are not proof of Source/Meaning identity.
}

// The same explicitly named subjects must survive three existing read-only
// operations. None of these calls provides an atomic multi-operation snapshot,
// or an independent proof that all required Function/Business edges exist.
func TestRealWorkspaceCatalogInspectTraceAgreeOnExactSources(t *testing.T) {
	root := filepath.Join("..", "workspace")
	catalog, err := operations.CatalogWorkspace(root, "")
	if err != nil {
		t.Fatalf("read model catalog from actual Workspace: %v", err)
	}
	if catalog.Effect != "read-only" || catalog.Workspace == nil ||
		catalog.WorldFingerprint == "" || catalog.SourceIndexFingerprint == "" {
		t.Fatalf("catalog missing read-only Workspace or revision witness: %+v", catalog)
	}
	for _, requested := range []sourceindex.Subject{
		{Kind: "query", ID: "order_entry.recent_orders"},
		{Kind: "semantic-relation", ID: "sales.order.lines"},
	} {
		t.Run(requested.Kind+"/"+requested.ID, func(t *testing.T) {
			var matching []operations.ModelCatalogEntry
			for _, entry := range catalog.Entries {
				if entry.Subject == requested {
					matching = append(matching, entry)
				}
			}
			if len(matching) != 1 || matching[0].Prelude || matching[0].Ambiguous ||
				matching[0].SourceStatus != "indexed" {
				t.Fatalf("required native Subject not uniquely indexed in catalog: %+v", matching)
			}
			inspected, err := operations.InspectSubject(root, requested)
			if err != nil {
				t.Fatalf("inspect exact catalog Subject: %v", err)
			}
			if inspected.Subject != requested || inspected.SourceStatus != matching[0].SourceStatus ||
				inspected.WorldFingerprint != catalog.WorldFingerprint ||
				inspected.SourceIndexFingerprint != catalog.SourceIndexFingerprint ||
				inspected.Workspace == nil || !reflect.DeepEqual(*inspected.Workspace, *catalog.Workspace) {
				t.Fatalf("catalog/inspect Workspace or Subject mismatch for %v", requested)
			}
			trace, err := operations.TraceSubjectSources(root, requested)
			if err != nil {
				t.Fatalf("read independent Source reference projection: %v", err)
			}
			if trace.Subject != requested || len(trace.References) == 0 ||
				!reflect.DeepEqual(trace.References, inspected.Sources.References) ||
				!reflect.DeepEqual(trace.Artifacts, inspected.Sources.Artifacts) {
				t.Fatalf("trace/inspect Source references differ for %v", requested)
			}
		})
	}
	// Cross-command agreement for two declared Subjects is not complete
	// Source closure, atomic snapshot, semantic approval, or Graph/CRUD export.
}

// The read-only Catalog lists Relation -> parent Data, while canonical
// inspection resolves the Relation's explicitly declared target Data. This
// probes a real Source-to-navigation edge without a second graph registry.
func TestRealWorkspaceRelationParentAndTargetMatchCatalog(t *testing.T) {
	root := filepath.Join("..", "workspace")
	requested := sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.lines"}
	catalog, err := operations.CatalogWorkspace(root, "")
	if err != nil {
		t.Fatal(err)
	}
	var relationEntries []operations.ModelCatalogEntry
	for _, entry := range catalog.Entries {
		if entry.Subject == requested {
			relationEntries = append(relationEntries, entry)
		}
	}
	if len(relationEntries) != 1 {
		t.Fatalf("source Relation has %d catalog occurrences", len(relationEntries))
	}
	entry := relationEntries[0]
	if entry.Ambiguous || entry.Parent == nil || entry.Parent.Kind != "semantic-data" {
		t.Fatalf("Relation has no unique qualified Data parent: %+v", entry)
	}
	inspected, err := operations.InspectSubject(root, requested)
	if err != nil {
		t.Fatal(err)
	}
	var relation struct {
		ID     string `json:"id"`
		Target string `json:"target"`
	}
	if err := json.Unmarshal(inspected.Canonical, &relation); err != nil {
		t.Fatal(err)
	}
	if relation.ID != requested.ID || relation.Target == "" || relation.Target != entry.Detail {
		t.Fatalf("Catalog Relation target and canonical Source disagree: %+v / %+v", entry, relation)
	}
	parent, err := operations.InspectSubject(root, *entry.Parent)
	if err != nil {
		t.Fatalf("canonical Relation parent is not inspectable: %v", err)
	}
	var data struct {
		ID        string `json:"id"`
		Relations []struct {
			ID string `json:"id"`
		} `json:"relations"`
	}
	if err := json.Unmarshal(parent.Canonical, &data); err != nil {
		t.Fatal(err)
	}
	matches := 0
	for _, member := range data.Relations {
		if member.ID == relation.ID {
			matches++
		}
	}
	if data.ID != entry.Parent.ID || matches != 1 ||
		parent.WorldFingerprint != inspected.WorldFingerprint ||
		parent.SourceIndexFingerprint != inspected.SourceIndexFingerprint {
		t.Fatalf("Relation parent/child Source identity mismatch: parent=%q relation=%q count=%d", data.ID, relation.ID, matches)
	}
	// A declared target name is not a resolved Data node. Require one exact
	// TEI-native target declaration, as opposed to accepting a diagram label,
	// a string in Catalog Detail, or an unindexed Source-only occurrence.
	targetSubject := sourceindex.Subject{Kind: "semantic-data", ID: relation.Target}
	targetMatches := 0
	for _, candidate := range catalog.Entries {
		if candidate.Subject != targetSubject {
			continue
		}
		targetMatches++
		if candidate.Ambiguous || candidate.Prelude || candidate.SourceStatus != "indexed" {
			t.Fatalf("relation target is not a uniquely indexed native Data: %+v", candidate)
		}
	}
	if targetMatches != 1 {
		t.Fatalf("relation target Data should be unique and indexed: target=%+v occurrences=%d", targetSubject, targetMatches)
	}
	target, err := operations.InspectSubject(root, targetSubject)
	if err != nil {
		t.Fatalf("declared Relation target is not inspectable in actual Workspace: %v", err)
	}
	var targetCanonical struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(target.Canonical, &targetCanonical); err != nil || targetCanonical.ID != targetSubject.ID {
		t.Fatalf("Relation target Canonical ID mismatch: expected=%q actual=%q err=%v", targetSubject.ID, targetCanonical.ID, err)
	}
	if target.Subject != targetSubject || target.SourceStatus != "indexed" ||
		len(target.Sources.References) == 0 || target.Workspace == nil ||
		!reflect.DeepEqual(*target.Workspace, *inspected.Workspace) ||
		target.WorldFingerprint != inspected.WorldFingerprint ||
		target.SourceIndexFingerprint != inspected.SourceIndexFingerprint {
		t.Fatalf("Relation target does not share expected Workspace or source-indexed World: %+v", target)
	}
	// This establishes only the three source-indexed declarations
	// parent Data -> Relation -> target Data in separate read attempts.
	// It is NOT a full Required closure, multi-read atomic snapshot,
	// accepted relation meaning, stable graphical edge ID, or edit grant.
}

package diagramcrossref

// The fixture contract tests above deliberately use synthetic JSON. This
// narrow test additionally enters the actual TEI Workspace through the
// existing application-facing InspectSubject operation. It is not a
// Data/Function/Business diagram or a CRUD projector conformance test.
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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

// An indexed Source locator is useful for read-only navigation only when
// its on-disk bytes still match its observed fingerprint and byte positions.
// This test checks two real TEI-native subjects, not imported Provider raw,
// Source edit authority, or the completeness of a whole diagram projection.
func TestRealWorkspaceIndexedSourceRangeAndByteFingerprint(t *testing.T) {
	root := filepath.Join("..", "workspace")
	for _, requested := range []sourceindex.Subject{
		{Kind: "query", ID: "order_entry.recent_orders"},
		{Kind: "semantic-relation", ID: "sales.order.lines"},
	} {
		t.Run(requested.Kind+"/"+requested.ID, func(t *testing.T) {
			inspected, err := operations.InspectSubject(root, requested)
			if err != nil {
				t.Fatal(err)
			}
			sources := inspected.Sources
			if sources.Subject != requested || len(sources.References) == 0 || len(sources.Artifacts) == 0 {
				t.Fatalf("no source locator bound to exact subject: %+v", sources)
			}
			index := sourceindex.Index{
				Artifacts:  sources.Artifacts,
				References: sources.References,
			}
			read := func(path string) ([]byte, error) {
				clean, err := sourceindex.NormalizeWorkspacePath(path)
				if err != nil || clean != path {
					t.Fatalf("invalid managed Source path %q: %v", path, err)
				}
				// Only a trusted, checked-in reference Workspace is used
				// here. Path normalization is not an OS symlink sandbox.
				return os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
			}
			if err := index.Validate(read); err != nil {
				t.Fatalf("actual Source bytes/fingerprint/range mismatch: %v", err)
			}
			for _, ref := range sources.References {
				if ref.Subject != requested {
					t.Fatalf("Source reference changed subject: %+v", ref)
				}
				bytes, err := read(ref.ArtifactPath)
				if err != nil {
					t.Fatal(err)
				}
				if err := ref.Range.Validate(bytes); err != nil {
					t.Fatalf("byte locator is stale: %v", err)
				}
			}
			changedPath := sources.Artifacts[0].Path
			// A harmless one-byte extension that preserves the old range
			// must nevertheless invalidate the artifact fingerprint.
			if err := index.Validate(func(path string) ([]byte, error) {
				bytes, err := read(path)
				if err != nil {
					return nil, err
				}
				if path == changedPath {
					return append(bytes, byte('\n')), nil
				}
				return bytes, nil
			}); err == nil {
				t.Fatal("changed Source bytes unexpectedly passed old index")
			}
			invalid := sourceindex.Index{
				Artifacts:  append([]sourceindex.Artifact(nil), sources.Artifacts...),
				References: append([]sourceindex.Reference(nil), sources.References...),
			}
			invalid.References[0].Range.End.Offset = 1 << 30
			if err := invalid.Validate(read); err == nil {
				t.Fatal("out-of-bounds Source span unexpectedly passed index")
			}
			// A matching fingerprint and byte range still cannot certify
			// that the span semantically owns this relation or that a
			// current edit/acceptance grant exists.
		})
	}
}

// Source Index validation checks artifact bytes and locator structure, not
// whether the referenced byte range owns the declared semantic Subject.
// This negative control deliberately forges an in-memory Subject field and
// documents why an Index or trace PASS is insufficient for safe reverse edits.
func TestSourceIndexStructuralValidityDoesNotAuthenticateSemanticBinding(t *testing.T) {
	root := filepath.Join("..", "workspace")
	requested := sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.lines"}
	inspected, err := operations.InspectSubject(root, requested)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspected.Sources.References) == 0 || len(inspected.Sources.Artifacts) == 0 {
		t.Fatal("real source evidence required to exercise false-binding control")
	}
	forged := sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.nonexistent_target_for_negative_control"}
	refs := append([]sourceindex.Reference(nil), inspected.Sources.References...)
	refs[0].Subject = forged
	idx := sourceindex.Index{
		Artifacts:  append([]sourceindex.Artifact(nil), inspected.Sources.Artifacts...),
		References: refs,
	}
	read := func(path string) ([]byte, error) {
		clean, err := sourceindex.NormalizeWorkspacePath(path)
		if err != nil || clean != path {
			return nil, fmt.Errorf("non-normalized Source path %q: %w", path, err)
		}
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	}
	if err := idx.Validate(read); err != nil {
		t.Fatalf("byte-valid Source Index should not claim semantic-binding validation: %v", err)
	}
	fakeTrace, err := operations.TraceSubjectSourcesFromIndex(idx, forged)
	if err != nil {
		t.Fatal(err)
	}
	if len(fakeTrace.References) != 1 || fakeTrace.References[0].Subject != forged {
		t.Fatalf("expected deliberately forged Source trace to show structural false positive: %+v", fakeTrace)
	}
	_, err = operations.InspectSubject(root, forged)
	var inspection *operations.InspectionError
	if !errors.As(err, &inspection) || inspection.Code != operations.InspectionNotFound {
		t.Fatalf("actual Canonical World must reject forged Subject despite byte-valid trace: %v", err)
	}
	// The trace above cannot become Source edit authority. Consumers must
	// resolve the exact Canonical Subject/Relation and current acceptance/Grant
	// independently. No invented Schema, Writer, or Registry is introduced.
}

// M0 must project one real Parent -> Relation -> Target topology without
// inventing an accepted Business Flow or treating a Source trace as meaning.
// MapWorkspace uses a single Loader/Source Index internally. The subsequent
// Inspect calls are separate reads and cannot establish an atomic snapshot.
func TestRealWorkspaceModelMapMatchesSourceIndexedRelationTriplet(t *testing.T) {
	root := filepath.Join("..", "workspace")
	mapped, err := operations.MapWorkspace(root)
	if err != nil {
		t.Fatalf("read real TEI native typed Model Map: %v", err)
	}
	if mapped.SchemaVersion != "tei.operations.model-map/v0" ||
		mapped.Effect != "read-only" || mapped.Coverage != "partial-projection" ||
		mapped.Catalog.Workspace == nil || mapped.Catalog.WorldFingerprint == "" ||
		mapped.Catalog.SourceIndexFingerprint == "" {
		t.Fatalf("typed Model Map overclaimed coverage or lost Workspace evidence: %+v", mapped)
	}

	relation := sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.lines"}
	entries := 0
	var parent sourceindex.Subject
	for _, entry := range mapped.Catalog.Entries {
		if entry.Subject != relation {
			continue
		}
		entries++
		if entry.Ambiguous || entry.Prelude || entry.SourceStatus != "indexed" ||
			entry.Parent == nil || entry.Parent.Kind != "semantic-data" {
			t.Fatalf("real relation is not uniquely source-indexed: %+v", entry)
		}
		parent = *entry.Parent
	}
	if entries != 1 {
		t.Fatalf("expected one exact canonical relation entry, got %d", entries)
	}
	inspected, err := operations.InspectSubject(root, relation)
	if err != nil {
		t.Fatal(err)
	}
	var canonical struct {
		ID     string `json:"id"`
		Target string `json:"target"`
	}
	if err := json.Unmarshal(inspected.Canonical, &canonical); err != nil ||
		canonical.ID != relation.ID || canonical.Target == "" {
		t.Fatalf("real relation Canonical identity/target missing: %+v err=%v", canonical, err)
	}
	target := sourceindex.Subject{Kind: "semantic-data", ID: canonical.Target}
	if inspected.WorldFingerprint != mapped.Catalog.WorldFingerprint ||
		inspected.SourceIndexFingerprint != mapped.Catalog.SourceIndexFingerprint ||
		inspected.Workspace == nil ||
		!reflect.DeepEqual(*inspected.Workspace, *mapped.Catalog.Workspace) {
		t.Fatal("the M0 projection and canonical relation observation differ in Workspace/revision")
	}

	for _, subject := range []sourceindex.Subject{parent, relation, target} {
		observed, err := operations.InspectSubject(root, subject)
		if err != nil {
			t.Fatalf("one endpoint cannot be resolved as canonical TEI Source: %+v: %v", subject, err)
		}
		if observed.Subject != subject || observed.SourceStatus != "indexed" ||
			len(observed.Sources.References) == 0 ||
			observed.WorldFingerprint != mapped.Catalog.WorldFingerprint ||
			observed.SourceIndexFingerprint != mapped.Catalog.SourceIndexFingerprint {
			t.Fatalf("Model Map endpoint lacks matching canonical Source evidence: %+v", subject)
		}
	}

	for _, expected := range []operations.ModelMapLink{
		{From: parent, To: relation, Relation: "contains", Resolution: "resolved"},
		{From: relation, To: target, Relation: "refers-to", Resolution: "resolved"},
	} {
		matches := 0
		for _, got := range mapped.Links {
			if got.From == expected.From && got.To == expected.To && got.Relation == expected.Relation {
				matches++
				if got.Resolution != "resolved" {
					t.Fatalf("actual indexed relation rendered as ambiguous/unresolved: %+v", got)
				}
			}
		}
		if matches != 1 {
			t.Fatalf("exact native Model Map edge must appear once: edge=%+v count=%d", expected, matches)
		}
	}
	// In M0 "resolved" means unique declared endpoints, never Owner
	// acceptance, required-world closure, Plugin Invoke, authorized effect,
	// safe inverse editing, complete CRUD, or external Consumer receipt.
}

// The native Model Map and the already-shipped model locate operation are
// independent read-only projections. A UI selection must recover the same
// qualified semantic Relation, its parent Data and its declared target.
// These separate Loads are observations, not an atomic source snapshot.
func TestRealWorkspaceTypedModelMapAndModelLocateAgreeOnRelation(t *testing.T) {
	root := filepath.Join("..", "workspace")
	relation := sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.lines"}
	mapped, err := operations.MapWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	located, err := operations.LocateWorkspace(root, relation, 1)
	if err != nil {
		t.Fatal(err)
	}
	if mapped.Coverage != "partial-projection" || located.Subject.Subject != relation ||
		located.Subject.SourceStatus != "indexed" || located.Effect != "read-only" ||
		located.Workspace == nil || mapped.Catalog.Workspace == nil ||
		!reflect.DeepEqual(*located.Workspace, *mapped.Catalog.Workspace) ||
		located.WorldFingerprint != mapped.Catalog.WorldFingerprint ||
		located.SourceIndexFingerprint != mapped.Catalog.SourceIndexFingerprint ||
		located.ModelRevision != mapped.Catalog.ModelRevision {
		t.Fatalf("model locate/map disagree on exact selected Source or version: %+v", located)
	}
	if len(located.Path) == 0 {
		t.Fatal("model locate lost the selected relation's parent path")
	}
	parent := located.Path[len(located.Path)-1].Subject
	if parent.Kind != "semantic-data" {
		t.Fatalf("last parent must be Data, got %+v", parent)
	}
	if len(located.Outgoing) != 1 ||
		located.Outgoing[0].Role != operations.LinkRoleRelationTarget ||
		located.Outgoing[0].From != relation || !located.Outgoing[0].Resolved {
		t.Fatalf("selected relation must expose one resolved target: %+v", located.Outgoing)
	}
	target := located.Outgoing[0].To
	if target.Kind != "semantic-data" {
		t.Fatalf("relation target must be Data, got %+v", target)
	}
	for _, expected := range []operations.ModelMapLink{
		{From: parent, To: relation, Relation: "contains", Resolution: "resolved"},
		{From: relation, To: target, Relation: "refers-to", Resolution: "resolved"},
	} {
		matches := 0
		for _, link := range mapped.Links {
			if link.From == expected.From && link.To == expected.To &&
				link.Relation == expected.Relation && link.Resolution == expected.Resolution {
				matches++
			}
		}
		if matches != 1 {
			t.Fatalf("selection navigation must recover exactly one typed map link: %+v count=%d", expected, matches)
		}
	}
	missing := false
	for _, family := range located.Coverage.NotDerived {
		if family == "query-data-dependency" {
			missing = true
		}
	}
	if !missing || located.Limit != 1 {
		t.Fatalf("bounded navigation must disclose missing dependency families: %+v", located.Coverage)
	}
	// Navigation agreement does not provide complete Function/Business/CRUD,
	// edit authority, full disclosure, external Provider freshness or receipt.
}

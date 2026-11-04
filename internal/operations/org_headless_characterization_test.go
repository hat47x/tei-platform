package operations

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/workspace"
)

// This intentionally consumes actual .tei source and the existing TEI loader,
// rather than constructing an independent JSON-only organization toy model.
// It is NOT a test of SQL execution, temporal rules, Host grants or data effects.
func TestOrgHeadlessAuthoringToOperations(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-headless-workspace")
	surface, err := SurfaceWorkspace(root, "org_view")
	if err != nil {
		t.Fatalf("load native organization workspace and derive operations: %v", err)
	}
	if surface.WorldFingerprint == "" || surface.ModelRevision == "" {
		t.Fatalf("missing actual workspace-derived Source/World identity: %+v", surface)
	}
	if surface.Effect != "read-only" || len(surface.AuthorityLimits) == 0 {
		t.Fatalf("operation surface must not claim Host authority: %+v", surface)
	}

	var list, read *Operation
	for i := range surface.Operations {
		op := &surface.Operations[i]
		switch op.ID {
		case "org_view.list.organization.membership":
			list = op
		case "org_view.read.organization.membership":
			read = op
		default:
			t.Fatalf("read-only application derived an unexpected operation: %+v", *op)
		}
	}
	if list == nil || read == nil || len(surface.Operations) != 2 {
		t.Fatalf("expected list and read of distinct Membership occurrences: %+v", surface.Operations)
	}
	if list.Kind != "list" || read.Kind != "read" || read.Autonomy != "unattended" {
		t.Fatalf("wrong operation semantics: list=%+v read=%+v", list, read)
	}
	if len(read.Inputs) != 1 || read.Inputs[0] != "organization.membership.membership_id" {
		t.Fatalf("read must use Membership occurrence identity, not Person/OrgUnit: %+v", read.Inputs)
	}
	if len(read.Effects) != 0 || read.Binding != "" || len(surface.Gaps) != 0 {
		t.Fatalf("read-only operation must not invent change authority: %+v", surface)
	}
}

func TestOrgHeadlessSeparateMembershipFromOccupancy(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-headless-workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("parse native organization .tei source: %v", err)
	}
	var membership, occupancy bool
	for _, ns := range loaded.World.Semantics {
		for _, data := range ns.Data {
			if data.ID != "organization.membership" && data.ID != "organization.occupancy" {
				continue
			}
			targets := map[string]bool{}
			for _, rel := range data.Relations {
				if rel.Cardinality != "one" || len(rel.Keys) != 1 {
					t.Fatalf("reference relation lost its explicitly authored target/key: %+v", rel)
				}
				targets[string(rel.Target)] = true
			}
			if data.ID == "organization.membership" {
				membership = targets["organization.person"] && targets["organization.org_unit"] && len(targets) == 2
			} else {
				occupancy = targets["organization.person"] && targets["organization.position"] && len(targets) == 2
			}
		}
	}
	if !membership || !occupancy {
		t.Fatal("native semantic model collapsed Membership and Occupancy")
	}
}

// CaseRecord deliberately has only a *semantic relation* to OrgUnit:
// no Persistence mapping, no Policy/Grant binding, no derived Case read/write.
// A relational path being authorable is not a runtime Row Level Security claim.
func TestOrgHeadlessCaseRelationDoesNotAuthorizeReadOrWrite(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-headless-workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load real .tei workspace with CaseRecord: %v", err)
	}
	var found bool
	for _, ns := range loaded.World.Semantics {
		for _, data := range ns.Data {
			if data.ID != "organization.case_record" {
				continue
			}
			found = true
			if len(data.Relations) != 1 {
				t.Fatalf("CaseRecord must declare one typed OrgUnit relation: %+v", data.Relations)
			}
			rel := data.Relations[0]
			if rel.Target != "organization.org_unit" || rel.Cardinality != "one" || len(rel.Keys) != 1 ||
				rel.Keys[0].SourceAttribute != "organization.case_record.handling_unit_id" ||
				rel.Keys[0].TargetAttribute != "organization.org_unit.org_unit_id" {
				t.Fatalf("CaseRecord→OrgUnit must preserve explicitly authored key identity: %+v", rel)
			}
		}
	}
	if !found {
		t.Fatal("CaseRecord did not survive actual .tei Parser/Loader")
	}
	surface, err := SurfaceWorkspace(root, "org_view")
	if err != nil {
		t.Fatalf("derive read-only operation surface: %v", err)
	}
	if surface.Effect != "read-only" || len(surface.Operations) != 2 {
		t.Fatalf("unpersisted CaseRecord relationship must not create operations/grants: %+v", surface)
	}
	for _, op := range surface.Operations {
		if op.Target == "organization.case_record" || len(op.Effects) != 0 || op.Binding != "" {
			t.Fatalf("CaseRecord OrgUnit relation must not imply read, edit or Host authorization: %+v", op)
		}
	}
}

// A source-only change can leave the normalized semantic World unchanged while
// changing the exact managed Source revision. A visible/read permission already
// granted for a prior Source must not be renewed merely from equal operation JSON.
// This test uses a real copy of .tei authoring, the existing Loader and SourceIndex.
// It does NOT model real physical Provider access, Policy enforcement or Host Grant.
func TestOrgHeadlessSourceByteEditPreservesWorldButChangesModelRevision(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-headless-workspace")
	before, err := SurfaceWorkspace(root, "org_view")
	if err != nil {
		t.Fatalf("load baseline organization workspace: %v", err)
	}
	copied := t.TempDir()
	if err := os.CopyFS(copied, os.DirFS(root)); err != nil {
		t.Fatalf("copy actual workspace Source: %v", err)
	}
	src := filepath.Join(copied, "semantic", "organization", "model.tei")
	original, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read copied organization model Source: %v", err)
	}
	commented := append(append([]byte(nil), original...), []byte("\n// source-only provenance edit; no accepted semantic change\n")...)
	if err := os.WriteFile(src, commented, 0o644); err != nil {
		t.Fatalf("write exact-source comment edit: %v", err)
	}
	after, err := SurfaceWorkspace(copied, "org_view")
	if err != nil {
		t.Fatalf("reparse comment-edited organization workspace: %v", err)
	}
	if before.WorldFingerprint != after.WorldFingerprint {
		t.Fatalf("comment-only Source edit changed normalized World fingerprint: %s -> %s",
			before.WorldFingerprint, after.WorldFingerprint)
	}
	if before.ModelRevision == after.ModelRevision {
		t.Fatalf("comment-only Source edit must change managed Source revision: %s", before.ModelRevision)
	}
	if !reflect.DeepEqual(before.Operations, after.Operations) {
		t.Fatalf("a comment-only Source edit must not create or change operations: before=%+v after=%+v",
			before.Operations, after.Operations)
	}
	if after.Effect != "read-only" || len(after.AuthorityLimits) == 0 {
		t.Fatalf("same World fingerprint and operation set cannot create action authority: %+v", after)
	}
}
// The workspace's logical meaning and operation IDs survive a path move of one
// fully parsed .tei source. SourceIndex ModelRevision still records the changed
// source path. This is a local Workspace test, not an external Import or atomic
// multi-file snapshot proof.
func TestOrgHeadlessSourcePathMoveChangesModelRevision(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-headless-workspace")
	before, err := SurfaceWorkspace(root, "org_view")
	if err != nil {
		t.Fatalf("load original organization workspace: %v", err)
	}
	copyRoot := t.TempDir()
	if err := os.CopyFS(copyRoot, os.DirFS(root)); err != nil {
		t.Fatalf("copy actual organization Source: %v", err)
	}
	from := filepath.Join(copyRoot, "semantic", "organization", "model.tei")
	to := filepath.Join(copyRoot, "semantic", "organization", "model-relocated.tei")
	if err := os.Rename(from, to); err != nil {
		t.Fatalf("relocate parsed .tei Source within managed root: %v", err)
	}
	after, err := SurfaceWorkspace(copyRoot, "org_view")
	if err != nil {
		t.Fatalf("load organization workspace after Source path move: %v", err)
	}
	if before.WorldFingerprint != after.WorldFingerprint {
		t.Fatalf("path-only move changed normalized World: before=%s after=%s",
			before.WorldFingerprint, after.WorldFingerprint)
	}
	if before.ModelRevision == after.ModelRevision {
		t.Fatal("a change to managed Source path must change the exact Source ModelRevision")
	}
	if !reflect.DeepEqual(before.Operations, after.Operations) {
		t.Fatalf("path-only move changed actual read-only operations: before=%+v after=%+v",
			before.Operations, after.Operations)
	}
}

// A->B->A restores both currently observed fingerprints. This test explicitly
// demonstrates that fingerprints are *state identifiers*, not revision history,
// event receipts, currently valid grants, or proof no intermediate change occurred.
func TestOrgHeadlessSourceABAReturnsToOriginalStateButNotHistory(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-headless-workspace")
	copyRoot := t.TempDir()
	if err := os.CopyFS(copyRoot, os.DirFS(root)); err != nil {
		t.Fatalf("copy original .tei workspace: %v", err)
	}
	source := filepath.Join(copyRoot, "semantic", "organization", "model.tei")
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read original .tei Source: %v", err)
	}
	a, err := SurfaceWorkspace(copyRoot, "org_view")
	if err != nil {
		t.Fatalf("load A: %v", err)
	}
	changed := append(append([]byte(nil), original...), []byte("\n// temporary B revision\n")...)
	if err := os.WriteFile(source, changed, 0o644); err != nil {
		t.Fatalf("write B: %v", err)
	}
	b, err := SurfaceWorkspace(copyRoot, "org_view")
	if err != nil {
		t.Fatalf("load B: %v", err)
	}
	if err := os.WriteFile(source, original, 0o644); err != nil {
		t.Fatalf("restore original Source C: %v", err)
	}
	c, err := SurfaceWorkspace(copyRoot, "org_view")
	if err != nil {
		t.Fatalf("load restored A: %v", err)
	}
	if a.WorldFingerprint != b.WorldFingerprint || a.WorldFingerprint != c.WorldFingerprint {
		t.Fatal("comment-only ABA must not change normalized semantic World")
	}
	if a.ModelRevision == b.ModelRevision || a.ModelRevision != c.ModelRevision {
		t.Fatalf("Source identity must be A != B and A == C: %s / %s / %s",
			a.ModelRevision, b.ModelRevision, c.ModelRevision)
	}
	if !reflect.DeepEqual(a.Operations, b.Operations) || !reflect.DeepEqual(a.Operations, c.Operations) {
		t.Fatal("comment-only ABA must not create or change read-only operations")
	}
	// The temporal observation (three actual Load calls) is recorded by this
	// test's own sequence, not inferred from the equal A/C fingerprints.
}

// A declared CaseRecord hierarchy link is *meaningful* even though the
// read-only Membership OperationSurface does not mention the unpersisted Case.
// Comparing only operation IDs/JSON would miss a changed Relation cardinality.
func TestOrgHeadlessSemanticRelationChangeIsInvisibleToCurrentOperations(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-headless-workspace")
	before, err := SurfaceWorkspace(root, "org_view")
	if err != nil {
		t.Fatalf("load original semantic relation: %v", err)
	}
	copyRoot := t.TempDir()
	if err := os.CopyFS(copyRoot, os.DirFS(root)); err != nil {
		t.Fatalf("copy actual .tei source: %v", err)
	}
	source := filepath.Join(copyRoot, "semantic", "organization", "model.tei")
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	oldDecl := "relation handlingUnit to organization.org_unit one"
	newDecl := "relation handlingUnit to organization.org_unit many"
	if strings.Count(string(raw), oldDecl) != 1 {
		t.Fatal("expected precisely one typed CaseRecord relation occurrence")
	}
	changed := strings.Replace(string(raw), oldDecl, newDecl, 1)
	if err := os.WriteFile(source, []byte(changed), 0o644); err != nil {
		t.Fatalf("write changed relation declaration: %v", err)
	}
	after, err := SurfaceWorkspace(copyRoot, "org_view")
	if err != nil {
		t.Fatalf("load modified semantic relation: %v", err)
	}
	if before.WorldFingerprint == after.WorldFingerprint {
		t.Fatal("different CaseRecord Relation cardinality must change semantic World")
	}
	if before.ModelRevision == after.ModelRevision {
		t.Fatal("different Relation declaration must change managed Source ModelRevision")
	}
	if !reflect.DeepEqual(before.Operations, after.Operations) {
		t.Fatalf("unpersisted CaseRecord Relation should not change Membership OperationSurface: before=%+v after=%+v",
			before.Operations, after.Operations)
	}
	if after.Effect != "read-only" || len(after.AuthorityLimits) == 0 {
		t.Fatal("a changed semantic Relation must not create an authorization Grant")
	}
}

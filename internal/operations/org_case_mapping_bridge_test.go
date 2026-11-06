package operations

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/workspace"
)

// This is an actual TEI workspace/Parser/Loader/OperationSurface characterization,
// using a copied native .tei source. It does NOT call a Function, SQL Provider,
// runtime Host grant or Consumer and is not a Product E2 pass.
func TestOrgCaseExplicitPersistenceEnablesReadCandidateOnly(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-headless-workspace")
	before, err := SurfaceWorkspace(root, "org_view")
	if err != nil {
		t.Fatalf("load baseline native org workspace: %v", err)
	}
	if len(before.Operations) != 2 {
		t.Fatalf("baseline should have Membership-only list/read: %+v", before.Operations)
	}
	for _, op := range before.Operations {
		if op.Target == "organization.case_record" {
			t.Fatalf("a structural CaseRecord relation cannot generate a read without Persistence: %+v", op)
		}
	}

	copyRoot := t.TempDir()
	if err := os.CopyFS(copyRoot, os.DirFS(root)); err != nil {
		t.Fatalf("copy real native org workspace: %v", err)
	}

	// Explicitly add only the physical mapping, not a Function, Policy or Grant.
	// The read-only Application already uses this Database. This deliberately
	// separates "Operation is derivable" from "there is an actual Provider read".
	mapping := `persist org_store.case_record {
    database org_store
    semantic organization.case_record
    table public.org_case_records

    column organization.case_record.case_id -> case_id storage varchar required
    column organization.case_record.handling_unit_id -> handling_unit_id storage varchar required

    primary-key organization.case_record.case_id
}
`
	file := filepath.Join(copyRoot, "databases", "org-store", "persistence", "case_record.tei")
	if err := os.WriteFile(file, []byte(mapping), 0o644); err != nil {
		t.Fatalf("author explicit CaseRecord Persistence Mapping: %v", err)
	}

	loaded, err := workspace.Load(copyRoot)
	if err != nil {
		t.Fatalf("actual TEI Loader rejected mapped CaseRecord: %v", err)
	}
	var mappingFound bool
	for _, item := range loaded.World.Persistence {
		if item.SemanticData == "organization.case_record" {
			mappingFound = true
			if item.Database != "org_store" || item.Table != "public.org_case_records" ||
				len(item.PrimaryKey) != 1 || item.PrimaryKey[0] != "organization.case_record.case_id" {
				t.Fatalf("CaseRecord mapping or key lost: %+v", item)
			}
		}
	}
	if !mappingFound {
		t.Fatal("native Loader did not consume the extra CaseRecord Persistence Mapping")
	}

	after, err := SurfaceWorkspace(copyRoot, "org_view")
	if err != nil {
		t.Fatalf("derive operations from actual mapped CaseRecord: %v", err)
	}
	if before.WorldFingerprint == after.WorldFingerprint || before.ModelRevision == after.ModelRevision {
		t.Fatalf("physical mapping must affect observed Source/World identity: before=%+v after=%+v", before, after)
	}
	if after.Effect != "read-only" || len(after.AuthorityLimits) == 0 ||
		!reflect.DeepEqual(after.AuthorityLimits, before.AuthorityLimits) {
		t.Fatalf("new mapping must not create an execution authority: %+v", after)
	}

	seen := map[string]bool{}
	for _, op := range after.Operations {
		seen[op.ID] = true
		if op.Binding != "" || len(op.Effects) != 0 {
			t.Fatalf("read-only mapping must not grant write or Provider execution: %+v", op)
		}
		if op.Target == "organization.case_record" {
			if op.Origin != "derived" || op.Autonomy != "unattended" || op.Basis == "" {
				t.Fatalf("mapped CaseRecord may be a read-only candidate, not a runtime invocation: %+v", op)
			}
			if op.Kind == "read" && !reflect.DeepEqual(
				op.Inputs, []string{"organization.case_record.case_id"},
			) {
				t.Fatalf("Case read must target its own business identity, not OrgUnit or view-row index: %+v", op)
			}
		}
	}
	for _, name := range []string{
		"org_view.list.organization.membership",
		"org_view.read.organization.membership",
		"org_view.list.organization.case_record",
		"org_view.read.organization.case_record",
	} {
		if !seen[name] {
			t.Errorf("missing derived operation %s; got %+v", name, after.Operations)
		}
	}
	if len(after.Operations) != 4 || len(seen) != 4 {
		t.Fatalf("mapping one CaseRecord must add exactly list/read, not extra permissions: %+v", after.Operations)
	}
}

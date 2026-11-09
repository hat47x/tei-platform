package operations

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	querymodel "tei/internal/query"
	"tei/internal/workspace"
)

// This is an actual TEI workspace/Parser/Loader/OperationSurface characterization,
// using a copied native .tei source. It calls the real Driver/Executor with
// a fake SQL Connector and offers opt-in isolated PostgreSQL execution. It
// does NOT establish Function Data Context, Host authorization, Consumer
// delivery, or Product E2 acceptance.
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
	// E1c pressure: an authored Query and native SQL file are *separate* source
	// contributions. They can be parsed and traced without a bound datasource,
	// a Function Data Context, a real Provider call or an authorized consumer.
	queryDir := filepath.Join(copyRoot, "applications", "org-view", "queries")
	if err := os.MkdirAll(queryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	querySource := `query org_view.case_read {
    input case_id organization.case_record_id
    datasource org_read
    sql postgres file "case_read.sql"
    result {
        column case_id -> organization.case_record.case_id
        column handling_unit_id -> organization.case_record.handling_unit_id
    }
}
`
	sqlSource := "select case_id, handling_unit_id from public.org_case_records where case_id = :case_id\n"
	queryFile := filepath.Join(queryDir, "case_read.tei")
	sqlFile := filepath.Join(queryDir, "case_read.sql")
	if err := os.WriteFile(queryFile, []byte(querySource), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sqlFile, []byte(sqlSource), 0o644); err != nil {
		t.Fatal(err)
	}
	withQuery, err := workspace.Load(copyRoot)
	if err != nil {
		t.Fatalf("native Loader did not accept authored CaseRecord Query: %v", err)
	}
	if len(withQuery.World.Queries) != 1 {
		t.Fatalf("expected exactly one authored Case query: %+v", withQuery.World.Queries)
	}
	query := withQuery.World.Queries[0]
	if query.ID != "org_view.case_read" || query.Application != "org_view" ||
		query.Datasource != "org_read" || query.Source != strings.TrimSpace(sqlSource) ||
		len(query.Inputs) != 1 || query.Inputs[0].Name != "case_id" ||
		query.Inputs[0].Type != "organization.case_record_id" {
		t.Fatalf("Case Query Source/Inputs/Datasource were not faithfully loaded: %+v", query)
	}
	if !reflect.DeepEqual(query.ResultColumns, []querymodel.ResultColumnMapping{
		{Column: "case_id", Attribute: "organization.case_record.case_id"},
		{Column: "handling_unit_id", Attribute: "organization.case_record.handling_unit_id"},
	}) {
		t.Fatalf("Case Query result columns are not typed Semantic references: %+v", query.ResultColumns)
	}
	if loc, ok := withQuery.Sources["query:org_view.case_read"]; !ok ||
		filepath.ToSlash(loc.File) != "applications/org-view/queries/case_read.tei" {
		t.Fatalf("authored Query source location missing: %+v", withQuery.Sources)
	}
	var sqlContributed bool
	for _, contribution := range withQuery.SourceContributions {
		if contribution.Subject.Kind == "query" &&
			contribution.Subject.ID == "org_view.case_read" &&
			filepath.ToSlash(contribution.File) == "applications/org-view/queries/case_read.sql" {
			sqlContributed = true
		}
	}
	if !sqlContributed {
		t.Fatal("native SQL file is not traced as a separate Query source contribution")
	}

	// E1d: real native Query bytes -> selected datasource -> existing PostgreSQL
	// Executor -> raw physical columns -> explicit Semantic result mapping.
	// The injected connection is an instrumented test double (no real DB).
	verifyNativeCaseQuerySQLProbe(t, withQuery)

	// The Query may be described, but it does not automatically create an
	// executable Function or grant Provider/Host authority.
	surfaceWithQuery, err := SurfaceWorkspace(copyRoot, "org_view")
	if err != nil {
		t.Fatal(err)
	}
	if surfaceWithQuery.ModelRevision == after.ModelRevision ||
		!reflect.DeepEqual(surfaceWithQuery.Operations, after.Operations) ||
		!reflect.DeepEqual(surfaceWithQuery.AuthorityLimits, after.AuthorityLimits) {
		t.Fatalf("Query must affect revision without promoting read candidates to execution: before=%+v query=%+v", after, surfaceWithQuery)
	}

	// E1f: make a Function-facing Browse Pattern explicit in the SAME copied
	// TEI Workspace. A Query declaration alone is not callable by a Pattern,
	// and the Pattern is not a Host Grant or a full Function Data Context.
	patternDir := filepath.Join(copyRoot, "applications", "org-view", "patterns")
	if err := os.MkdirAll(patternDir, 0o755); err != nil {
		t.Fatal(err)
	}
	patternFile := filepath.Join(patternDir, "case-browse.tei")
	const patternSource = `pattern org_view.case_browse browse {
    data organization.case_record
    query org_view.case_read
}
`
	if err := os.WriteFile(patternFile, []byte(patternSource), 0o644); err != nil {
		t.Fatal(err)
	}
	withPattern, err := workspace.Load(copyRoot)
	if err != nil {
		t.Fatalf("native Case browse Pattern could not be loaded: %v", err)
	}
	if loc, ok := withPattern.Sources["application-pattern:org_view.case_browse"]; !ok ||
		filepath.ToSlash(loc.File) != "applications/org-view/patterns/case-browse.tei" {
		t.Fatalf("browse Pattern is not bound to its exact source: %+v", withPattern.Sources)
	}
	verifyNativeCasePatternReadProbe(t, withPattern)

	// The same native Browse Pattern can cross the real PostgreSQL boundary
	// only with an explicitly isolated, opt-in database. Default is SKIP;
	// this is not current Host authorization, nor a Function Data Context.
	t.Run("isolated-postgres-opt-in", func(t *testing.T) {
		verifyNativeCaseReadWithPostgres(t, withPattern)
	})
	// The read Pattern adds a source/meaning revision but never derives new
	// Case grants or write operations merely from its presence.
	surfaceWithPattern, err := SurfaceWorkspace(copyRoot, "org_view")
	if err != nil {
		t.Fatal(err)
	}
	if surfaceWithPattern.ModelRevision == surfaceWithQuery.ModelRevision ||
		!reflect.DeepEqual(surfaceWithPattern.Operations, surfaceWithQuery.Operations) ||
		!reflect.DeepEqual(surfaceWithPattern.AuthorityLimits, surfaceWithQuery.AuthorityLimits) {
		t.Fatalf("browse Pattern must not become a new Host capability: before=%+v after=%+v",
			surfaceWithQuery, surfaceWithPattern)
	}

	// A native SQL predicate change must invalidate the Source/Model revision
	// even though the Semantic Query ID, datasource alias and mapping stay put.
	if err := os.WriteFile(sqlFile, []byte(strings.Replace(sqlSource,
		"where case_id = :case_id", "where case_id = :case_id and handling_unit_id is not null", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	sqlChanged, err := SurfaceWorkspace(copyRoot, "org_view")
	if err != nil {
		t.Fatal(err)
	}
	if sqlChanged.ModelRevision == surfaceWithPattern.ModelRevision ||
		sqlChanged.WorldFingerprint == surfaceWithPattern.WorldFingerprint {
		t.Fatalf("native SQL source change must invalidate the read-only world's revision: before=%s after=%s",
			surfaceWithPattern.ModelRevision, sqlChanged.ModelRevision)
	}
}

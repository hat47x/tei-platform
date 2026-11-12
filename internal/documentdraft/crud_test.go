package documentdraft

import (
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

// crudFixture is one synthetic Workspace with an application, a mapped data
// element and a capability that declares an effect. It is written to disk so the
// projection runs through the real loader and Source Index, exactly as the CLI
// does. It uses no customer data and no reference/workspace content.
func crudFixture(t *testing.T) fixture {
	t.Helper()
	base := newFixture(t)
	write := func(relative, body string) {
		t.Helper()
		writeFile(t, base.root, relative, []byte(body))
	}
	write("tei.workspace.yaml", "workspace: crud_fixture\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n  - databases\n")
	write("semantic/sales/crud.tei", `namespace sales

capability sales.order.create {
    input order sales.order
    output sales.order
    effect create sales.order
}
`)
	write("applications/order-entry/application.tei", `application order_entry {
    uses semantic sales

    database sales {
        target sales_core
        access read-write
    }
}
`)
	// The CRUD matrix is derived from the database alias, its persistence
	// mappings and the declared capability effects, so this fixture needs no
	// pattern: the derivation reads the same accepted meaning, not a screen.
	write("databases/sales-core/database.tei", "database sales_core {\n    kind relational\n    ownership managed-here\n    uses semantic sales\n}\n")
	write("databases/sales-core/persistence/order.tei", `persist sales_core.order {
    database sales_core
    semantic sales.order
    table public.orders
    column sales.order.order_id -> order_id storage varchar required
    column sales.order.customer_id -> customer_id storage varchar required

    primary-key sales.order.order_id
}
`)
	return base
}

// TestBuildCRUDTableProjectsTheDerivedOperations is the first-pass proof for the
// crud-table stage: one application and its mapped data become a draft whose
// rows are the operations the accepted model derives.
func TestBuildCRUDTableProjectsTheDerivedOperations(t *testing.T) {
	fixture := crudFixture(t)
	draft, err := Build(Request{
		Root:         fixture.root,
		DocumentKind: DocumentKindCRUDTable,
		Application:  "order_entry",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if draft.DocumentKind != DocumentKindCRUDTable {
		t.Fatalf("documentKind=%q", draft.DocumentKind)
	}
	if draft.Scope == nil || draft.Scope.Application != "order_entry" ||
		draft.Scope.CRUDRule != "tei.operations.operation-surface/v0" {
		t.Fatalf("scope=%+v", draft.Scope)
	}
	if len(draft.Subjects) < 2 {
		t.Fatalf("subjects=%+v", draft.Subjects)
	}
	if draft.Subjects[0].Subject != (sourceindex.Subject{Kind: "application", ID: "order_entry"}) {
		t.Fatalf("first subject=%+v", draft.Subjects[0])
	}
	// The data element the rows target is a subject too, so the document says
	// which elements it talks about without re-deriving them.
	foundData := false
	for _, subject := range draft.Subjects {
		if subject.Subject == (sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}) {
			foundData = true
		}
	}
	if !foundData {
		t.Fatalf("the mapped data is missing from the subject list: %+v", draft.Subjects)
	}
	table, ok := occurrenceByID(draft, "occ-table-1")
	if !ok || table.Content.Table == nil {
		t.Fatalf("CRUD matrix missing: %+v", draft.Occurrences)
	}
	if len(table.Content.Table.Header) != 6 {
		t.Fatalf("header=%v", table.Content.Table.Header)
	}
	if len(table.Content.Table.Rows) == 0 {
		t.Fatal("CRUD matrix has no row")
	}
	if len(table.Content.Table.RowSubjects) != len(table.Content.Table.Rows) {
		t.Fatalf("row subjects=%d rows=%d",
			len(table.Content.Table.RowSubjects), len(table.Content.Table.Rows))
	}
	kinds := map[string]bool{}
	for _, row := range table.Content.Table.Rows {
		kinds[row[0]] = true
	}
	for _, want := range []string{"list", "read", "create", "update"} {
		if !kinds[want] {
			t.Fatalf("derived operation %q is missing from the matrix: %+v", want, table.Content.Table.Rows)
		}
	}
	// The row subjects name the data and the application, and never a guessed
	// element: every data subject in a row must resolve in this Workspace.
	for _, subjects := range table.Content.Table.RowSubjects {
		if len(subjects) != 2 {
			t.Fatalf("row subjects=%+v", subjects)
		}
		if _, err := fixture.readSubject(subjects[0]); err != nil {
			t.Fatalf("row subject %+v does not resolve: %v", subjects[0], err)
		}
	}
	for _, occurrence := range draft.Occurrences {
		if !strings.HasPrefix(occurrence.ContentDigest, "sha256:") {
			t.Fatalf("occurrence %s has no digest", occurrence.OccurrenceID)
		}
		if occurrence.GenerationConditions.Rule == "" {
			t.Fatalf("occurrence %s has no rule", occurrence.OccurrenceID)
		}
	}
	if len(draft.AuthorityLimits) == 0 {
		t.Fatal("draft states no authority limit")
	}
	for _, limit := range draft.AuthorityLimits {
		if limit == "reprojection-never-overwrites-author-text" {
			return
		}
	}
	t.Fatalf("authorityLimits=%v", draft.AuthorityLimits)
}

// TestBuildCRUDTableIsDeterministic pins that the same read produces the same
// draft and the same row order.
func TestBuildCRUDTableIsDeterministic(t *testing.T) {
	fixture := crudFixture(t)
	first, err := Build(Request{Root: fixture.root, DocumentKind: DocumentKindCRUDTable, Application: "order_entry"})
	if err != nil {
		t.Fatalf("first Build: %v", err)
	}
	second, err := Build(Request{Root: fixture.root, DocumentKind: DocumentKindCRUDTable, Application: "order_entry"})
	if err != nil {
		t.Fatalf("second Build: %v", err)
	}
	if first.DraftID != second.DraftID {
		t.Fatalf("draftId changed: %q vs %q", first.DraftID, second.DraftID)
	}
	table1, _ := occurrenceByID(first, "occ-table-1")
	table2, _ := occurrenceByID(second, "occ-table-1")
	if len(table1.Content.Table.Rows) != len(table2.Content.Table.Rows) {
		t.Fatalf("row counts differ: %d vs %d",
			len(table1.Content.Table.Rows), len(table2.Content.Table.Rows))
	}
	for i := range table1.Content.Table.Rows {
		if strings.Join(table1.Content.Table.Rows[i], "|") != strings.Join(table2.Content.Table.Rows[i], "|") {
			t.Fatalf("row %d differs: %v vs %v", i, table1.Content.Table.Rows[i], table2.Content.Table.Rows[i])
		}
	}
}

// TestBuildCRUDTableRejectsAnUnknownApplication keeps an unknown filter from
// reading as "this application does nothing": the operation surface would return
// an empty result for it.
func TestBuildCRUDTableRejectsAnUnknownApplication(t *testing.T) {
	fixture := crudFixture(t)
	_, err := Build(Request{Root: fixture.root, DocumentKind: DocumentKindCRUDTable, Application: "no_such_app"})
	if err == nil {
		t.Fatal("Build accepted an application the workspace does not declare")
	}
	var draftErr *DraftError
	if !errorsAsDraft(err, &draftErr) || draftErr.Code != CodeApplicationNotFound {
		t.Fatalf("code=%v (%v)", err, draftErr)
	}
}

// TestBuildCRUDTableRejectsMixedInputs keeps the two input shapes apart: a CRUD
// table is about an application, so an element selector, a relation or a
// missing application is refused instead of silently ignored.
func TestBuildCRUDTableRejectsMixedInputs(t *testing.T) {
	fixture := crudFixture(t)
	cases := map[string]Request{
		"missing application": {Root: fixture.root, DocumentKind: DocumentKindCRUDTable},
		"blank application":   {Root: fixture.root, DocumentKind: DocumentKindCRUDTable, Application: " "},
		"element selector": {Root: fixture.root, DocumentKind: DocumentKindCRUDTable,
			Application: "order_entry", Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}},
		"relation": {Root: fixture.root, DocumentKind: DocumentKindCRUDTable,
			Application: "order_entry", RelationID: "sales.order.customer"},
	}
	for name, request := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Build(request)
			if err == nil {
				t.Fatalf("Build accepted %+v", request)
			}
			var draftErr *DraftError
			if !errorsAsDraft(err, &draftErr) || draftErr.Code != CodeInvalidArgument {
				t.Fatalf("code=%v want %q", err, CodeInvalidArgument)
			}
		})
	}
}

// TestBuildElementKindRejectsAnApplicationFlag pins the other direction: a kind
// that does not use -application refuses it rather than ignoring it.
func TestBuildElementKindRejectsAnApplicationFlag(t *testing.T) {
	fixture := crudFixture(t)
	_, err := Build(Request{
		Root: fixture.root, Subject: orderSubject(), Application: "order_entry",
	})
	if err == nil {
		t.Fatal("an element draft accepted -application")
	}
	var draftErr *DraftError
	if !errorsAsDraft(err, &draftErr) || draftErr.Code != CodeInvalidArgument {
		t.Fatalf("code=%v want %q", err, CodeInvalidArgument)
	}
}

// TestCRUDTableReconcilesPerOccurrence pins that a changed model moves only the
// occurrences it actually affects.
func TestCRUDTableReconcilesPerOccurrence(t *testing.T) {
	fixture := crudFixture(t)
	draft, err := Build(Request{Root: fixture.root, DocumentKind: DocumentKindCRUDTable, Application: "order_entry"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if report.ModelReference.Status != ModelReferenceCurrent {
		t.Fatalf("model reference=%+v", report.ModelReference)
	}
	if len(report.NeedsReconfirmation) != 0 {
		t.Fatalf("needsReconfirmation=%v", report.NeedsReconfirmation)
	}
}

func errorsAsDraft(err error, target **DraftError) bool {
	typed, ok := err.(*DraftError)
	if ok {
		*target = typed
	}
	return ok
}

// TestCRUDTableReconcileNamesChangedRowsAfterAModelChange is the negative case:
// a changed capability effect must move only the occurrences it actually
// affects, and the report must name them instead of reporting everything stale.
func TestCRUDTableReconcileNamesChangedRowsAfterAModelChange(t *testing.T) {
	fixture := crudFixture(t)
	draft, err := Build(Request{Root: fixture.root, DocumentKind: DocumentKindCRUDTable, Application: "order_entry"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	before := fixture.digestOf(t, "databases/sales-core/persistence/order.tei")

	// The row basis covers the mapping identity, table and primary key, so
	// renaming the table moves the derived rows' basis and therefore the matrix
	// content itself, not only the model reference.
	writeFile(t, fixture.root, "databases/sales-core/persistence/order.tei", []byte(`persist sales_core.order {
    database sales_core
    semantic sales.order
    table public.orders_v2
    column sales.order.order_id -> order_id storage varchar required
    column sales.order.customer_id -> customer_id storage varchar required

    primary-key sales.order.order_id
}
`))
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if report.ModelReference.Status != ModelReferenceStale {
		t.Fatalf("model reference=%+v", report.ModelReference)
	}
	statuses := map[string]string{}
	for _, occurrence := range report.Occurrences {
		statuses[occurrence.OccurrenceID] = occurrence.Status
	}
	if statuses["occ-table-1"] != OccurrenceContentChanged {
		t.Fatalf("matrix status=%q want %q", statuses["occ-table-1"], OccurrenceContentChanged)
	}
	if len(report.NeedsReconfirmation) == 0 {
		t.Fatal("a changed model produced nothing to reconfirm")
	}
	if fixture.digestOf(t, "databases/sales-core/persistence/order.tei") == before {
		t.Fatal("fixture did not change; the test proves nothing")
	}
	// The comparison wrote nothing back into the draft or the document.
	again, err := Build(Request{Root: fixture.root, DocumentKind: DocumentKindCRUDTable, Application: "order_entry"})
	if err != nil {
		t.Fatalf("re-Build: %v", err)
	}
	if again.DraftID == draft.DraftID {
		t.Fatal("a changed model produced the same draft identity")
	}
}

// TestCRUDTableReconcileReportsARemovedApplication pins that an application that
// disappears is reported as missing for every occurrence, not as an empty table.
func TestCRUDTableReconcileReportsARemovedApplication(t *testing.T) {
	fixture := crudFixture(t)
	draft, err := Build(Request{Root: fixture.root, DocumentKind: DocumentKindCRUDTable, Application: "order_entry"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	fixture.remove(t, "applications/order-entry/application.tei")
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(report.NeedsReconfirmation) != len(draft.Occurrences) {
		t.Fatalf("needsReconfirmation=%v", report.NeedsReconfirmation)
	}
	for _, occurrence := range report.Occurrences {
		if occurrence.Status != OccurrenceSubjectMissing {
			t.Fatalf("occurrence %s status=%q", occurrence.OccurrenceID, occurrence.Status)
		}
	}
}

// TestCRUDTableReconcileNeverWrites is the read-only boundary of the CRUD path.
func TestCRUDTableReconcileNeverWrites(t *testing.T) {
	fixture := crudFixture(t)
	draft, err := Build(Request{Root: fixture.root, DocumentKind: DocumentKindCRUDTable, Application: "order_entry"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	before := map[string]string{
		"databases/sales-core/persistence/order.tei": fixture.digestOf(t, "databases/sales-core/persistence/order.tei"),
		"applications/order-entry/application.tei":   fixture.digestOf(t, "applications/order-entry/application.tei"),
		"tei.workspace.yaml":                         fixture.digestOf(t, "tei.workspace.yaml"),
	}
	if _, err := Reconcile(draft, fixture.root); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	for relative, want := range before {
		if got := fixture.digestOf(t, relative); got != want {
			t.Fatalf("%s changed: %s -> %s", relative, want, got)
		}
	}
}

// TestCRUDTableReconcileReportsAnOccurrenceTheModelStoppedDeriving covers the
// removal direction. Withdrawing write access removes the write rows from the
// matrix, so the stored matrix keeps content the current projection no longer
// produces. The occurrence itself stays (the matrix is still a matrix), which is
// why this is a content change and not a removed occurrence.
func TestCRUDTableReconcileReportsAnOccurrenceTheModelStoppedDeriving(t *testing.T) {
	fixture := crudFixture(t)
	draft, err := Build(Request{Root: fixture.root, DocumentKind: DocumentKindCRUDTable, Application: "order_entry"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	writeFile(t, fixture.root, "applications/order-entry/application.tei", []byte(`application order_entry {
    uses semantic sales

    database sales {
        target sales_core
        access read
    }
}
`))
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	statuses := reportStatuses(report)
	if statuses["occ-table-1"] != OccurrenceContentChanged {
		t.Fatalf("matrix status=%q want %q", statuses["occ-table-1"], OccurrenceContentChanged)
	}
	if len(report.NeedsReconfirmation) == 0 {
		t.Fatal("a withdrawn write access produced nothing to reconfirm")
	}
}

// TestCRUDTableReconcileReportsARemovedGapTable covers the case where an
// occurrence the stored draft had disappears entirely: withdrawing update intent
// removes the retire gap, so the gap table is no longer derived at all. A row
// that vanished from the model must not survive as content.
func TestCRUDTableReconcileReportsARemovedGapTable(t *testing.T) {
	fixture := crudFixture(t)
	// The fixture starts with update intent, so the surface reports a retire gap
	// and the draft carries a gap table occurrence.
	writeFile(t, fixture.root, "semantic/sales/crud.tei", []byte(`namespace sales

capability sales.order.create {
    input order sales.order
    output sales.order
    effect create sales.order
}

capability sales.order.update {
    input order sales.order
    output sales.order
    effect update sales.order
}
`))
	draft, err := Build(Request{Root: fixture.root, DocumentKind: DocumentKindCRUDTable, Application: "order_entry"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	gapTable, ok := occurrenceByID(draft, "occ-table-2")
	if !ok || gapTable.Content.Table == nil {
		t.Fatalf("fixture must start with a gap table: %+v", draft.Occurrences)
	}
	// Withdrawing the update effect removes the update intent, and with it the
	// retire gap the surface reported.
	writeFile(t, fixture.root, "semantic/sales/crud.tei", []byte(`namespace sales

capability sales.order.create {
    input order sales.order
    output sales.order
    effect create sales.order
}
`))
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	statuses := reportStatuses(report)
	if statuses["occ-table-2"] != OccurrenceRemoved {
		t.Fatalf("gap table status=%q want %q", statuses["occ-table-2"], OccurrenceRemoved)
	}
	found := false
	for _, id := range report.NeedsReconfirmation {
		if id == "occ-table-2" {
			found = true
		}
	}
	if !found {
		t.Fatalf("a removed occurrence is not listed for reconfirmation: %v", report.NeedsReconfirmation)
	}
}

// TestCRUDTableReconcileReportsAnOccurrenceTheModelStartedDeriving covers the
// addition direction: a new fact must appear in the report, not be silently
// merged into the stored list. Declaring an update effect gives the model update
// intent, which makes the surface report a retire gap that did not exist before,
// so the gap table becomes an occurrence the stored draft never had.
func TestCRUDTableReconcileReportsAnOccurrenceTheModelStartedDeriving(t *testing.T) {
	fixture := crudFixture(t)
	draft, err := Build(Request{Root: fixture.root, DocumentKind: DocumentKindCRUDTable, Application: "order_entry"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if _, ok := occurrenceByID(draft, "occ-table-2"); ok {
		t.Fatal("fixture must start without a gap table")
	}
	writeFile(t, fixture.root, "semantic/sales/crud.tei", []byte(`namespace sales

capability sales.order.create {
    input order sales.order
    output sales.order
    effect create sales.order
}

capability sales.order.update {
    input order sales.order
    output sales.order
    effect update sales.order
}
`))
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	added := 0
	for _, occurrence := range report.Occurrences {
		if occurrence.Status == OccurrenceAdded {
			added++
		}
	}
	if added == 0 {
		t.Fatalf("a new derived gap produced no added occurrence: %+v", report.Occurrences)
	}
	if statuses := reportStatuses(report); statuses["occ-table-1"] != OccurrenceContentChanged {
		t.Fatalf("matrix status=%q want %q", statuses["occ-table-1"], OccurrenceContentChanged)
	}
}

func reportStatuses(report ReconcileReport) map[string]string {
	statuses := map[string]string{}
	for _, occurrence := range report.Occurrences {
		statuses[occurrence.OccurrenceID] = occurrence.Status
	}
	return statuses
}

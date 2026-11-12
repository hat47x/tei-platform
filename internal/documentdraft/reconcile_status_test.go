package documentdraft

import (
	"testing"

	"tei/internal/sourceindex"
)

// The reconcile path answers four more questions than "did the content change".
// Each one is a different reason a stored occurrence cannot be trusted as-is, and
// a reader must be able to tell them apart.

// TestReconcileReportsASubjectAmbiguous pins that one identity declared twice in
// the same namespace is reported as ambiguous, not silently matched to whichever
// declaration the read happened to return first. A name is never used to decide
// identity, and two declarations of one identity is exactly that case.
func TestReconcileReportsASubjectAmbiguous(t *testing.T) {
	fixture := newFixture(t)
	// A second file in the SAME namespace declares the same data again, so the
	// identity `sales.order` now has two declarations.
	writeFile(t, fixture.root, "semantic/sales/order-extra.tei", []byte(`namespace sales

data Order {
    attribute otherId type std.text
}
`))
	// The draft is about a different, still-unambiguous subject, so the Workspace
	// read succeeds and the ambiguity is reported per occurrence.
	customer := sourceindex.Subject{Kind: "semantic-data", ID: "customer.customer"}
	draft, err := Build(Request{Root: fixture.root, Subject: customer})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	tampered := draft
	tampered.Occurrences = append([]Occurrence(nil), draft.Occurrences...)
	tampered.Occurrences[0].SubjectRefs = []sourceindex.Subject{orderSubject()}
	report, err := Reconcile(tampered, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	first := report.Occurrences[0]
	if first.Status != OccurrenceSubjectAmbiguous {
		t.Fatalf("status=%q want %q", first.Status, OccurrenceSubjectAmbiguous)
	}
}

// TestReconcileReportsAnAmbiguousDraftSubject pins the other end of the same
// problem: when the draft's own subject stopped being unambiguous, the whole
// comparison is refused rather than comparing against one of the two
// declarations. The reader is told the identity moved, not which one won.
func TestReconcileReportsAnAmbiguousDraftSubject(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject()})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	writeFile(t, fixture.root, "semantic/sales/order-extra.tei", []byte(`namespace sales

data Order {
    attribute otherId type std.text
}
`))
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if report.ModelReference.Status != ModelReferenceStale {
		t.Fatalf("model reference=%+v", report.ModelReference)
	}
	found := false
	for _, diff := range report.ModelReference.Diffs {
		if diff == "subject-ambiguous" {
			found = true
		}
	}
	if !found {
		t.Fatalf("diffs=%v do not name the ambiguity", report.ModelReference.Diffs)
	}
	// Every occurrence needs reconfirmation: none of them can be compared.
	if len(report.NeedsReconfirmation) != len(draft.Occurrences) {
		t.Fatalf("needsReconfirmation=%v of %d occurrences",
			report.NeedsReconfirmation, len(draft.Occurrences))
	}
}

// TestReconcileRefusesAnUnsupportedSubjectKind pins that a subject kind this
// version does not project is reported as such, rather than being read as a
// missing subject or compared as if it were one of the supported kinds.
func TestReconcileRefusesAnUnsupportedSubjectKind(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject()})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	tampered := draft
	tampered.Occurrences = append([]Occurrence(nil), draft.Occurrences...)
	// `application` is a real subject kind in the Workspace but not one an
	// element draft projects, so the occurrence is not comparable.
	tampered.Occurrences[0].SubjectRefs = []sourceindex.Subject{
		{Kind: "application", ID: "order_entry"},
	}
	report, err := Reconcile(tampered, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	first := report.Occurrences[0]
	if first.Status != OccurrenceUnsupportedKind {
		t.Fatalf("status=%q want %q", first.Status, OccurrenceUnsupportedKind)
	}
}

// TestReconcileNamesItsOwnSchemaVersion pins the report's identity: a report that
// does not name the contract it answers from cannot be compared later.
func TestReconcileNamesItsOwnSchemaVersion(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject()})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if report.SchemaVersion != "tei.document-draft.reconcile/v0" {
		t.Fatalf("schemaVersion=%q", report.SchemaVersion)
	}
	if report.Effect != EffectReadOnlyProjection {
		t.Fatalf("effect=%q", report.Effect)
	}
}

// TestDiagramOccurrenceIsAClosedKind pins that `diagram` is one of the kinds this
// version can name. Stage 3 (the three-element diagram) is not implemented yet, so
// a stored diagram occurrence is reported as an unknown rule rather than silently
// treated as an explanation: the kind exists, the generator does not.
func TestDiagramOccurrenceIsAClosedKind(t *testing.T) {
	if OccurrenceDiagram != "diagram" {
		t.Fatalf("diagram kind = %q", OccurrenceDiagram)
	}
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject()})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	tampered := draft
	tampered.Occurrences = append([]Occurrence(nil), draft.Occurrences...)
	tampered.Occurrences[0].Kind = OccurrenceDiagram
	tampered.Occurrences[0].GenerationConditions.Rule = "diagram-v0"
	report, err := Reconcile(tampered, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	first := report.Occurrences[0]
	if first.Status != OccurrenceUnknownRule {
		t.Fatalf("status=%q want %q (an unimplemented generator is not a passing one)",
			first.Status, OccurrenceUnknownRule)
	}
}
